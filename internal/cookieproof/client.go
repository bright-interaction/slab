// Package cookieproof talks to the CookieProof consent service to provision
// per-site consent banners + allowed origins. Atomicsite calls EnsureDomain at
// build time so newly built sites can fetch their config + submit consent
// proofs without a 403.
package cookieproof

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpTimeout caps a single CookieProof HTTP call. Build time is best-effort,
// so we'd rather time out and ship the build than block forever on a flaky
// upstream.
const httpTimeout = 15 * time.Second

// Client provisions CookieProof state for a domain. Configured once per
// process from cfg.CookieProofAPIBase + cfg.CookieProofAdminToken; pass the
// same instance to as many builds as you want.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// New returns a configured Client. baseURL should be e.g.
// "https://consent.example.com" (no trailing slash). authToken is a
// CookieProof API key (env API_KEY or a row in api_keys); we send it as
// "Authorization: Bearer <token>" because that is what CookieProof's
// getAuthContext accepts.
func New(baseURL, authToken string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		authToken:  authToken,
		httpClient: &http.Client{Timeout: httpTimeout},
	}
}

// SetHTTPClient overrides the HTTP client. Used by tests to inject httptest
// servers.
func (c *Client) SetHTTPClient(hc *http.Client) {
	if hc != nil {
		c.httpClient = hc
	}
}

// EnsureDomain provisions everything CookieProof needs so a site at `domain`
// can load its consent banner config and submit consent proofs:
//
//  1. Upserts a domain_configs row via PUT /api/config. We pass a minimal
//     default config; users can override per-domain via the CookieProof UI.
//  2. Adds https://{domain} to allowed_origins via POST /api/settings/domains
//     so /api/proof submissions from the site don't hit a 403. Idempotent:
//     CookieProof returns 409 if the origin already exists, which we treat
//     as success.
//
// On a misconfigured client (no authToken, no baseURL) this returns nil
// without making any calls so build pipelines continue working in dev. The
// caller already logs a warning at the call site.
//
// IMPORTANT: this assumes the standard public CookieProof admin API surface
// observed in CookieProof/api/server.ts. Endpoints are PUT /api/config and
// POST /api/settings/domains. The original Phase 10 brief mentioned
// `/api/admin/domains` and `/api/admin/orgs/{org_id}/allowed-origins`, but
// those routes do not exist in the open-source build (the EE-only /api/admin
// surface is users/orgs/billing, not domain provisioning). Documented here
// so future spec drift doesn't surprise anyone.
func (c *Client) EnsureDomain(ctx context.Context, domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return errors.New("cookieproof: domain is required")
	}
	if c.baseURL == "" || c.authToken == "" {
		return nil // not configured; caller logs and continues
	}

	if err := c.putConfig(ctx, domain); err != nil {
		return fmt.Errorf("cookieproof: put config: %w", err)
	}
	if err := c.postAllowedOrigin(ctx, "https://"+domain); err != nil {
		return fmt.Errorf("cookieproof: post allowed origin: %w", err)
	}
	return nil
}

// putConfigRequest mirrors the body PUT /api/config expects.
type putConfigRequest struct {
	Domain string         `json:"domain"`
	Config map[string]any `json:"config"`
}

// putConfig upserts the domain_configs row. CookieProof's ownership check
// silently no-ops when an org-scoped key updates an existing row owned by
// the same org, and rejects with 403 when an unscoped (env) key tries to
// overwrite an org-owned row -- we surface that error to the caller.
func (c *Client) putConfig(ctx context.Context, domain string) error {
	body := putConfigRequest{
		Domain: domain,
		Config: map[string]any{
			// Minimal default config. Real banner copy + categories come from
			// the CookieProof UI. We just need a row so /api/config/{domain}
			// returns 200 instead of 404.
			"categories": []any{},
			"locale":     "en",
		},
	}
	return c.doJSON(ctx, http.MethodPut, "/api/config", body)
}

// originRequest mirrors POST /api/settings/domains.
type originRequest struct {
	Origin string `json:"origin"`
}

// postAllowedOrigin adds an origin to allowed_domains. 409 means the origin
// already exists -- treat as success.
func (c *Client) postAllowedOrigin(ctx context.Context, origin string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/settings/domains", originRequest{Origin: origin})
}

// doJSON is the shared transport helper. 2xx + 409 -> nil; everything else
// returns an error including the response body for debugging. We bound the
// read at 4 KiB because CookieProof error bodies are tiny JSON objects.
func (c *Client) doJSON(ctx context.Context, method, path string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
}
