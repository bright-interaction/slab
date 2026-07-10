// Package handlers provides HTTP handler implementations.
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bright-interaction/slab/internal/store"
)

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json", "error", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// parseJSON decodes a JSON request body into v.
func parseJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// urlParam extracts a chi URL parameter.
func urlParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// newID generates a random 12-byte hex ID.
func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// pageInSite verifies the page exists and belongs to the given site. Returns
// the page on success. On miss it writes a 404 to w and returns ok=false so
// the caller can return without leaking which IDs exist in other tenants.
//
// Cross-tenant defence: SiteAccessMiddleware gates the URL's siteID, but a
// member of site A who knows a pageID belonging to site B can still hit
// /api/sites/{siteA}/pages/{siteB_pageID} unless every handler verifies the
// page actually belongs to the URL's siteID.
func pageInSite(ctx context.Context, queries *store.Queries, w http.ResponseWriter, pageID, siteID string) (store.Page, bool) {
	page, err := queries.GetPageByID(ctx, pageID)
	if err != nil || page.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Page not found")
		return store.Page{}, false
	}
	return page, true
}

// blockInSite verifies the block exists and its parent page belongs to the
// given site. Mirrors pageInSite for the /api/sites/{siteID}/.../blocks/{blockID}
// surface. On miss it writes 404 and returns ok=false.
func blockInSite(ctx context.Context, queries *store.Queries, w http.ResponseWriter, blockID, siteID string) (store.Block, bool) {
	block, err := queries.GetBlockByID(ctx, blockID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Block not found")
		return store.Block{}, false
	}
	page, err := queries.GetPageByID(ctx, block.PageID)
	if err != nil || page.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Block not found")
		return store.Block{}, false
	}
	return block, true
}
