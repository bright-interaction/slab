package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/imaging"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/storage"
	"github.com/bright-interaction/slab/internal/store"
)

// MediaHandler handles media CRUD + upload + public serving.
type MediaHandler struct {
	cfg     *config.Config
	queries *store.Queries
	store   storage.Store
}

func NewMediaHandler(cfg *config.Config, queries *store.Queries, st storage.Store) *MediaHandler {
	return &MediaHandler{cfg: cfg, queries: queries, store: st}
}

// Extensions rejected outright (mirrors brightcrm's list, adapted for image-only).
var dangerousExts = map[string]bool{
	".html": true, ".xhtml": true, ".htm": true,
	".js": true, ".mjs": true, ".php": true, ".phtml": true,
	".svg":  true, // XSS risk without sanitization
	".exe":  true, ".dll": true, ".sh": true, ".bat": true,
	".jsp":  true, ".asp": true, ".aspx": true,
}

// ---------------- Admin endpoints ----------------

// List returns paginated media for a site.
func (h *MediaHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	rows, err := h.queries.ListMediaBySitePaginated(r.Context(), store.ListMediaBySitePaginatedParams{
		SiteID: siteID,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list media")
		return
	}
	count, _ := h.queries.CountMediaBySite(r.Context(), siteID)

	writeJSON(w, http.StatusOK, map[string]any{
		"items":  rows,
		"total":  count,
		"limit":  limit,
		"offset": offset,
	})
}

// Get returns one media row.
func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "mediaID")
	m, err := h.queries.GetMediaByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Media not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// Update edits the alt_text.
func (h *MediaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "mediaID")
	_, err := h.queries.GetMediaByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Media not found")
		return
	}
	var req struct {
		AltText string `json:"alt_text"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.queries.UpdateMedia(r.Context(), store.UpdateMediaParams{
		AltText: req.AltText,
		ID:      id,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update media")
		return
	}
	m, _ := h.queries.GetMediaByID(r.Context(), id)
	writeJSON(w, http.StatusOK, m)
}

// Delete removes media from DB and disk.
func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "mediaID")
	m, err := h.queries.GetMediaByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Media not found")
		return
	}
	// Wipe entire {siteID}/{mediaID}/ dir on disk
	prefix := filepath.Join(m.SiteID, id)
	if err := h.store.DeleteDir(r.Context(), prefix); err != nil {
		slog.Warn("delete media dir failed", "prefix", prefix, "err", err)
	}
	if err := h.queries.DeleteMedia(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete media")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Upload handles multipart upload from admin UI.
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	h.uploadMultipart(w, r, siteID)
}

// ---------------- Agent endpoints ----------------

// AgentUpload: multipart upload via agent API key.
func (h *MediaHandler) AgentUpload(w http.ResponseWriter, r *http.Request) {
	a := authmw.GetAgent(r)
	if a == nil {
		writeError(w, http.StatusUnauthorized, "Agent not authenticated")
		return
	}
	h.uploadMultipart(w, r, a.SiteID)
}

// AgentUploadFromBase64: for AI agents that can emit base64 but not multipart.
func (h *MediaHandler) AgentUploadFromBase64(w http.ResponseWriter, r *http.Request) {
	a := authmw.GetAgent(r)
	if a == nil {
		writeError(w, http.StatusUnauthorized, "Agent not authenticated")
		return
	}
	var req struct {
		Data     string `json:"data"`
		MimeType string `json:"mime_type"`
		Filename string `json:"filename"`
		AltText  string `json:"alt_text"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Data == "" || req.Filename == "" {
		writeError(w, http.StatusBadRequest, "data and filename are required")
		return
	}
	// Accept both "data:image/jpeg;base64,..." and raw base64
	data := req.Data
	if idx := strings.Index(data, "base64,"); idx >= 0 {
		data = data[idx+len("base64,"):]
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid base64 data")
		return
	}
	if int64(len(raw)) > h.cfg.MaxUploadSize {
		writeError(w, http.StatusRequestEntityTooLarge, "File too large")
		return
	}
	media, status, msg := h.processAndSave(r.Context(), a.SiteID, req.Filename, req.AltText, raw)
	if status != http.StatusCreated {
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusCreated, media)
}

// AgentUploadFromURL: server fetches a URL and processes. SSRF-guarded.
func (h *MediaHandler) AgentUploadFromURL(w http.ResponseWriter, r *http.Request) {
	a := authmw.GetAgent(r)
	if a == nil {
		writeError(w, http.StatusUnauthorized, "Agent not authenticated")
		return
	}
	var req struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
		AltText  string `json:"alt_text"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	raw, finalFilename, err := fetchWithSSRFGuard(r.Context(), req.URL, h.cfg.MaxUploadSize)
	if err != nil {
		slog.Warn("agent upload from-url rejected", "url", req.URL, "err", err)
		writeError(w, http.StatusBadRequest, "Failed to fetch URL (rejected or unreachable)")
		return
	}
	filename := req.Filename
	if filename == "" {
		filename = finalFilename
	}
	if filename == "" {
		filename = "image"
	}
	media, status, msg := h.processAndSave(r.Context(), a.SiteID, filename, req.AltText, raw)
	if status != http.StatusCreated {
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusCreated, media)
}

// AgentUpdate: edit alt text.
func (h *MediaHandler) AgentUpdate(w http.ResponseWriter, r *http.Request) {
	a := authmw.GetAgent(r)
	if a == nil {
		writeError(w, http.StatusUnauthorized, "Agent not authenticated")
		return
	}
	id := urlParam(r, "mediaID")
	m, err := h.queries.GetMediaByID(r.Context(), id)
	if err != nil || m.SiteID != a.SiteID {
		writeError(w, http.StatusNotFound, "Media not found")
		return
	}
	var req struct {
		AltText string `json:"alt_text"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.queries.UpdateMedia(r.Context(), store.UpdateMediaParams{
		AltText: req.AltText,
		ID:      id,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update media")
		return
	}
	m, _ = h.queries.GetMediaByID(r.Context(), id)
	writeJSON(w, http.StatusOK, m)
}

// AgentDelete: delete media owned by the agent's site.
func (h *MediaHandler) AgentDelete(w http.ResponseWriter, r *http.Request) {
	a := authmw.GetAgent(r)
	if a == nil {
		writeError(w, http.StatusUnauthorized, "Agent not authenticated")
		return
	}
	id := urlParam(r, "mediaID")
	m, err := h.queries.GetMediaByID(r.Context(), id)
	if err != nil || m.SiteID != a.SiteID {
		writeError(w, http.StatusNotFound, "Media not found")
		return
	}
	prefix := filepath.Join(m.SiteID, id)
	if err := h.store.DeleteDir(r.Context(), prefix); err != nil {
		slog.Warn("delete media dir failed", "prefix", prefix, "err", err)
	}
	if err := h.queries.DeleteMedia(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete media")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---------------- Public file server ----------------

// ServePublic serves processed media files at /media/{siteID}/{mediaID}/{variant}.{ext}.
// Long-cache headers are safe because filenames are content-addressed.
func (h *MediaHandler) ServePublic(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	if key == "" || strings.Contains(key, "..") {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	reader, err := h.store.Get(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	defer reader.Close()

	// Content-Type from extension
	ext := strings.ToLower(filepath.Ext(key))
	ct := extMimeMap[ext]
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, reader)
}

var extMimeMap = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
	".avif": "image/avif",
}

// ---------------- Internal helpers ----------------

// uploadMultipart handles the multipart form path (used by both admin and agent).
func (h *MediaHandler) uploadMultipart(w http.ResponseWriter, r *http.Request, siteID string) {
	// Enforce max size at the reader level
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxUploadSize)

	if err := r.ParseMultipartForm(h.cfg.MaxUploadSize); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "File too large or invalid form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing file field")
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read upload")
		return
	}
	if int64(len(raw)) > h.cfg.MaxUploadSize {
		writeError(w, http.StatusRequestEntityTooLarge, "File too large")
		return
	}

	altText := r.FormValue("alt_text")
	filename := header.Filename
	if filename == "" {
		filename = "image"
	}

	media, status, msg := h.processAndSave(r.Context(), siteID, filename, altText, raw)
	if status != http.StatusCreated {
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusCreated, media)
}

// processAndSave validates the raw bytes, runs the imaging pipeline, and writes
// a media row. Returns the created row + HTTP status + error message on failure.
func (h *MediaHandler) processAndSave(ctx context.Context, siteID, filename, altText string, raw []byte) (*store.Medium, int, string) {
	// 1. Sniff MIME and validate
	mime := imaging.Sniff(raw)
	format := imaging.FormatFromMime(mime)
	if format == "" {
		return nil, http.StatusUnsupportedMediaType, "Unsupported image format (must be JPEG/PNG/GIF/WebP)"
	}

	// 2. Filename sanitation + extension blocklist
	safeName := filepath.Base(filename)
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	ext := strings.ToLower(filepath.Ext(safeName))
	if dangerousExts[ext] {
		return nil, http.StatusBadRequest, "File extension not allowed"
	}

	// 3. Generate media ID + run pipeline
	mediaID := newID()
	result, err := imaging.Process(ctx, raw, h.store, siteID, mediaID, format, h.cfg.MediaVariants)
	if err != nil {
		slog.Error("media pipeline failed", "err", err)
		return nil, http.StatusInternalServerError, "Image processing failed"
	}

	// 4. Find original path from variants
	var originalPath string
	for _, v := range result.Variants {
		if strings.Contains(v.Path, "/original.") {
			originalPath = v.Path
			break
		}
	}

	// 5. Persist
	if err := h.queries.CreateMedia(ctx, store.CreateMediaParams{
		ID:           mediaID,
		SiteID:       siteID,
		Filename:     safeName,
		AltText:      altText,
		MimeType:     mime,
		FileSize:     int64(len(raw)),
		Width:        int64(result.OriginalWidth),
		Height:       int64(result.OriginalHeight),
		OriginalPath: originalPath,
	}); err != nil {
		return nil, http.StatusInternalServerError, "Failed to save media record"
	}
	if err := h.queries.UpdateMediaVariants(ctx, store.UpdateMediaVariantsParams{
		Width:        int64(result.OriginalWidth),
		Height:       int64(result.OriginalHeight),
		Blurhash:     result.Blurhash,
		VariantsJson: marshalField(result.Variants),
		ID:           mediaID,
	}); err != nil {
		return nil, http.StatusInternalServerError, "Failed to save variants"
	}

	m, _ := h.queries.GetMediaByID(ctx, mediaID)
	return &m, http.StatusCreated, ""
}

// fetchWithSSRFGuard downloads a URL with strict SSRF protection.
// Blocks private IP ranges, limits body size, follows up to 3 redirects re-checking each.
func fetchWithSSRFGuard(ctx context.Context, rawURL string, maxSize int64) ([]byte, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, "", fmt.Errorf("only http/https URLs allowed")
	}

	// DNS resolve + block private ranges
	host := u.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, "", fmt.Errorf("DNS lookup failed: %s", host)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return nil, "", fmt.Errorf("private address blocked: %s", ip)
		}
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			// Re-check each redirect target for SSRF
			ips, err := net.LookupIP(req.URL.Hostname())
			if err != nil {
				return err
			}
			for _, ip := range ips {
				if isPrivateIP(ip) {
					return fmt.Errorf("redirect to private address blocked")
				}
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("fetch returned status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxSize+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", err
	}
	if int64(len(raw)) > maxSize {
		return nil, "", fmt.Errorf("response exceeds max size")
	}

	// Extract filename from URL path
	filename := filepath.Base(u.Path)
	if filename == "" || filename == "/" || filename == "." {
		filename = "image"
	}
	return raw, filename, nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// IsPrivate covers RFC1918 and IPv6 ULA
	if ip.IsPrivate() {
		return true
	}
	return false
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// Unused but keep for future: bytes.Reader for io-based processing.
var _ = bytes.NewReader
