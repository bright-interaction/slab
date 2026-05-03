package handlers

import (
	"net/http"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// KnowledgebaseHandler handles knowledgebase CRUD endpoints.
type KnowledgebaseHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewKnowledgebaseHandler(cfg *config.Config, queries *store.Queries) *KnowledgebaseHandler {
	return &KnowledgebaseHandler{cfg: cfg, queries: queries}
}

func (h *KnowledgebaseHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	entries, err := h.queries.ListKnowledgebaseBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list knowledgebase entries")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *KnowledgebaseHandler) Get(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "entryID")
	entry, err := h.queries.GetKnowledgebaseByID(r.Context(), id)
	if err != nil || entry.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Entry not found")
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *KnowledgebaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	var req struct {
		Category  string `json:"category"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		SortOrder int64  `json:"sort_order"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Category == "" || req.Title == "" || req.Content == "" {
		writeError(w, http.StatusBadRequest, "category, title, and content are required")
		return
	}

	id := newID()
	err := h.queries.CreateKnowledgebaseEntry(r.Context(), store.CreateKnowledgebaseEntryParams{
		ID:        id,
		SiteID:    siteID,
		Category:  req.Category,
		Title:     req.Title,
		Content:   req.Content,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create entry")
		return
	}

	entry, _ := h.queries.GetKnowledgebaseByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, entry)
}

func (h *KnowledgebaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "entryID")
	existing, err := h.queries.GetKnowledgebaseByID(r.Context(), id)
	if err != nil || existing.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Entry not found")
		return
	}

	var req struct {
		Category  *string `json:"category"`
		Title     *string `json:"title"`
		Content   *string `json:"content"`
		IsActive  *int64  `json:"is_active"`
		SortOrder *int64  `json:"sort_order"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	cat := existing.Category
	if req.Category != nil {
		cat = *req.Category
	}
	title := existing.Title
	if req.Title != nil {
		title = *req.Title
	}
	content := existing.Content
	if req.Content != nil {
		content = *req.Content
	}
	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	sortOrder := existing.SortOrder
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	err = h.queries.UpdateKnowledgebaseEntry(r.Context(), store.UpdateKnowledgebaseEntryParams{
		ID:        id,
		Category:  cat,
		Title:     title,
		Content:   content,
		IsActive:  isActive,
		SortOrder: sortOrder,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update entry")
		return
	}

	entry, _ := h.queries.GetKnowledgebaseByID(r.Context(), id)
	writeJSON(w, http.StatusOK, entry)
}

func (h *KnowledgebaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "entryID")
	existing, err := h.queries.GetKnowledgebaseByID(r.Context(), id)
	if err != nil || existing.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Entry not found")
		return
	}
	if err := h.queries.DeleteKnowledgebaseEntry(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete entry")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
