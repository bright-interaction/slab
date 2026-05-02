package builder

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// PresetCookieDeclarations returns the cookie metadata atomicsite already
// knows about a tenant site's stack: always-on entries (consent record,
// language) plus conditional rows gated on the relevant analytics
// settings (GA4 enabled, etc.).
//
// Presets exist so a tenant who hasn't manually filled in the per-cookie
// table still gets a populated extended-settings modal — and stays
// IMY-2026-aligned out of the box. Tenants can override any preset by
// adding a row with the same (category, name) to their user list; the
// merge step in mergeDeclarations applies user-wins precedence.
//
// Exported so the admin API (cookie-presets endpoint) can show tenants
// which rows come from their tracker stack vs. their own edits.
func PresetCookieDeclarations(site store.Site, settingsMap map[string]string) []CookieDeclaration {
	out := []CookieDeclaration{
		{
			Category: "necessary",
			Name:     AtomicSiteConsentCookieName,
			Provider: "This site",
			Purpose:  "Stores consent preferences",
			Expiry:   "1 year",
		},
	}

	// Tracker presets. Each block fires when the operator has either
	// flipped the corresponding *_enabled boolean or pasted a tracker
	// ID. Mirrors CookieProof's KNOWN_COOKIES catalogue
	// (CookieProof/src/scanner/cookie-database.ts) so atomicsite tenants
	// who connect a tracker get the right disclosure rows automatically.
	if boolSetting(settingsMap["analytics.ga4_enabled"], false) ||
		strings.TrimSpace(settingsMap["analytics.ga4_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "analytics", Name: "_ga", Provider: "Google", Purpose: "Distinguishes unique users", Expiry: "2 years"},
			CookieDeclaration{Category: "analytics", Name: "_ga_*", Provider: "Google", Purpose: "Stores session state", Expiry: "2 years"},
			CookieDeclaration{Category: "analytics", Name: "_gid", Provider: "Google", Purpose: "Distinguishes users", Expiry: "24 hours"},
			CookieDeclaration{Category: "analytics", Name: "_gac_*", Provider: "Google", Purpose: "Campaign information", Expiry: "90 days"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.gtm_id"]) != "" &&
		strings.TrimSpace(settingsMap["analytics.ga4_id"]) == "" {
		// GTM with no GA4 still loads GA via cookies under the same _ga name.
		out = append(out,
			CookieDeclaration{Category: "analytics", Name: "_ga", Provider: "Google", Purpose: "Distinguishes unique users (loaded via Google Tag Manager)", Expiry: "2 years"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.google_ads_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "marketing", Name: "_gcl_au", Provider: "Google", Purpose: "Stores and tracks conversions", Expiry: "90 days"},
			CookieDeclaration{Category: "marketing", Name: "_gcl_aw", Provider: "Google", Purpose: "Conversion linker for Google Ads click tracking", Expiry: "90 days"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.meta_pixel_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "marketing", Name: "_fbp", Provider: "Meta", Purpose: "Used to deliver, measure, and improve advertising relevance", Expiry: "3 months"},
			CookieDeclaration{Category: "marketing", Name: "_fbc", Provider: "Meta", Purpose: "Stores click identifier from Facebook ad clicks", Expiry: "3 months"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.linkedin_insight_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "marketing", Name: "li_sugr", Provider: "LinkedIn", Purpose: "Used for LinkedIn conversion tracking", Expiry: "3 months"},
			CookieDeclaration{Category: "marketing", Name: "UserMatchHistory", Provider: "LinkedIn", Purpose: "LinkedIn Ads ID syncing", Expiry: "30 days"},
			CookieDeclaration{Category: "marketing", Name: "li_fat_id", Provider: "LinkedIn", Purpose: "LinkedIn member identifier for conversion tracking", Expiry: "30 days"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.tiktok_pixel_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "marketing", Name: "_ttp", Provider: "TikTok", Purpose: "Used by TikTok to track visits and attribute conversions", Expiry: "13 months"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.hubspot_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "marketing", Name: "hubspotutk", Provider: "HubSpot", Purpose: "Tracks visitor identity for HubSpot CRM", Expiry: "13 months"},
			CookieDeclaration{Category: "marketing", Name: "__hstc", Provider: "HubSpot", Purpose: "Main tracking cookie (visitor, timestamp, session)", Expiry: "13 months"},
			CookieDeclaration{Category: "marketing", Name: "__hssc", Provider: "HubSpot", Purpose: "Tracks sessions", Expiry: "30 minutes"},
			CookieDeclaration{Category: "marketing", Name: "__hssrc", Provider: "HubSpot", Purpose: "Detects browser restart", Expiry: "Session"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.hotjar_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "analytics", Name: "_hj*", Provider: "Hotjar", Purpose: "Hotjar analytics and user feedback tools", Expiry: "1 year"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.clarity_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "analytics", Name: "_clck", Provider: "Microsoft", Purpose: "Persists the Clarity user ID", Expiry: "1 year"},
			CookieDeclaration{Category: "analytics", Name: "_clsk", Provider: "Microsoft", Purpose: "Combines pageviews into a session recording", Expiry: "1 day"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.matomo_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "analytics", Name: "_pk_id.*", Provider: "Matomo", Purpose: "Stores unique visitor ID", Expiry: "13 months"},
			CookieDeclaration{Category: "analytics", Name: "_pk_ses.*", Provider: "Matomo", Purpose: "Stores temporary session data", Expiry: "30 minutes"},
		)
	}
	if strings.TrimSpace(settingsMap["analytics.intercom_app_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "preferences", Name: "intercom-id-*", Provider: "Intercom", Purpose: "Identifies anonymous visitors for live chat", Expiry: "9 months"},
			CookieDeclaration{Category: "preferences", Name: "intercom-session-*", Provider: "Intercom", Purpose: "Maintains live chat session", Expiry: "1 week"},
		)
	}
	// Umami / Plausible: cookieless by default, no declarations.

	if strings.TrimSpace(site.Lang) != "" ||
		strings.TrimSpace(settingsMap["general.additional_langs"]) != "" {
		out = append(out, CookieDeclaration{
			Category: "preferences",
			Name:     "lang",
			Provider: "This site",
			Purpose:  "Stores language preference",
			Expiry:   "1 year",
		})
	}

	return out
}

// parseUserCookieDeclarations decodes the JSON array stored under
// analytics.cookie_declarations. Returns nil on empty/blank input. Logs
// (but does not propagate) malformed JSON so a hand-edit typo doesn't
// crash a build — the validator at the API boundary is the gate that
// keeps bad JSON out of storage in the first place.
func parseUserCookieDeclarations(raw string) []CookieDeclaration {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	var decls []CookieDeclaration
	if err := json.Unmarshal([]byte(v), &decls); err != nil {
		slog.Warn("cookie_declarations: invalid JSON in setting, ignoring",
			"err", err)
		return nil
	}
	// Strip rows with no name — they're meaningless and would render an
	// empty table cell. Defensive against partially-written admin UI rows.
	out := decls[:0]
	for _, d := range decls {
		if strings.TrimSpace(d.Name) == "" {
			continue
		}
		out = append(out, d)
	}
	return out
}

// mergeDeclarations layers user entries on top of presets. User rows take
// precedence on (Category, Name) collision so a tenant can translate a
// preset's strings to Swedish, fix a provider name, or extend the expiry
// without losing the row to the next build.
//
// Order is preserved: presets first, then any user rows that don't collide
// with a preset. This keeps the auto-populated cookies at the top of the
// table where they're easiest to scan.
func mergeDeclarations(presets, user []CookieDeclaration) []CookieDeclaration {
	userByKey := make(map[string]CookieDeclaration, len(user))
	userKeys := make(map[string]bool, len(user))
	for _, u := range user {
		k := declKey(u.Category, u.Name)
		userByKey[k] = u
		userKeys[k] = true
	}

	out := make([]CookieDeclaration, 0, len(presets)+len(user))
	usedFromUser := make(map[string]bool, len(user))

	for _, p := range presets {
		k := declKey(p.Category, p.Name)
		if u, ok := userByKey[k]; ok {
			out = append(out, u)
			usedFromUser[k] = true
			continue
		}
		out = append(out, p)
	}
	for _, u := range user {
		k := declKey(u.Category, u.Name)
		if usedFromUser[k] {
			continue
		}
		out = append(out, u)
	}
	return out
}

func declKey(category, name string) string {
	return strings.ToLower(strings.TrimSpace(category)) + "\x00" + strings.TrimSpace(name)
}
