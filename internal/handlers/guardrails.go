package handlers

import (
	"net/http"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// GuardrailHandler handles guardrail rule CRUD endpoints (admin).
type GuardrailHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewGuardrailHandler(cfg *config.Config, queries *store.Queries) *GuardrailHandler {
	return &GuardrailHandler{cfg: cfg, queries: queries}
}

func (h *GuardrailHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	rules, err := h.queries.ListGuardrailsBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list guardrails")
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *GuardrailHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "ruleID")
	rule, err := h.queries.GetGuardrailByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Rule not found")
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *GuardrailHandler) Create(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	var req struct {
		RuleType string `json:"rule_type"`
		Target   string `json:"target"`
		Value    string `json:"value"`
		Severity string `json:"severity"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.RuleType == "" || req.Target == "" || req.Value == "" {
		writeError(w, http.StatusBadRequest, "rule_type, target, and value are required")
		return
	}
	if req.Severity == "" {
		req.Severity = "error"
	}

	id := newID()
	err := h.queries.CreateGuardrail(r.Context(), store.CreateGuardrailParams{
		ID:       id,
		SiteID:   siteID,
		RuleType: req.RuleType,
		Target:   req.Target,
		Value:    req.Value,
		Severity: req.Severity,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create rule")
		return
	}

	rule, _ := h.queries.GetGuardrailByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, rule)
}

func (h *GuardrailHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "ruleID")
	existing, err := h.queries.GetGuardrailByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Rule not found")
		return
	}

	var req struct {
		RuleType *string `json:"rule_type"`
		Target   *string `json:"target"`
		Value    *string `json:"value"`
		Severity *string `json:"severity"`
		IsActive *int64  `json:"is_active"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	rt := existing.RuleType
	if req.RuleType != nil {
		rt = *req.RuleType
	}
	tgt := existing.Target
	if req.Target != nil {
		tgt = *req.Target
	}
	val := existing.Value
	if req.Value != nil {
		val = *req.Value
	}
	sev := existing.Severity
	if req.Severity != nil {
		sev = *req.Severity
	}
	active := existing.IsActive
	if req.IsActive != nil {
		active = *req.IsActive
	}

	err = h.queries.UpdateGuardrail(r.Context(), store.UpdateGuardrailParams{
		ID:       id,
		RuleType: rt,
		Target:   tgt,
		Value:    val,
		Severity: sev,
		IsActive: active,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update rule")
		return
	}

	rule, _ := h.queries.GetGuardrailByID(r.Context(), id)
	writeJSON(w, http.StatusOK, rule)
}

func (h *GuardrailHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := urlParam(r, "ruleID")
	if err := h.queries.DeleteGuardrail(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
