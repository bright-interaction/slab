package handlers

import (
	"net/http"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// ComponentHandler handles component library CRUD endpoints.
type ComponentHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewComponentHandler(cfg *config.Config, queries *store.Queries) *ComponentHandler {
	return &ComponentHandler{cfg: cfg, queries: queries}
}

func (h *ComponentHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	comps, err := h.queries.ListComponentsBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list components")
		return
	}
	writeJSON(w, http.StatusOK, comps)
}

func (h *ComponentHandler) Get(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "componentID")
	comp, err := h.queries.GetComponentByID(r.Context(), id)
	if err != nil || comp.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Component not found")
		return
	}
	writeJSON(w, http.StatusOK, comp)
}

func (h *ComponentHandler) Create(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	var req struct {
		Name        string   `json:"name"`
		Category    string   `json:"category"`
		Template    string   `json:"template"`
		PropsSchema any      `json:"props_schema"`
		CSSClasses  []string `json:"css_classes"`
		UsageNote   string   `json:"usage_note"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Name == "" || req.Template == "" {
		writeError(w, http.StatusBadRequest, "name and template are required")
		return
	}
	if req.Category == "" {
		req.Category = "section"
	}

	id := newID()
	err := h.queries.CreateComponent(r.Context(), store.CreateComponentParams{
		ID:          id,
		SiteID:      siteID,
		Name:        req.Name,
		Category:    req.Category,
		Template:    req.Template,
		PropsSchema: marshalField(req.PropsSchema),
		CssClasses:  marshalField(req.CSSClasses),
		UsageNote:   req.UsageNote,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create component")
		return
	}

	comp, _ := h.queries.GetComponentByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, comp)
}

func (h *ComponentHandler) Update(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "componentID")
	existing, err := h.queries.GetComponentByID(r.Context(), id)
	if err != nil || existing.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Component not found")
		return
	}

	var req struct {
		Name        *string  `json:"name"`
		Category    *string  `json:"category"`
		Template    *string  `json:"template"`
		PropsSchema any      `json:"props_schema"`
		CSSClasses  []string `json:"css_classes"`
		UsageNote   *string  `json:"usage_note"`
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
	tmpl := existing.Template
	if req.Template != nil {
		tmpl = *req.Template
	}
	propsSchema := existing.PropsSchema
	if req.PropsSchema != nil {
		propsSchema = marshalField(req.PropsSchema)
	}
	cssClasses := existing.CssClasses
	if req.CSSClasses != nil {
		cssClasses = marshalField(req.CSSClasses)
	}
	usageNote := existing.UsageNote
	if req.UsageNote != nil {
		usageNote = *req.UsageNote
	}

	err = h.queries.UpdateComponent(r.Context(), store.UpdateComponentParams{
		ID:          id,
		Name:        name,
		Category:    cat,
		Template:    tmpl,
		PropsSchema: propsSchema,
		CssClasses:  cssClasses,
		UsageNote:   usageNote,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update component")
		return
	}

	comp, _ := h.queries.GetComponentByID(r.Context(), id)
	writeJSON(w, http.StatusOK, comp)
}

func (h *ComponentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "componentID")
	existing, err := h.queries.GetComponentByID(r.Context(), id)
	if err != nil || existing.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Component not found")
		return
	}
	if err := h.queries.DeleteComponent(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete component")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
