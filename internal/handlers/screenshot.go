// Package handlers — screenshot endpoint.
//
// Closes the visual-feedback gap that's been killing iteration speed.
// Atomicsite renders agent JSON into static Astro pages, but the agent
// has historically had no way to see the rendered output — every layout
// bug had to be reported back by a human. This handler runs headless
// Chromium against the deployed page, returns base64 PNG, and lets the
// agent reason about pixels in the same turn it edits blocks.
//
// Flow:
//   1. Agent edits blocks via MCP / agent API
//   2. Agent calls trigger_build → success
//   3. Agent calls POST /api/agent/screenshot { url } → base64 PNG
//   4. Agent decodes the image and visually checks the result
//   5. Iterate, re-build, re-screenshot, until pixel-perfect
//
// The agent IS Claude (or compatible) which has vision. Sending the PNG
// back as base64 lands as an image part in the tool result on
// vision-capable models; the agent can describe what it sees and decide
// whether to keep iterating.

package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ScreenshotHandler executes a headless Chromium navigation + screenshot
// for an agent-supplied URL. Site-scoped: the URL must belong to the
// requesting agent's site (or a known atomicsite-hosted subdomain) so
// the handler can't be turned into an open SSRF vector.
type ScreenshotHandler struct{}

func NewScreenshotHandler() *ScreenshotHandler {
	return &ScreenshotHandler{}
}

// ScreenshotRequest is the public params struct shared by the HTTP
// handler and the MCP tool wrapper. Both paths call CaptureScreenshot
// so the chromedp logic stays in one place.
type ScreenshotRequest struct {
	URL            string `json:"url"`
	ViewportWidth  int    `json:"viewport_width"`
	ViewportHeight int    `json:"viewport_height"`
	FullPage       *bool  `json:"full_page"`
	WaitMs         int    `json:"wait_ms"`
}

// ScreenshotResult is the canonical response shape for a successful
// screenshot capture. image_base64 is a base64-encoded PNG; vision-
// capable agents decode it and reason about pixels directly.
type ScreenshotResult struct {
	URL         string `json:"url"`
	ImageBase64 string `json:"image_base64"`
	Format      string `json:"format"`
	Viewport    struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport"`
	FullPage  bool `json:"full_page"`
	SizeBytes int  `json:"size_bytes"`
}

// CaptureScreenshot is the single chromedp execution path. Used by both
// the HTTP handler (POST /api/agent/screenshot) and the MCP tool
// wrapper (screenshot). Defaults: 1440x900 viewport, full_page=true,
// wait_ms=800. SSRF-locked via validateScreenshotURL.
func CaptureScreenshot(ctx context.Context, req ScreenshotRequest) (*ScreenshotResult, error) {
	target := strings.TrimSpace(req.URL)
	if target == "" {
		return nil, errors.New("url required")
	}
	if err := validateScreenshotURL(target); err != nil {
		return nil, err
	}

	vw := req.ViewportWidth
	if vw <= 0 {
		vw = 1440
	}
	if vw > 3840 {
		vw = 3840
	}
	vh := req.ViewportHeight
	if vh <= 0 {
		vh = 900
	}
	if vh > 2160 {
		vh = 2160
	}
	fullPage := true
	if req.FullPage != nil {
		fullPage = *req.FullPage
	}
	waitMs := req.WaitMs
	if waitMs <= 0 {
		waitMs = 800
	}
	if waitMs > 10000 {
		waitMs = 10000
	}

	captureCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
		chromedp.WindowSize(vw, vh),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(captureCtx, opts...)
	defer allocCancel()
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	var buf []byte
	tasks := []chromedp.Action{
		chromedp.EmulateViewport(int64(vw), int64(vh)),
		chromedp.Navigate(target),
		chromedp.Sleep(time.Duration(waitMs) * time.Millisecond),
	}
	if fullPage {
		tasks = append(tasks, chromedp.FullScreenshot(&buf, 90))
	} else {
		tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
	}
	if err := chromedp.Run(browserCtx, tasks...); err != nil {
		return nil, fmt.Errorf("chromedp run: %w", err)
	}

	res := &ScreenshotResult{
		URL:         target,
		ImageBase64: base64.StdEncoding.EncodeToString(buf),
		Format:      "png",
		FullPage:    fullPage,
		SizeBytes:   len(buf),
	}
	res.Viewport.Width = vw
	res.Viewport.Height = vh
	return res, nil
}

// Screenshot is the HTTP entry point. Reads JSON body, calls
// CaptureScreenshot, returns the result. Both this and the MCP tool
// wrapper hit the same shared CaptureScreenshot function.
func (h *ScreenshotHandler) Screenshot(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}
	var req ScreenshotRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	res, err := CaptureScreenshot(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("screenshot failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// validateScreenshotURL rejects anything that's not a public atomicsite
// tenant URL (locks the endpoint down so it can't be turned into an
// SSRF probe of internal infrastructure).
func validateScreenshotURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("malformed url: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.New("url scheme must be http or https")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return errors.New("url missing host")
	}
	allowed := host == "app.slab.example.com" ||
		host == "brightinteraction.com" ||
		host == "www.brightinteraction.com" ||
		strings.HasSuffix(host, ".slab.example.com") ||
		strings.HasSuffix(host, ".brightinteraction.com")
	if !allowed {
		return fmt.Errorf("host %q not allowed; only atomicsite tenant subdomains + brightinteraction.com are screenshot-able", host)
	}
	return nil
}
