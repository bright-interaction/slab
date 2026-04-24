package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/bright-interaction/slab/internal/agent"
	"github.com/bright-interaction/slab/internal/config"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/store"
)

// AgentHandler handles AI agent API endpoints.
type AgentHandler struct {
	cfg        *config.Config
	queries    *store.Queries
	context    *agent.ContextBuilder
	guardrails *agent.GuardrailEngine
}

func NewAgentHandler(cfg *config.Config, queries *store.Queries) *AgentHandler {
	return &AgentHandler{
		cfg:        cfg,
		queries:    queries,
		context:    agent.NewContextBuilder(queries),
		guardrails: agent.NewGuardrailEngine(queries),
	}
}

// requireAgent extracts and validates the agent identity. Returns nil and writes an error if invalid.
func requireAgent(w http.ResponseWriter, r *http.Request) *authmw.AgentIdentity {
	a := authmw.GetAgent(r)
	if a == nil {
		writeError(w, http.StatusUnauthorized, "Agent not authenticated")
		return nil
	}
	return a
}

// requireWrite checks that the agent has write capability.
func requireWrite(w http.ResponseWriter, a *authmw.AgentIdentity) bool {
	if !agent.ValidateCapability(a.Capabilities, "write") {
		writeError(w, http.StatusForbidden, "Agent key lacks 'write' capability")
		return false
	}
	return true
}

// resolvePageBySlug looks up a page by slug using the efficient indexed query.
func (h *AgentHandler) resolvePageBySlug(w http.ResponseWriter, r *http.Request, siteID string) (*store.Page, bool) {
	pageSlug := r.URL.Query().Get("page")
	if pageSlug == "" {
		pageSlug = urlParam(r, "slug")
	}
	if pageSlug == "" {
		writeError(w, http.StatusBadRequest, "page slug is required (query param ?page= or path param)")
		return nil, false
	}
	// Normalize: ensure leading slash (URL params strip it)
	if pageSlug != "/" && !strings.HasPrefix(pageSlug, "/") {
		pageSlug = "/" + pageSlug
	}
	page, err := h.queries.GetPageBySiteAndSlug(r.Context(), store.GetPageBySiteAndSlugParams{
		SiteID: siteID,
		Slug:   pageSlug,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "Page not found with slug: "+pageSlug)
		return nil, false
	}
	return &page, true
}

// --- Context ---

// Context returns the full site context for an AI agent.
func (h *AgentHandler) Context(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	ctx, err := h.context.Build(r.Context(), a.SiteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to build context")
		return
	}
	writeJSON(w, http.StatusOK, ctx)
}

// --- Pages ---

// CreatePage creates a new page via the agent API.
func (h *AgentHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	var req struct {
		Title    string `json:"title"`
		Slug     string `json:"slug"`
		Layout   string `json:"layout"`
		NavLabel string `json:"nav_label"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Layout == "" {
		req.Layout = "default"
	}

	violations := h.guardrails.ValidatePageSlug(r.Context(), a.SiteID, req.Slug)
	if agent.HasErrors(violations) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":      "Guardrail violations",
			"violations": violations,
		})
		return
	}

	id := newID()
	err := h.queries.CreatePage(r.Context(), store.CreatePageParams{
		ID:     id,
		SiteID: a.SiteID,
		Title:  req.Title,
		Slug:   req.Slug,
		Layout: req.Layout,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create page")
		return
	}

	page, _ := h.queries.GetPageByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, page)
}

// UpdatePage updates a page via the agent API.
func (h *AgentHandler) UpdatePage(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	page, ok := h.resolvePageBySlug(w, r, a.SiteID)
	if !ok {
		return
	}

	var req struct {
		Title           *string `json:"title"`
		Slug            *string `json:"slug"`
		Status          *string `json:"status"`
		MetaTitle       *string `json:"meta_title"`
		MetaDescription *string `json:"meta_description"`
		Layout          *string `json:"layout"`
		ShowInNav       *int64  `json:"show_in_nav"`
		NavLabel        *string `json:"nav_label"`
		NoIndex         *int64  `json:"no_index"`
		CanonicalURL    *string `json:"canonical_url"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	params := store.UpdatePageParams{
		ID:              page.ID,
		Title:           page.Title,
		Slug:            page.Slug,
		Status:          page.Status,
		MetaTitle:       page.MetaTitle,
		MetaDescription: page.MetaDescription,
		OgImageID:       page.OgImageID,
		Layout:          page.Layout,
		SortOrder:       page.SortOrder,
		ShowInNav:       page.ShowInNav,
		NavLabel:        page.NavLabel,
		NoIndex:         page.NoIndex,
		CanonicalUrl:    page.CanonicalUrl,
	}

	if req.Title != nil {
		params.Title = *req.Title
	}
	if req.Slug != nil && *req.Slug != page.Slug {
		violations := h.guardrails.ValidatePageSlug(r.Context(), a.SiteID, *req.Slug)
		if agent.HasErrors(violations) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":      "Guardrail violations",
				"violations": violations,
			})
			return
		}
		// Auto-create 301 redirect from old slug to new slug
		_ = h.queries.CreateRedirect(r.Context(), store.CreateRedirectParams{
			ID:         newID(),
			SiteID:     a.SiteID,
			FromPath:   page.Slug,
			ToPath:     *req.Slug,
			StatusCode: 301,
			IsAuto:     1,
		})
		params.Slug = *req.Slug
	}
	if req.Status != nil {
		params.Status = *req.Status
	}
	if req.MetaTitle != nil {
		params.MetaTitle = *req.MetaTitle
	}
	if req.MetaDescription != nil {
		params.MetaDescription = *req.MetaDescription
	}
	if req.Layout != nil {
		params.Layout = *req.Layout
	}
	if req.ShowInNav != nil {
		params.ShowInNav = *req.ShowInNav
	}
	if req.NavLabel != nil {
		params.NavLabel = *req.NavLabel
	}
	if req.NoIndex != nil {
		params.NoIndex = *req.NoIndex
	}
	if req.CanonicalURL != nil {
		params.CanonicalUrl = *req.CanonicalURL
	}

	if err := h.queries.UpdatePage(r.Context(), params); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update page")
		return
	}

	updated, _ := h.queries.GetPageByID(r.Context(), page.ID)
	writeJSON(w, http.StatusOK, updated)
}

// DeletePage deletes a page via the agent API.
func (h *AgentHandler) DeletePage(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	page, ok := h.resolvePageBySlug(w, r, a.SiteID)
	if !ok {
		return
	}

	if err := h.queries.DeletePage(r.Context(), page.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete page")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Blocks ---

// CreateBlock creates a new block on a page via the agent API.
func (h *AgentHandler) CreateBlock(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	page, ok := h.resolvePageBySlug(w, r, a.SiteID)
	if !ok {
		return
	}

	var req struct {
		BlockType string `json:"block_type"`
		Data      any    `json:"data"`
		Style     any    `json:"style"`
		SortOrder *int64 `json:"sort_order"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.BlockType == "" {
		writeError(w, http.StatusBadRequest, "block_type is required")
		return
	}

	dataStr := marshalField(req.Data)
	styleStr := marshalField(req.Style)

	violations := h.guardrails.ValidateBlock(r.Context(), a.SiteID, req.BlockType, dataStr)
	// Also check block count limit
	countViolations := h.guardrails.ValidateBlockCount(r.Context(), a.SiteID, page.ID)
	violations = append(violations, countViolations...)
	if agent.HasErrors(violations) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":      "Guardrail violations",
			"violations": violations,
		})
		return
	}

	id := newID()
	var sortOrder int64
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	err := h.queries.CreateBlock(r.Context(), store.CreateBlockParams{
		ID:        id,
		PageID:    page.ID,
		BlockType: req.BlockType,
		DataJson:  dataStr,
		StyleJson: styleStr,
		SortOrder: sortOrder,
		IsVisible: 1,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create block")
		return
	}

	block, _ := h.queries.GetBlockByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, block)
}

// UpdateBlock updates a block via the agent API.
func (h *AgentHandler) UpdateBlock(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	blockID := urlParam(r, "blockID")
	existing, err := h.queries.GetBlockByID(r.Context(), blockID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}

	var req struct {
		BlockType *string `json:"block_type"`
		Data      any     `json:"data"`
		Style     any     `json:"style"`
		IsVisible *int64  `json:"is_visible"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	blockType := existing.BlockType
	if req.BlockType != nil {
		blockType = *req.BlockType
	}
	dataStr := existing.DataJson
	if req.Data != nil {
		dataStr = marshalField(req.Data)
	}
	styleStr := existing.StyleJson
	if req.Style != nil {
		styleStr = marshalField(req.Style)
	}
	isVisible := existing.IsVisible
	if req.IsVisible != nil {
		isVisible = *req.IsVisible
	}

	violations := h.guardrails.ValidateBlock(r.Context(), a.SiteID, blockType, dataStr)
	if agent.HasErrors(violations) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":      "Guardrail violations",
			"violations": violations,
		})
		return
	}

	err = h.queries.UpdateBlock(r.Context(), store.UpdateBlockParams{
		ID:        blockID,
		BlockType: blockType,
		DataJson:  dataStr,
		StyleJson: styleStr,
		IsVisible: isVisible,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update block")
		return
	}

	block, _ := h.queries.GetBlockByID(r.Context(), blockID)
	writeJSON(w, http.StatusOK, block)
}

// DeleteBlock deletes a block via the agent API.
func (h *AgentHandler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	blockID := urlParam(r, "blockID")

	// Look up the block to find its page for required blocks check
	block, err := h.queries.GetBlockByID(r.Context(), blockID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}

	// Find the page to get its slug for required blocks validation
	page, err := h.queries.GetPageByID(r.Context(), block.PageID)
	if err == nil {
		violations := h.guardrails.ValidateRequiredBlocks(r.Context(), a.SiteID, page.Slug, page.ID, blockID)
		if agent.HasErrors(violations) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":      "Guardrail violations",
				"violations": violations,
			})
			return
		}
	}

	if err := h.queries.DeleteBlock(r.Context(), blockID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete block")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Components ---

// CreateComponent registers a new reusable component.
func (h *AgentHandler) CreateComponent(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

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
		SiteID:      a.SiteID,
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

// UpdateComponent updates an existing component.
func (h *AgentHandler) UpdateComponent(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	name := urlParam(r, "name")
	existing, err := h.queries.GetComponentByName(r.Context(), store.GetComponentByNameParams{
		SiteID: a.SiteID,
		Name:   name,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "Component not found: "+name)
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

	n := existing.Name
	if req.Name != nil {
		n = *req.Name
	}
	cat := existing.Category
	if req.Category != nil {
		cat = *req.Category
	}
	tmpl := existing.Template
	if req.Template != nil {
		tmpl = *req.Template
	}
	ps := existing.PropsSchema
	if req.PropsSchema != nil {
		ps = marshalField(req.PropsSchema)
	}
	cc := existing.CssClasses
	if req.CSSClasses != nil {
		cc = marshalField(req.CSSClasses)
	}
	un := existing.UsageNote
	if req.UsageNote != nil {
		un = *req.UsageNote
	}

	err = h.queries.UpdateComponent(r.Context(), store.UpdateComponentParams{
		ID:          existing.ID,
		Name:        n,
		Category:    cat,
		Template:    tmpl,
		PropsSchema: ps,
		CssClasses:  cc,
		UsageNote:   un,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update component")
		return
	}

	comp, _ := h.queries.GetComponentByID(r.Context(), existing.ID)
	writeJSON(w, http.StatusOK, comp)
}

// --- CSS Classes ---

// CreateCSSClass adds a new global CSS class.
func (h *AgentHandler) CreateCSSClass(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

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
		SiteID:    a.SiteID,
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

// UpdateCSSClass updates an existing CSS class.
func (h *AgentHandler) UpdateCSSClass(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	name := urlParam(r, "name")
	existing, err := h.queries.GetCSSClassByName(r.Context(), store.GetCSSClassByNameParams{
		SiteID: a.SiteID,
		Name:   name,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "CSS class not found: "+name)
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

	n := existing.Name
	if req.Name != nil {
		n = *req.Name
	}
	cat := existing.Category
	if req.Category != nil {
		cat = *req.Category
	}
	css := existing.Css
	if req.CSS != nil {
		css = *req.CSS
	}
	un := existing.UsageNote
	if req.UsageNote != nil {
		un = *req.UsageNote
	}
	so := existing.SortOrder
	if req.SortOrder != nil {
		so = *req.SortOrder
	}

	err = h.queries.UpdateCSSClass(r.Context(), store.UpdateCSSClassParams{
		ID:        existing.ID,
		Name:      n,
		Category:  cat,
		Css:       css,
		UsageNote: un,
		SortOrder: so,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update CSS class")
		return
	}

	cc, _ := h.queries.GetCSSClassByID(r.Context(), existing.ID)
	writeJSON(w, http.StatusOK, cc)
}

// --- Global Blocks ---

// UpdateGlobalBlock updates a global block by slot name.
func (h *AgentHandler) UpdateGlobalBlock(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	if !requireWrite(w, a) {
		return
	}

	slot := urlParam(r, "slot")
	blocks, _ := h.queries.ListGlobalBlocksBySite(r.Context(), a.SiteID)
	var existing *store.GlobalBlock
	for _, gb := range blocks {
		if gb.Slot == slot {
			existing = &gb
			break
		}
	}

	var req struct {
		Name      string `json:"name"`
		BlockType string `json:"block_type"`
		Data      any    `json:"data"`
		Style     any    `json:"style"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if existing == nil {
		// Create new global block
		if req.Name == "" {
			req.Name = slot
		}
		if req.BlockType == "" {
			req.BlockType = "html"
		}
		id := newID()
		err := h.queries.CreateGlobalBlock(r.Context(), store.CreateGlobalBlockParams{
			ID:        id,
			SiteID:    a.SiteID,
			Name:      req.Name,
			Slot:      slot,
			BlockType: req.BlockType,
			DataJson:  marshalField(req.Data),
			StyleJson: marshalField(req.Style),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create global block")
			return
		}
		gb, _ := h.queries.GetGlobalBlockByID(r.Context(), id)
		writeJSON(w, http.StatusCreated, gb)
		return
	}

	// Update existing
	blockType := existing.BlockType
	if req.BlockType != "" {
		blockType = req.BlockType
	}
	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}
	dataStr := existing.DataJson
	if req.Data != nil {
		dataStr = marshalField(req.Data)
	}
	styleStr := existing.StyleJson
	if req.Style != nil {
		styleStr = marshalField(req.Style)
	}

	err := h.queries.UpdateGlobalBlock(r.Context(), store.UpdateGlobalBlockParams{
		ID:        existing.ID,
		Name:      name,
		BlockType: blockType,
		DataJson:  dataStr,
		StyleJson: styleStr,
		IsActive:  existing.IsActive,
		SortOrder: existing.SortOrder,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update global block")
		return
	}

	gb, _ := h.queries.GetGlobalBlockByID(r.Context(), existing.ID)
	writeJSON(w, http.StatusOK, gb)
}

// --- Evaluation ---

// GetEvaluation returns evaluation results for a build.
func (h *AgentHandler) GetEvaluation(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}

	buildID := urlParam(r, "buildID")
	evals, err := h.queries.ListEvaluationsByBuild(r.Context(), buildID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get evaluations")
		return
	}
	writeJSON(w, http.StatusOK, evals)
}

// --- Media ---

// ListMedia lists all media for the agent's site.
func (h *AgentHandler) ListMedia(w http.ResponseWriter, r *http.Request) {
	a := requireAgent(w, r)
	if a == nil {
		return
	}
	media, err := h.queries.ListMediaBySite(r.Context(), a.SiteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list media")
		return
	}
	writeJSON(w, http.StatusOK, media)
}

// --- Admin: Agent Key Management ---

// GenerateAgentKey creates a new API key for a site (admin endpoint).
func (h *AgentHandler) GenerateAgentKey(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")

	var req struct {
		Name         string   `json:"name"`
		Capabilities []string `json:"capabilities"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Capabilities) == 0 {
		req.Capabilities = []string{"read", "write"}
	}

	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate key")
		return
	}
	rawKey := "ask_" + hex.EncodeToString(rawBytes)
	keyHash := authmw.HashAgentKey(rawKey)

	id := newID()
	err := h.queries.CreateAgentKey(r.Context(), store.CreateAgentKeyParams{
		ID:           id,
		SiteID:       siteID,
		Name:         req.Name,
		KeyHash:      keyHash,
		Capabilities: marshalField(req.Capabilities),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create agent key")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":   id,
		"name": req.Name,
		"key":  rawKey,
		"note": "Save this key now. It cannot be retrieved again.",
	})
}

// ListAgentKeys lists all agent keys for a site (admin endpoint).
func (h *AgentHandler) ListAgentKeys(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	keys, err := h.queries.ListAgentKeysBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list keys")
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

// RevokeAgentKey deactivates an agent key (admin endpoint).
func (h *AgentHandler) RevokeAgentKey(w http.ResponseWriter, r *http.Request) {
	keyID := urlParam(r, "keyID")
	if err := h.queries.DeactivateAgentKey(r.Context(), keyID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to revoke key")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
