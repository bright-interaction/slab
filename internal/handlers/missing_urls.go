// Package handlers - missing_urls.go (Layer 5 / Sprint 2 of the migration
// system, 2026-05-06).
//
// Routes:
//
//	GET    /api/sites/{siteID}/missing-urls?limit=50         top 404 paths
//	POST   /api/sites/{siteID}/missing-urls/{id}/redirect    one-click create
//	DELETE /api/sites/{siteID}/missing-urls/{id}             dismiss without redirect
//
// The analytics parser populates `missing_urls` in real time when a 404
// hits the access log. This handler surfaces those rows so the operator
// (or the agent) can convert ongoing organic 404 traffic into recovered
// link juice with one click. None of the competing CMSes do this.
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
)

const missingURLsDefaultLimit = 50
const missingURLsMaxLimit = 500

// MissingURLsHandler exposes the per-site 404 inbox.
type MissingURLsHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

// NewMissingURLsHandler wires the handler against the open store.
func NewMissingURLsHandler(cfg *config.Config, queries *store.Queries) *MissingURLsHandler {
	return &MissingURLsHandler{cfg: cfg, queries: queries}
}

// List returns top-N missing URLs ordered by hit_count desc, last_seen
// desc. The default limit is 50; max is 500. Empty result is fine.
func (h *MissingURLsHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	limit := missingURLsDefaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > missingURLsMaxLimit {
				n = missingURLsMaxLimit
			}
			limit = n
		}
	}
	rows, err := h.queries.ListMissingURLsBySite(r.Context(), store.ListMissingURLsBySiteParams{
		SiteID: siteID, Limit: int64(limit),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "List missing URLs")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, m := range rows {
		out = append(out, map[string]any{
			"id":         m.ID,
			"path":       m.Path,
			"referer":    m.Referer,
			"hit_count":  m.HitCount,
			"first_seen": m.FirstSeen,
			"last_seen":  m.LastSeen,
		})
	}
	total, _ := h.queries.CountMissingURLsBySite(r.Context(), siteID)
	writeJSON(w, http.StatusOK, map[string]any{
		"items": out,
		"total": total,
	})
}

// missingURLToRedirectRequest is the input for the one-click "convert
// 404 into 301" endpoint. Only ToPath is required.
type missingURLToRedirectRequest struct {
	ToPath string `json:"to_path"`
	// StatusCode lets the operator choose 301 vs 410 ("gone"). 410 still
	// removes the missing-url row but creates a 410 redirect rule so
	// Google drops the URL cleanly. Default 301.
	StatusCode int64 `json:"status_code,omitempty"`
}

// CreateRedirect converts a missing-URL row into a permanent 301 (or 410)
// rule, then deletes the missing-URL entry. The two writes are not in a
// SQLite transaction because that would require pulling the *sql.DB into
// every handler; instead we accept the (very small) risk of a redirect
// row landing without the missing-URL row clearing, which is
// idempotent-recoverable (dismiss the row manually).
//
// Cross-tenant: re-fetches the missing-URL row by ID and matches site_id
// before writing anything.
func (h *MissingURLsHandler) CreateRedirect(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "missingID")
	row, err := h.queries.GetMissingURLByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Missing URL not found")
		return
	}

	var req missingURLToRedirectRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	to := strings.TrimSpace(req.ToPath)
	if to == "" {
		writeError(w, http.StatusBadRequest, "to_path is required")
		return
	}
	statusCode := req.StatusCode
	if statusCode == 0 {
		statusCode = 301
	}
	if !validRedirectStatusCodes[statusCode] {
		writeError(w, http.StatusUnprocessableEntity,
			"status_code must be one of 301, 302, 307, 308, 410")
		return
	}
	if !strings.HasPrefix(to, "/") {
		writeError(w, http.StatusUnprocessableEntity,
			"to_path must start with / (off-site redirects not supported)")
		return
	}
	if strings.HasPrefix(to, "//") {
		writeError(w, http.StatusUnprocessableEntity,
			"to_path cannot start with // (open-redirect vector)")
		return
	}
	if !pathTokenSafeForHandler(to) {
		writeError(w, http.StatusUnprocessableEntity,
			"to_path contains forbidden characters")
		return
	}
	if to == row.Path {
		writeError(w, http.StatusUnprocessableEntity,
			"to_path equals from_path; no-op redirect rejected")
		return
	}

	// Conflict guard: if a redirect already exists at this from_path,
	// surface it instead of silently overwriting. Common when the
	// operator clicked twice or the same 404 was already addressed.
	if existing, err := h.queries.GetRedirectByPath(r.Context(), store.GetRedirectByPathParams{
		SiteID: siteID, FromPath: row.Path,
	}); err == nil && existing.ID != "" {
		// Still clear the missing-URL row since the redirect is in place.
		_ = h.queries.DeleteMissingURL(r.Context(), id)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "already_existed",
			"redirect_id": existing.ID,
			"from":        existing.FromPath,
			"to":          existing.ToPath,
		})
		return
	}

	rid := newID()
	if err := h.queries.CreateRedirect(r.Context(), store.CreateRedirectParams{
		ID:         rid,
		SiteID:     siteID,
		FromPath:   row.Path,
		ToPath:     to,
		StatusCode: statusCode,
		IsAuto:     0, // operator-initiated; not slug-edit auto
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Create redirect")
		return
	}
	_ = h.queries.DeleteMissingURL(r.Context(), id)
	AuditLog(r.Context(), h.queries, r, siteID, AuditActionCreate, "missing_url_redirect", rid, map[string]any{
		"from": row.Path, "to": to, "status": statusCode,
		"hit_count_at_conversion": row.HitCount,
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":      "created",
		"redirect_id": rid,
		"from":        row.Path,
		"to":          to,
		"status_code": statusCode,
	})
}

// Dismiss removes a missing-URL row without creating a redirect. Useful
// for known-junk paths (bot scans, accidental hits) the operator never
// wants surfaced again. The row will reappear if the path 404s again.
// To make a dismissal stick permanently the operator should create a 410
// redirect via CreateRedirect with status_code=410.
func (h *MissingURLsHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "missingID")
	row, err := h.queries.GetMissingURLByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Missing URL not found")
		return
	}
	if err := h.queries.DeleteMissingURL(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Dismiss missing URL")
		return
	}
	AuditLog(r.Context(), h.queries, r, siteID, AuditActionDelete, "missing_url", id, map[string]any{
		"path": row.Path, "hit_count": row.HitCount,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

// notFoundError is exported so the migration handler can re-use the same
// 404 surface when needed (e.g. when verify-coverage references a deleted
// migration). Currently unused but reserved for future composition.
var notFoundError = errors.New("not found")

func init() { _ = notFoundError }
