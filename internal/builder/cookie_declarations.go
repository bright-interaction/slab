package builder

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/store"
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

	if boolSetting(settingsMap["analytics.ga4_enabled"], false) ||
		strings.TrimSpace(settingsMap["analytics.ga4_id"]) != "" {
		out = append(out,
			CookieDeclaration{Category: "analytics", Name: "_ga", Provider: "Google", Purpose: "Distinguishes unique users", Expiry: "2 years"},
			CookieDeclaration{Category: "analytics", Name: "_ga_*", Provider: "Google", Purpose: "Stores session state", Expiry: "2 years"},
			CookieDeclaration{Category: "analytics", Name: "_gid", Provider: "Google", Purpose: "Distinguishes users", Expiry: "24 hours"},
			CookieDeclaration{Category: "analytics", Name: "_gac_*", Provider: "Google", Purpose: "Campaign information", Expiry: "90 days"},
		)
	}

	// Umami is cookieless by default — no analytics declarations to add.

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
