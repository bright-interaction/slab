package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// ConsentHandler exposes admin-side reads over the consent_records table . 
// the GDPR proof-of-consent log atomicsite became system of record for after
// the CookieProof fold-in. Reads are site-scoped and require an
// authenticated admin (mounted behind the auth middleware in server.go).
type ConsentHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewConsentHandler(cfg *config.Config, queries *store.Queries) *ConsentHandler {
	return &ConsentHandler{cfg: cfg, queries: queries}
}

// proofRow is the JSON shape the dashboard receives. Mirrors the table but
// hides the IP hash to half its length (truncated for display). Server-side
// the full hash stays available for audit lookups.
type proofRow struct {
	ID             string         `json:"id"`
	CreatedAt      string         `json:"created_at"`
	CreatedAtMs    int64          `json:"created_at_ms"`
	Domain         string         `json:"domain"`
	Method         string         `json:"method"`
	Version        int64          `json:"version"`
	Categories     map[string]any `json:"categories"`
	Page           string         `json:"page"`
	Referrer       string         `json:"referrer"`
	UserAgent      string         `json:"user_agent"`
	IPHashTrunc    string         `json:"ip_hash_trunc"`
	GPCActive      bool           `json:"gpc_active"`
	SessionID      string         `json:"session_id,omitempty"`
}

func toProofRow(r store.ConsentRecord) proofRow {
	cats := map[string]any{}
	if r.CategoriesJson != "" {
		_ = decodeJSON(r.CategoriesJson, &cats)
	}
	return proofRow{
		ID:          r.ID,
		CreatedAt:   r.CreatedAtIso,
		CreatedAtMs: r.CreatedAt,
		Domain:      r.Domain,
		Method:      r.ConsentMethod,
		Version:     r.ConsentVersion,
		Categories:  cats,
		Page:        r.PageUrl,
		Referrer:    r.Referrer,
		UserAgent:   r.UserAgent,
		IPHashTrunc: truncateIPHash(r.IpHash),
		GPCActive:   r.GpcActive == 1,
		SessionID:   r.SessionID,
	}
}

func truncateIPHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

// parseTimeRange pulls ?from and ?to from the request and converts them to
// unix-ms. Defaults: from = 30 days ago, to = now. Both accept RFC3339
// or unix-ms.
func parseTimeRange(r *http.Request) (int64, int64) {
	now := time.Now().UTC()
	to := now.UnixMilli()
	from := now.Add(-30 * 24 * time.Hour).UnixMilli()
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t.UnixMilli()
		} else if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			from = n
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t.UnixMilli()
		} else if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			to = n
		}
	}
	return from, to
}

// List returns paginated consent proofs for a site. Filters: from, to,
// method (one of accept-all/reject-all/custom/gpc/dns/do-not-sell). Default
// page size 50, max 200.
func (h *ConsentHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid siteID")
		return
	}
	from, to := parseTimeRange(r)
	method := strings.TrimSpace(r.URL.Query().Get("method"))
	if method != "" && !validConsentMethods[method] {
		writeError(w, http.StatusBadRequest, "Invalid method filter")
		return
	}
	limit := int64(50)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	offset := int64(0)
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			offset = n
		}
	}
	rows, err := h.queries.ListConsentBySite(r.Context(), store.ListConsentBySiteParams{
		SiteID:      siteID,
		CreatedAt:   from,
		CreatedAt_2: to,
		Method:      method,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list consent records")
		return
	}
	total, err := h.queries.CountConsentBySite(r.Context(), store.CountConsentBySiteParams{
		SiteID:      siteID,
		CreatedAt:   from,
		CreatedAt_2: to,
		Method:      method,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to count consent records")
		return
	}
	out := make([]proofRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, toProofRow(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"records": out,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"from":    from,
		"to":      to,
	})
}

// Stats returns aggregate counters and a daily time-series for charting.
// Same time-range semantics as List.
func (h *ConsentHandler) Stats(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid siteID")
		return
	}
	from, to := parseTimeRange(r)
	stats, err := h.queries.ConsentStatsBySite(r.Context(), store.ConsentStatsBySiteParams{
		SiteID:      siteID,
		CreatedAt:   from,
		CreatedAt_2: to,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch stats")
		return
	}
	daily, err := h.queries.ConsentDailyBySite(r.Context(), store.ConsentDailyBySiteParams{
		SiteID:      siteID,
		CreatedAt:   from,
		CreatedAt_2: to,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch time-series")
		return
	}
	type daySplit struct {
		Day        string           `json:"day"`
		ByMethod   map[string]int64 `json:"by_method"`
		Total      int64            `json:"total"`
	}
	dayIndex := map[string]*daySplit{}
	dayOrder := []string{}
	for _, row := range daily {
		day := fmt.Sprint(row.Day)
		ds, ok := dayIndex[day]
		if !ok {
			ds = &daySplit{Day: day, ByMethod: map[string]int64{}}
			dayIndex[day] = ds
			dayOrder = append(dayOrder, day)
		}
		ds.ByMethod[row.ConsentMethod] += row.N
		ds.Total += row.N
	}
	series := make([]daySplit, 0, len(dayOrder))
	for _, d := range dayOrder {
		series = append(series, *dayIndex[d])
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total":           stats.Total,
		"accepts":         nullFloatToInt64(stats.Accepts),
		"rejects":         nullFloatToInt64(stats.Rejects),
		"customs":         nullFloatToInt64(stats.Customs),
		"gpcs":            nullFloatToInt64(stats.Gpcs),
		"dns":             nullFloatToInt64(stats.DnsCount),
		"do_not_sell":     nullFloatToInt64(stats.DoNotSellCount),
		"daily":           series,
		"from":            from,
		"to":              to,
	})
}

// ExportCSV streams the full proof log for a date range as CSV. Designed
// for compliance audits ("show me every consent record from Q1 2026") so it
// uses StreamConsentBySite (no pagination) and writes directly to the
// response.
func (h *ConsentHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid siteID")
		return
	}
	from, to := parseTimeRange(r)
	rows, err := h.queries.StreamConsentBySite(r.Context(), store.StreamConsentBySiteParams{
		SiteID:      siteID,
		CreatedAt:   from,
		CreatedAt_2: to,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch consent records")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="consent-records-%s.csv"`, siteID))
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{
		"id", "created_at_iso", "created_at_ms", "domain", "method", "version",
		"categories_json", "gpc_active", "session_id", "page_url", "referrer",
		"user_agent", "ip_hash",
	})
	for _, row := range rows {
		_ = cw.Write([]string{
			row.ID,
			row.CreatedAtIso,
			strconv.FormatInt(row.CreatedAt, 10),
			row.Domain,
			row.ConsentMethod,
			strconv.FormatInt(row.ConsentVersion, 10),
			row.CategoriesJson,
			strconv.FormatInt(row.GpcActive, 10),
			row.SessionID,
			row.PageUrl,
			row.Referrer,
			row.UserAgent,
			row.IpHash,
		})
	}
}

// Get returns a single proof record (for legal request lookup by ID).
func (h *ConsentHandler) Get(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "recordID")
	if !isSafeSiteID(siteID) || strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	row, err := h.queries.GetConsentByID(r.Context(), store.GetConsentByIDParams{
		ID:     id,
		SiteID: siteID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "Record not found")
		return
	}
	writeJSON(w, http.StatusOK, toProofRow(row))
}
