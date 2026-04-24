package handlers

import (
	"net/http"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// CSSClassHandler handles global CSS class CRUD endpoints.
type CSSClassHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewCSSClassHandler(cfg *config.Config, queries *store.Queries) *CSSClassHandler {
	return &CSSClassHandler{cfg: cfg, queries: queries}
}

func (h *CSSClassHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	classes, err := h.queries.ListCSSClassesBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list CSS classes")
		return
	}
	writeJSON(w, http.StatusOK, classes)
}

func (h *CSSClassHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "classID")
	cc, err := h.queries.GetCSSClassByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "CSS class not found")
		return
	}
	writeJSON(w, http.StatusOK, cc)
}

func (h *CSSClassHandler) Create(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	var req struct {
		Name      string `json:"name"`
		Category  string `json:"category"`
		CSS       string `json:"css"`
		UsageNote string `json:"usage_note"`
		SortOrder int64  `json:"sort_order"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Name == "" || req.CSS == "" {
		writeError(w, http.StatusBadRequest, "name and css are required")
		return
	}
	if req.Category == "" {
		req.Category = "utility"
	}

	id := newID()
	err := h.queries.CreateCSSClass(r.Context(), store.CreateCSSClassParams{
		ID:        id,
		SiteID:    siteID,
		Name:      req.Name,
		Category:  req.Category,
		Css:       req.CSS,
		UsageNote: req.UsageNote,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create CSS class")
		return
	}

	cc, _ := h.queries.GetCSSClassByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, cc)
}

func (h *CSSClassHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "classID")
	existing, err := h.queries.GetCSSClassByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "CSS class not found")
		return
	}

	var req struct {
		Name      *string `json:"name"`
		Category  *string `json:"category"`
		CSS       *string `json:"css"`
		UsageNote *string `json:"usage_note"`
		SortOrder *int64  `json:"sort_order"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	cat := existing.Category
	if req.Category != nil {
		cat = *req.Category
	}
	css := existing.Css
	if req.CSS != nil {
		css = *req.CSS
	}
	usageNote := existing.UsageNote
	if req.UsageNote != nil {
		usageNote = *req.UsageNote
	}
	sortOrder := existing.SortOrder
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	err = h.queries.UpdateCSSClass(r.Context(), store.UpdateCSSClassParams{
		ID:        id,
		Name:      name,
		Category:  cat,
		Css:       css,
		UsageNote: usageNote,
		SortOrder: sortOrder,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update CSS class")
		return
	}

	cc, _ := h.queries.GetCSSClassByID(r.Context(), id)
	writeJSON(w, http.StatusOK, cc)
}

func (h *CSSClassHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "classID")
	if err := h.queries.DeleteCSSClass(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete CSS class")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
