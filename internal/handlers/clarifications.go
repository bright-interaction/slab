// Package handlers: clarifications endpoint.
//
// Backs the request_clarification + get_clarification MCP tools and the
// dashboard inbox surface. When the agent reaches a decision point that
// would otherwise force a guess (which hero variant, formal vs casual
// voice, Stockholm office in hero or footer, etc), it opens a
// clarification: a row in the clarifications table with the question,
// optional pre-baked option list, and a snapshot of the context the
// human needs to make a sensible choice.
//
// Loop:
//   1. Agent calls MCP tool request_clarification(question, options[], context)
//   2. Handler creates a pending row, returns the id
//   3. Dashboard surfaces pending rows in an inbox; human picks an option
//      or writes free-text, hits resolve.
//   4. Agent polls MCP tool get_clarification(id) until status flips
//      to resolved or cancelled, then keeps building with the answer.
//
// Closes the "agent guesses, ships wrong, corrects after-the-fact"
// loop that costs the one-shot wow.

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/config"
	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

const clarificationsListDefaultLimit = 50

type ClarificationsHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewClarificationsHandler(cfg *config.Config, queries *store.Queries) *ClarificationsHandler {
	return &ClarificationsHandler{cfg: cfg, queries: queries}
}

type clarificationOut struct {
	ID               string   `json:"id"`
	SiteID           string   `json:"site_id"`
	RequestedBy      string   `json:"requested_by"`
	Question         string   `json:"question"`
	Context          string   `json:"context"`
	Options          []string `json:"options"`
	Status           string   `json:"status"`
	ResolutionOption int      `json:"resolution_option"`
	ResolutionText   string   `json:"resolution_text"`
	ResolvedBy       string   `json:"resolved_by"`
	CreatedAt        string   `json:"created_at"`
	ResolvedAt       string   `json:"resolved_at"`
}

func toClarificationOut(row store.Clarification) clarificationOut {
	out := clarificationOut{
		ID:               row.ID,
		SiteID:           row.SiteID,
		RequestedBy:      row.RequestedBy,
		Question:         row.Question,
		Context:          row.Context,
		Status:           row.Status,
		ResolutionOption: int(row.ResolutionOption),
		ResolutionText:   row.ResolutionText,
		ResolvedBy:       row.ResolvedBy,
		CreatedAt:        row.CreatedAt,
		ResolvedAt:       row.ResolvedAt,
	}
	if row.OptionsJson != "" {
		var opts []string
		if err := json.Unmarshal([]byte(row.OptionsJson), &opts); err == nil {
			out.Options = opts
		}
	}
	if out.Options == nil {
		out.Options = []string{}
	}
	return out
}

// CreateClarificationParams is the shared input shape for the REST
// handler AND the request_clarification MCP tool. RequestedBy is
// stamped server-side from the auth context, NOT from the body.
type CreateClarificationParams struct {
	Question string   `json:"question"`
	Context  string   `json:"context"`
	Options  []string `json:"options"`
}

// CreateClarification persists a new pending clarification. Returns
// the inserted row. Used both by the REST handler and the MCP tool
// so validation + audit shape stays in one place.
func (h *ClarificationsHandler) CreateClarification(ctx context.Context, siteID, requestedBy string, p CreateClarificationParams) (store.Clarification, error) {
	question := strings.TrimSpace(p.Question)
	if question == "" {
		return store.Clarification{}, errors.New("question required")
	}
	if len(question) > 2000 {
		return store.Clarification{}, errors.New("question must be 2000 chars or fewer")
	}
	if len(p.Context) > 8000 {
		return store.Clarification{}, errors.New("context must be 8000 chars or fewer")
	}
	if len(p.Options) > 12 {
		return store.Clarification{}, errors.New("too many options (max 12)")
	}
	clean := make([]string, 0, len(p.Options))
	for _, o := range p.Options {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if len(o) > 200 {
			return store.Clarification{}, errors.New("each option must be 200 chars or fewer")
		}
		clean = append(clean, o)
	}
	optionsJSON := "[]"
	if len(clean) > 0 {
		b, _ := json.Marshal(clean)
		optionsJSON = string(b)
	}

	id := newID()
	if err := h.queries.CreateClarification(ctx, store.CreateClarificationParams{
		ID:          id,
		SiteID:      siteID,
		RequestedBy: requestedBy,
		Question:    question,
		Context:     strings.TrimSpace(p.Context),
		OptionsJson: optionsJSON,
	}); err != nil {
		return store.Clarification{}, err
	}
	return h.queries.GetClarificationByID(ctx, store.GetClarificationByIDParams{
		ID:     id,
		SiteID: siteID,
	})
}

// Create is the admin REST entry point: an authenticated admin user
// can pre-create a clarification on the agent's behalf (e.g. to
// document a decision before the agent reaches the surface).
func (h *ClarificationsHandler) Create(w http.ResponseWriter, r *http.Request) {
	u := authmw.GetUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	siteID := urlParam(r, "siteID")
	var p CreateClarificationParams
	if err := parseJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	row, err := h.CreateClarification(r.Context(), siteID, "user:"+u.ID, p)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toClarificationOut(row))
}

// List returns the dashboard inbox view. ?status=pending (default) for
// the unread queue; ?status=all for the full history.
func (h *ClarificationsHandler) List(w http.ResponseWriter, r *http.Request) {
	if authmw.GetUser(r) == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	siteID := urlParam(r, "siteID")
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	limit := clarificationsListDefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	var rows []store.Clarification
	var err error
	switch status {
	case "", "pending":
		rows, err = h.queries.ListPendingClarificationsBySite(r.Context(), store.ListPendingClarificationsBySiteParams{
			SiteID: siteID,
			Limit:  int64(limit),
		})
	case "all":
		rows, err = h.queries.ListAllClarificationsBySite(r.Context(), store.ListAllClarificationsBySiteParams{
			SiteID: siteID,
			Limit:  int64(limit),
		})
	default:
		writeError(w, http.StatusBadRequest, "status must be pending or all")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]clarificationOut, 0, len(rows))
	for _, row := range rows {
		out = append(out, toClarificationOut(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clarifications": out,
		"count":          len(out),
	})
}

// Get returns a single row. Used by the dashboard detail view AND by
// the get_clarification MCP tool's polling loop (via a shared
// GetClarification helper below).
func (h *ClarificationsHandler) Get(w http.ResponseWriter, r *http.Request) {
	if authmw.GetUser(r) == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "id")
	row, err := h.queries.GetClarificationByID(r.Context(), store.GetClarificationByIDParams{
		ID:     id,
		SiteID: siteID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "clarification not found")
		return
	}
	writeJSON(w, http.StatusOK, toClarificationOut(row))
}

// ResolveParams is the body shape for both the REST resolve handler
// and any MCP-side resolver helpers.
type ResolveParams struct {
	Option int    `json:"option"`
	Text   string `json:"text"`
}

// Resolve flips a pending clarification to resolved. Option is the
// index into options[] (-1 for free-text-only). Text is the operator's
// elaboration / free-text answer (required when option is -1).
func (h *ClarificationsHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	u := authmw.GetUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "id")
	var p ResolveParams
	if err := parseJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if p.Option < -1 {
		writeError(w, http.StatusBadRequest, "option must be -1 (free text) or a non-negative index")
		return
	}
	text := strings.TrimSpace(p.Text)
	if p.Option == -1 && text == "" {
		writeError(w, http.StatusBadRequest, "text required when no option is picked")
		return
	}
	if len(text) > 4000 {
		writeError(w, http.StatusBadRequest, "text must be 4000 chars or fewer")
		return
	}

	row, err := h.queries.GetClarificationByID(r.Context(), store.GetClarificationByIDParams{
		ID:     id,
		SiteID: siteID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "clarification not found")
		return
	}
	if row.Status != "pending" {
		writeError(w, http.StatusConflict, "clarification already "+row.Status)
		return
	}
	if p.Option >= 0 {
		var opts []string
		if err := json.Unmarshal([]byte(row.OptionsJson), &opts); err != nil || p.Option >= len(opts) {
			writeError(w, http.StatusBadRequest, "option index out of range")
			return
		}
	}

	if err := h.queries.ResolveClarification(r.Context(), store.ResolveClarificationParams{
		ResolutionOption: int64(p.Option),
		ResolutionText:   text,
		ResolvedBy:       "user:" + u.ID,
		ID:               id,
		SiteID:           siteID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "resolve failed")
		return
	}
	row, _ = h.queries.GetClarificationByID(r.Context(), store.GetClarificationByIDParams{
		ID:     id,
		SiteID: siteID,
	})
	writeJSON(w, http.StatusOK, toClarificationOut(row))
}

// Cancel marks a pending clarification as cancelled. Used when the
// agent has moved past the surface that needed the answer, or when
// the admin wants to clear the inbox.
func (h *ClarificationsHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	u := authmw.GetUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "id")
	row, err := h.queries.GetClarificationByID(r.Context(), store.GetClarificationByIDParams{
		ID:     id,
		SiteID: siteID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "clarification not found")
		return
	}
	if row.Status != "pending" {
		writeError(w, http.StatusConflict, "clarification already "+row.Status)
		return
	}
	if err := h.queries.CancelClarification(r.Context(), store.CancelClarificationParams{
		ResolvedBy: "user:" + u.ID,
		ID:         id,
		SiteID:     siteID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "cancel failed")
		return
	}
	row, _ = h.queries.GetClarificationByID(r.Context(), store.GetClarificationByIDParams{
		ID:     id,
		SiteID: siteID,
	})
	writeJSON(w, http.StatusOK, toClarificationOut(row))
}

// GetClarification is the shared lookup the MCP tool uses to poll. Same
// query as the REST Get handler but without HTTP plumbing so the MCP
// tool can call it from request_clarification's wait-loop without
// going through the REST stack.
func (h *ClarificationsHandler) GetClarification(ctx context.Context, siteID, id string) (ClarificationView, error) {
	row, err := h.queries.GetClarificationByID(ctx, store.GetClarificationByIDParams{
		ID:     id,
		SiteID: siteID,
	})
	if err != nil {
		return ClarificationView{}, err
	}
	return toClarificationView(row), nil
}

// ListPendingByAgent returns clarifications the agent has opened that
// are still pending. Backs the list_pending_clarifications MCP tool.
func (h *ClarificationsHandler) ListPendingByAgent(ctx context.Context, siteID, requestedBy string, limit int) ([]ClarificationView, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := h.queries.ListPendingClarificationsByAgent(ctx, store.ListPendingClarificationsByAgentParams{
		SiteID:      siteID,
		RequestedBy: requestedBy,
		Limit:       int64(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]ClarificationView, 0, len(rows))
	for _, row := range rows {
		out = append(out, toClarificationView(row))
	}
	return out, nil
}

// ClarificationView is the public JSON shape. Exported so the MCP layer
// can reuse it in tool responses without redefining the field set.
type ClarificationView = clarificationOut

func toClarificationView(row store.Clarification) ClarificationView {
	return toClarificationOut(row)
}
