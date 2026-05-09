// Package handlers - webhooks.go.
//
// Admin endpoints for outbound webhook subscriptions. Mounted under
// the siteR group so site members can configure subscriptions for
// their own site.
//
//   GET    /api/sites/{siteID}/webhooks            - list subscriptions
//   POST   /api/sites/{siteID}/webhooks            - create subscription
//   GET    /api/sites/{siteID}/webhooks/{id}       - read one
//   PATCH  /api/sites/{siteID}/webhooks/{id}       - update
//   DELETE /api/sites/{siteID}/webhooks/{id}       - delete
//   GET    /api/sites/{siteID}/webhooks/deliveries - recent delivery log
//
// Secret rotation: the secret is generated server-side at create time
// and returned ONCE in the create response. Subsequent reads return
// an empty secret field; the operator who needs to change a leaked
// secret deletes + recreates the subscription. Future iteration could
// add an explicit rotate verb; V1 keeps the surface tight.

package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// WebhookHandler owns the per-site webhook CRUD endpoints.
type WebhookHandler struct {
	queries *store.Queries
}

func NewWebhookHandler(queries *store.Queries) *WebhookHandler {
	return &WebhookHandler{queries: queries}
}

// webhookView is the JSON shape returned for list/read/update. The
// secret is intentionally omitted on every path EXCEPT the create
// response.
type webhookView struct {
	ID              string   `json:"id"`
	SiteID          string   `json:"site_id"`
	TargetURL       string   `json:"target_url"`
	EventTypes      []string `json:"event_types"`
	Active          bool     `json:"active"`
	LastDeliveredAt string   `json:"last_delivered_at"`
	LastError       string   `json:"last_error"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func toWebhookView(row store.WebhookSubscription) webhookView {
	v := webhookView{
		ID:              row.ID,
		SiteID:          row.SiteID,
		TargetURL:       row.TargetUrl,
		Active:          row.Active != 0,
		LastDeliveredAt: row.LastDeliveredAt,
		LastError:       row.LastError,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		EventTypes:      []string{},
	}
	_ = json.Unmarshal([]byte(row.EventTypesJson), &v.EventTypes)
	return v
}

// List returns every subscription on the site.
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	rows, err := h.queries.ListWebhookSubscriptionsBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list webhooks")
		return
	}
	out := make([]webhookView, 0, len(rows))
	for _, row := range rows {
		out = append(out, toWebhookView(row))
	}
	writeJSON(w, http.StatusOK, out)
}

// Create mints a fresh subscription with a server-generated 32-byte
// hex secret. The secret is returned in the response body ONCE.
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if _, err := h.queries.GetSiteByID(r.Context(), siteID); err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}
	var req struct {
		TargetURL  string   `json:"target_url"`
		EventTypes []string `json:"event_types"`
		Active     *bool    `json:"active"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	target := strings.TrimSpace(req.TargetURL)
	if target == "" {
		writeError(w, http.StatusBadRequest, "target_url is required")
		return
	}
	if !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "http://") {
		writeError(w, http.StatusBadRequest, "target_url must start with http:// or https://")
		return
	}
	if len(target) > 2048 {
		writeError(w, http.StatusBadRequest, "target_url too long (max 2048)")
		return
	}
	if len(req.EventTypes) > 50 {
		writeError(w, http.StatusBadRequest, "too many event types (max 50)")
		return
	}
	for _, t := range req.EventTypes {
		if len(t) > 128 {
			writeError(w, http.StatusBadRequest, "event type too long (max 128)")
			return
		}
	}
	secret := newWebhookSecret()
	id := newWebhookID()
	active := int64(1)
	if req.Active != nil && !*req.Active {
		active = 0
	}
	typesJSON, _ := json.Marshal(req.EventTypes)

	if err := h.queries.CreateWebhookSubscription(r.Context(), store.CreateWebhookSubscriptionParams{
		ID: id, SiteID: siteID, TargetUrl: target, Secret: secret,
		EventTypesJson: string(typesJSON), Active: active,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create webhook")
		return
	}
	AuditLog(r.Context(), h.queries, r,
		siteID, AuditActionCreate, "webhook_subscription", id,
		map[string]any{"target_url": target, "event_types": req.EventTypes})

	row, err := h.queries.GetWebhookSubscription(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load created webhook")
		return
	}
	resp := toWebhookView(row)
	writeJSON(w, http.StatusCreated, map[string]any{
		"webhook": resp,
		"secret":  secret,
		"_note":   "Save this secret now. It is returned ONCE; subsequent reads omit it. Use it to verify the X-Atomicsite-Signature header on incoming POSTs.",
	})
}

// Get reads one subscription. Secret is NEVER returned on this path.
func (h *WebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "webhookID")
	row, err := h.queries.GetWebhookSubscription(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Webhook not found")
		return
	}
	writeJSON(w, http.StatusOK, toWebhookView(row))
}

// Update changes target_url, event_types, or active. The secret is
// not editable through this path.
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "webhookID")
	row, err := h.queries.GetWebhookSubscription(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Webhook not found")
		return
	}
	var req struct {
		TargetURL  *string   `json:"target_url"`
		EventTypes *[]string `json:"event_types"`
		Active     *bool     `json:"active"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	target := row.TargetUrl
	if req.TargetURL != nil {
		t := strings.TrimSpace(*req.TargetURL)
		if !strings.HasPrefix(t, "https://") && !strings.HasPrefix(t, "http://") {
			writeError(w, http.StatusBadRequest, "target_url must start with http:// or https://")
			return
		}
		target = t
	}
	typesJSON := row.EventTypesJson
	if req.EventTypes != nil {
		b, _ := json.Marshal(*req.EventTypes)
		typesJSON = string(b)
	}
	active := row.Active
	if req.Active != nil {
		active = 0
		if *req.Active {
			active = 1
		}
	}
	if err := h.queries.UpdateWebhookSubscription(r.Context(), store.UpdateWebhookSubscriptionParams{
		ID: id, TargetUrl: target, EventTypesJson: typesJSON, Active: active,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update webhook")
		return
	}
	AuditLog(r.Context(), h.queries, r,
		siteID, AuditActionUpdate, "webhook_subscription", id,
		map[string]any{"target_url": target, "active": active != 0})
	updated, _ := h.queries.GetWebhookSubscription(r.Context(), id)
	writeJSON(w, http.StatusOK, toWebhookView(updated))
}

// Delete removes the subscription. Cascades to webhook_deliveries via FK.
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "webhookID")
	row, err := h.queries.GetWebhookSubscription(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Webhook not found")
		return
	}
	if err := h.queries.DeleteWebhookSubscription(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete webhook")
		return
	}
	AuditLog(r.Context(), h.queries, r,
		siteID, AuditActionDelete, "webhook_subscription", id, map[string]any{})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Deliveries returns the most recent delivery log rows for the
// site. Useful for the admin UI's "what fired" panel.
func (h *WebhookHandler) Deliveries(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	limit := int64(50)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	rows, err := h.queries.ListWebhookDeliveriesBySite(r.Context(), store.ListWebhookDeliveriesBySiteParams{
		SiteID: siteID, Limit: limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list deliveries")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func newWebhookID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func newWebhookSecret() string {
	var b [32]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
