package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

// AnalyticsHandler exposes admin-side reads over the visit_events table populated
// by the per-site nginx log tailer (internal/analytics).
type AnalyticsHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewAnalyticsHandler(cfg *config.Config, queries *store.Queries) *AnalyticsHandler {
	return &AnalyticsHandler{cfg: cfg, queries: queries}
}

// VisitEvents returns the most recent visit_events for a site. Newest first,
// capped at 1000 rows. ?limit=N (default 100) trims the response.
func (h *AnalyticsHandler) VisitEvents(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	limit := int64(100)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = n
		}
	}
	rows, err := h.queries.ListVisitsBySite(r.Context(), store.ListVisitsBySiteParams{
		SiteID: siteID,
		Limit:  limit,
		Offset: 0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list visit events")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// nameCount is the standard {name, count} shape returned for every
// aggregation list (top browsers, top countries, etc).
type nameCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type pathCount struct {
	Path  string `json:"path"`
	Count int64  `json:"count"`
}

type referCount struct {
	Referer string `json:"referer"`
	Count   int64  `json:"count"`
}

type timePoint struct {
	Bucket      string `json:"bucket"` // ISO date (daily) or "YYYY-MM-DDTHH" (hourly)
	Count       int64  `json:"count"`
	UniqueCount int64  `json:"unique_count"`
}

// analyticsOverview is the contract the frontend page consumes. Field
// names stay snake_case to match the existing TypeScript types.
type analyticsOverview struct {
	Since           string       `json:"since"`
	Visits          int64        `json:"visits"`
	UniqueVisitors  int64        `json:"unique_visitors"`
	IdentifiedCount int64        `json:"identified_count"`
	LiveVisitors    int64        `json:"live_visitors"`
	TopPages        []pathCount  `json:"top_pages"`
	TopReferers     []referCount `json:"top_referers"`
	TopBrowsers     []nameCount  `json:"top_browsers"`
	TopOS           []nameCount  `json:"top_os"`
	TopDevices      []nameCount  `json:"top_devices"`
	TopCountries    []nameCount  `json:"top_countries"`
	TopLanguages    []nameCount  `json:"top_languages"`
	TopUTMSources   []nameCount  `json:"top_utm_sources"`
	TopUTMCampaigns []nameCount  `json:"top_utm_campaigns"`
	TimeSeries      []timePoint  `json:"time_series"`
	BucketUnit      string       `json:"bucket_unit"` // "day" | "hour"
}

// AnalyticsOverview handles GET /api/sites/{siteID}/analytics/overview?since=7d.
// One round-trip aggregate so the dashboard renders without N+1 fetches.
func (h *AnalyticsHandler) AnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	since := r.URL.Query().Get("since")
	if since == "" {
		since = "7d"
	}
	cutoff, bucketUnit := sinceToCutoff(since)

	ctx := r.Context()
	out := analyticsOverview{
		Since:           since,
		BucketUnit:      bucketUnit,
		TopPages:        []pathCount{},
		TopReferers:     []referCount{},
		TopBrowsers:     []nameCount{},
		TopOS:           []nameCount{},
		TopDevices:      []nameCount{},
		TopCountries:    []nameCount{},
		TopLanguages:    []nameCount{},
		TopUTMSources:   []nameCount{},
		TopUTMCampaigns: []nameCount{},
		TimeSeries:      []timePoint{},
	}

	if v, err := h.queries.CountVisitsBySite(ctx, store.CountVisitsBySiteParams{SiteID: siteID, Ts: cutoff}); err == nil {
		out.Visits = v
	}
	if v, err := h.queries.CountUniqueVisitorsSince(ctx, store.CountUniqueVisitorsSinceParams{SiteID: siteID, Ts: cutoff}); err == nil {
		out.UniqueVisitors = v
	}
	if v, err := h.queries.CountLiveVisitorsSince(ctx, store.CountLiveVisitorsSinceParams{
		SiteID: siteID,
		Ts:     time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339),
	}); err == nil {
		out.LiveVisitors = v
	}

	const topN = int64(10)
	if rows, err := h.queries.CountVisitsByPath(ctx, store.CountVisitsByPathParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopPages = append(out.TopPages, pathCount{Path: r.Path, Count: r.Count})
		}
	}
	if rows, err := h.queries.TopReferers(ctx, store.TopReferersParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopReferers = append(out.TopReferers, referCount{Referer: r.Referer, Count: r.Count})
		}
	}
	if rows, err := h.queries.TopBrowsers(ctx, store.TopBrowsersParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopBrowsers = append(out.TopBrowsers, nameCount{Name: r.Name, Count: r.Count})
		}
	}
	if rows, err := h.queries.TopOS(ctx, store.TopOSParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopOS = append(out.TopOS, nameCount{Name: r.Name, Count: r.Count})
		}
	}
	if rows, err := h.queries.TopDevices(ctx, store.TopDevicesParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopDevices = append(out.TopDevices, nameCount{Name: r.Name, Count: r.Count})
		}
	}
	if rows, err := h.queries.TopCountries(ctx, store.TopCountriesParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopCountries = append(out.TopCountries, nameCount{Name: r.Name, Count: r.Count})
		}
	}
	if rows, err := h.queries.TopLanguages(ctx, store.TopLanguagesParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopLanguages = append(out.TopLanguages, nameCount{Name: r.Name, Count: r.Count})
		}
	}
	if rows, err := h.queries.TopUTMSources(ctx, store.TopUTMSourcesParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopUTMSources = append(out.TopUTMSources, nameCount{Name: r.Name, Count: r.Count})
		}
	}
	if rows, err := h.queries.TopUTMCampaigns(ctx, store.TopUTMCampaignsParams{SiteID: siteID, Ts: cutoff, Limit: topN}); err == nil {
		for _, r := range rows {
			out.TopUTMCampaigns = append(out.TopUTMCampaigns, nameCount{Name: r.Name, Count: r.Count})
		}
	}

	if bucketUnit == "hour" {
		if rows, err := h.queries.PageviewsTimeSeriesHourly(ctx, store.PageviewsTimeSeriesHourlyParams{SiteID: siteID, Ts: cutoff}); err == nil {
			for _, r := range rows {
				out.TimeSeries = append(out.TimeSeries, timePoint{Bucket: r.Hour, Count: r.Count, UniqueCount: r.UniqueCount})
			}
		}
	} else {
		if rows, err := h.queries.PageviewsTimeSeriesDaily(ctx, store.PageviewsTimeSeriesDailyParams{SiteID: siteID, Ts: cutoff}); err == nil {
			for _, r := range rows {
				out.TimeSeries = append(out.TimeSeries, timePoint{Bucket: r.Day, Count: r.Count, UniqueCount: r.UniqueCount})
			}
		}
	}

	// identified_count is the size of the identified-sessions list within
	// the window, counted by last_seen_at >= cutoff.
	if rows, err := h.queries.ListIdentifiedSessions(ctx, store.ListIdentifiedSessionsParams{SiteID: siteID, Limit: 10_000, Offset: 0}); err == nil {
		for _, s := range rows {
			if s.LastSeenAt >= cutoff {
				out.IdentifiedCount++
			}
		}
	}

	writeJSON(w, http.StatusOK, out)
}

// AnalyticsSessions handles GET /api/sites/{siteID}/analytics/sessions.
// Returns identified sessions (?identified=true) newest first, capped at 200.
// Without identified=true we return an empty list: anonymous fingerprints
// don't get individual session pages by design.
func (h *AnalyticsHandler) AnalyticsSessions(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	identified := r.URL.Query().Get("identified") == "true"
	limit := int64(25)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	offset := int64(0)
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			offset = n
		}
	}

	if !identified {
		writeJSON(w, http.StatusOK, map[string]any{"sessions": []any{}})
		return
	}

	rows, err := h.queries.ListIdentifiedSessions(r.Context(), store.ListIdentifiedSessionsParams{
		SiteID: siteID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list sessions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": rows})
}

type conversionStep struct {
	Path string `json:"path"`
	TS   string `json:"ts"`
}

type conversionPath struct {
	Steps       []conversionStep `json:"steps"`
	Email       string           `json:"email"`
	ConvertedAt string           `json:"converted_at"`
}

// AnalyticsConversionPaths handles GET /api/sites/{siteID}/analytics/conversion-paths.
// Groups visit_events for identified sessions into per-session journey lists.
func (h *AnalyticsHandler) AnalyticsConversionPaths(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	limit := int64(5)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	rows, err := h.queries.ConversionPathsForIdentified(r.Context(), store.ConversionPathsForIdentifiedParams{
		SiteID: siteID,
		Limit:  limit * 50, // 50 events per path budget
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list conversion paths")
		return
	}

	byFP := map[string]*conversionPath{}
	order := []string{}
	for _, e := range rows {
		cp, ok := byFP[e.Fingerprint]
		if !ok {
			cp = &conversionPath{Email: e.Email, ConvertedAt: e.IdentifiedAt, Steps: []conversionStep{}}
			byFP[e.Fingerprint] = cp
			order = append(order, e.Fingerprint)
		}
		cp.Steps = append(cp.Steps, conversionStep{Path: e.Path, TS: e.Ts})
	}

	out := []*conversionPath{}
	for _, fp := range order {
		out = append(out, byFP[fp])
		if int64(len(out)) >= limit {
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"paths": out})
}

// AnalyticsTrackedFields lets the admin UI render an explicit "what we
// track" panel. The list is hard-coded so we can describe each field in
// plain language and keep the user-facing privacy promise in sync with
// the actual schema.
func (h *AnalyticsHandler) AnalyticsTrackedFields(w http.ResponseWriter, r *http.Request) {
	tracked := []map[string]string{
		{"field": "fingerprint", "stored": "SHA-256 hash (16 hex chars)", "source": "IP + UA + Accept-Language + per-site salt", "purpose": "Distinct visitor count without storing IPs"},
		{"field": "path", "stored": "raw", "source": "$request_uri from nginx", "purpose": "Top pages"},
		{"field": "referer", "stored": "raw", "source": "Referer header", "purpose": "Top referrers"},
		{"field": "status", "stored": "integer", "source": "$status from nginx", "purpose": "Error rate breakdown"},
		{"field": "ms", "stored": "integer (response time)", "source": "$request_time from nginx", "purpose": "Performance signal"},
		{"field": "browser", "stored": "family name", "source": "Parsed from User-Agent", "purpose": "Top browsers"},
		{"field": "os", "stored": "family name", "source": "Parsed from User-Agent", "purpose": "Top operating systems"},
		{"field": "device", "stored": "desktop / mobile / tablet / bot", "source": "Heuristic from UA", "purpose": "Device breakdown, bot exclusion"},
		{"field": "country", "stored": "ISO 3166-1 alpha-2", "source": "CF-IPCountry header (Cloudflare)", "purpose": "Top countries (no IP geolocation)"},
		{"field": "lang", "stored": "primary tag (e.g. sv-SE)", "source": "Accept-Language header", "purpose": "Visitor language signal"},
		{"field": "utm_source/medium/campaign", "stored": "raw param value", "source": "Query string of request", "purpose": "Campaign attribution"},
		{"field": "ts", "stored": "ISO-8601 UTC", "source": "$time_iso8601 from nginx", "purpose": "Time-bucketed charts"},
	}
	notTracked := []map[string]string{
		{"field": "ip", "reason": "Hashed into fingerprint then dropped at the parser. Never written to disk."},
		{"field": "user_agent", "reason": "Parsed for browser/os/device, then dropped. Raw UA is never stored."},
		{"field": "city / region", "reason": "We use only CF-IPCountry. No IP geo-lookup, no city/region storage."},
		{"field": "screen / viewport", "reason": "Server-side log tail can't see client-side fields. Add a beacon if needed."},
		{"field": "cookies", "reason": "No analytics cookies. Visit identification uses the hashed fingerprint."},
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tracked":     tracked,
		"not_tracked": notTracked,
	})
}

// sinceToCutoff turns a since=… string into (cutoffTimestamp, bucketUnit).
// 1d window uses hourly buckets; everything else daily. "all" uses a
// far-past cutoff so every event is included.
func sinceToCutoff(since string) (cutoff string, bucketUnit string) {
	now := time.Now().UTC()
	switch strings.ToLower(strings.TrimSpace(since)) {
	case "1d", "24h":
		return now.Add(-24 * time.Hour).Format(time.RFC3339), "hour"
	case "7d":
		return now.AddDate(0, 0, -7).Format(time.RFC3339), "day"
	case "30d":
		return now.AddDate(0, 0, -30).Format(time.RFC3339), "day"
	case "90d":
		return now.AddDate(0, 0, -90).Format(time.RFC3339), "day"
	case "all":
		return "1970-01-01T00:00:00Z", "day"
	}
	return now.AddDate(0, 0, -7).Format(time.RFC3339), "day"
}
