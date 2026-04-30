package builder

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// CookieProofConfig captures the per-tenant inputs the snippet generator
// needs. Most fields are optional and default to widget defaults if empty.
type CookieProofConfig struct {
	SiteID    string
	Domain    string
	Locale    string
	TrackPath string // POST endpoint for proofs, e.g. "/t/consent"
	CssVars   map[string]string

	// Optional banner copy overrides. Empty values fall back to the widget's
	// built-in i18n strings for the resolved Locale.
	Title       string
	Description string
	AcceptLabel string
	RejectLabel string
	SettingsLabel string

	// Banner position: "bottom" (default), "top", or "center".
	Position string

	// Categories to enable. Empty = use widget defaults (necessary, analytics,
	// marketing, preferences with necessary required, others off).
	Categories []CookieCategory

	// PrivacyPolicyURL surfaces a link in the banner footer when set.
	PrivacyPolicyURL string
}

// CookieCategory mirrors the relevant subset of the widget's CategoryConfig.
// Atomicsite stores category preferences flat per site; we render them out as
// a list when emitting the inline config.
type CookieCategory struct {
	ID       string `json:"id"`
	Required bool   `json:"required,omitempty"`
	Enabled  bool   `json:"enabled,omitempty"`
}

// RenderCookieProofSnippet returns the HTML snippet that the build pipeline
// injects into a tenant site's <head> when CookieProof is enabled.
//
// Three parts, all same-origin and inline-configured (no third-party fetches):
//
//  1. Synchronous Google Consent Mode V2 default-denied stub. Must run before
//     any tracking script so GA/GTM see denied-by-default.
//  2. Inline config blob written to `window.__CCB__` .  the embed widget
//     reads this on load instead of fetching from a remote API.
//  3. <script defer src="/_ccb.{hash}.js"> .  the widget bundle, served from
//     the tenant's own origin. Hash comes from CookieProofWidgetHash() so
//     the file is content-addressed and immutably cacheable.
//
// Replaces the previous snippet that loaded
// https://consent.example.com/loader.js as a third-party script
// and POSTed proofs back to that origin. Atomic Site is now standalone for
// cookies: tenant pages have zero third-party connections for the banner.
func RenderCookieProofSnippet(cfg CookieProofConfig) string {
	if cfg.SiteID == "" || cfg.Domain == "" {
		return ""
	}
	trackPath := cfg.TrackPath
	if trackPath == "" {
		trackPath = "/t"
	}
	proofEndpoint := strings.TrimRight(trackPath, "/") + "/consent"

	type embedConfig struct {
		SiteID            string                 `json:"siteId"`
		Domain            string                 `json:"domain"`
		ProofEndpoint     string                 `json:"proofEndpoint"`
		Language          string                 `json:"language,omitempty"`
		Position          string                 `json:"position,omitempty"`
		Categories        []CookieCategory       `json:"categories,omitempty"`
		PrivacyPolicyURL  string                 `json:"privacyPolicyUrl,omitempty"`
		GcmEnabled        bool                   `json:"gcmEnabled"`
		RespectGPC        bool                   `json:"respectGPC"`
		FloatingTrigger   string                 `json:"floatingTrigger,omitempty"`
		Theme             string                 `json:"theme,omitempty"`
		CssVars           map[string]string      `json:"cssVars,omitempty"`
		Copy              map[string]string      `json:"copy,omitempty"`
	}

	cfgJSON := embedConfig{
		SiteID:           cfg.SiteID,
		Domain:           cfg.Domain,
		ProofEndpoint:    proofEndpoint,
		Language:         cfg.Locale,
		Position:         normalisePosition(cfg.Position),
		Categories:       cfg.Categories,
		PrivacyPolicyURL: cfg.PrivacyPolicyURL,
		GcmEnabled:       true,
		RespectGPC:       true,
		FloatingTrigger:  "left",
		Theme:            "light",
		CssVars:          cfg.CssVars,
	}

	copy := map[string]string{}
	if cfg.Title != "" {
		copy["title"] = cfg.Title
	}
	if cfg.Description != "" {
		copy["description"] = cfg.Description
	}
	if cfg.AcceptLabel != "" {
		copy["accept"] = cfg.AcceptLabel
	}
	if cfg.RejectLabel != "" {
		copy["reject"] = cfg.RejectLabel
	}
	if cfg.SettingsLabel != "" {
		copy["customize"] = cfg.SettingsLabel
	}
	if len(copy) > 0 {
		cfgJSON.Copy = copy
	}

	cfgBytes, err := json.Marshal(cfgJSON)
	if err != nil {
		// Should never happen for a fixed struct; emit an empty snippet to
		// avoid breaking the build.
		return ""
	}

	var b strings.Builder
	// Part 1: GCM default-denied (synchronous).
	b.WriteString("<script>\n")
	b.WriteString("(function(){\n")
	b.WriteString("  window.dataLayer = window.dataLayer || [];\n")
	b.WriteString("  if (typeof window.gtag !== 'function') { window.gtag = function(){ window.dataLayer.push(arguments); }; }\n")
	b.WriteString("  window.gtag('consent','default',{ad_storage:'denied',analytics_storage:'denied',ad_user_data:'denied',ad_personalization:'denied',functionality_storage:'denied',personalization_storage:'denied',security_storage:'granted',wait_for_update:2500});\n")
	b.WriteString("})();\n")
	b.WriteString("</script>\n")

	// Part 2: Inline config blob.
	b.WriteString("<script>window.__CCB__=")
	b.Write(cfgBytes)
	b.WriteString(";</script>\n")

	// Part 3: Same-origin widget bundle. defer preserves <head> order without
	// blocking parse.
	b.WriteString(fmt.Sprintf(`<script defer src="/%s"></script>`+"\n", CookieProofWidgetFilename()))

	return b.String()
}

// WriteCookieProofWidgetAsset writes the embedded widget bundle into the
// tenant workspace's public/ directory under the cache-busted filename
// returned by CookieProofWidgetFilename. Idempotent: overwrites any
// previous version on each build.
func WriteCookieProofWidgetAsset(workspaceDir string) error {
	publicDir := filepath.Join(workspaceDir, "public")
	if err := EnsureDir(publicDir); err != nil {
		return fmt.Errorf("ensure public dir: %w", err)
	}
	target := filepath.Join(publicDir, CookieProofWidgetFilename())
	return WriteFile(target, string(CookieProofWidget))
}


// BuildCookieProofConfig assembles a CookieProofConfig from a site row +
// settings map + any per-site profile, picking sensible defaults so the
// caller doesn't have to know which keys map where.
func BuildCookieProofConfig(site store.Site, settingsMap map[string]string) CookieProofConfig {
	domain := strings.TrimSpace(site.CookieproofDomain)
	if domain == "" {
		domain = strings.TrimSpace(site.Domain)
	}
	cfg := CookieProofConfig{
		SiteID:    site.ID,
		Domain:    domain,
		Locale:    strings.TrimSpace(site.Lang),
		TrackPath: orDefault(settingsMap["analytics.track_path"], "/t"),
		CssVars:   cookieProofThemeFromBranding(site),
	}

	if v := strings.TrimSpace(settingsMap["analytics.cookie_banner_position"]); v != "" {
		cfg.Position = v
	}
	cfg.Title = settingsMap["analytics.cookie_banner_title"]
	cfg.Description = settingsMap["analytics.cookie_banner_description"]
	cfg.AcceptLabel = settingsMap["analytics.cookie_banner_accept"]
	cfg.RejectLabel = settingsMap["analytics.cookie_banner_reject"]
	cfg.SettingsLabel = settingsMap["analytics.cookie_banner_customize"]
	cfg.PrivacyPolicyURL = settingsMap["seo.privacy_policy_url"]

	cfg.Categories = []CookieCategory{
		{ID: "necessary", Required: true, Enabled: true},
		{ID: "analytics", Enabled: boolSetting(settingsMap["analytics.cookie_cat_analytics"], true)},
		{ID: "marketing", Enabled: boolSetting(settingsMap["analytics.cookie_cat_marketing"], true)},
		{ID: "preferences", Enabled: boolSetting(settingsMap["analytics.cookie_cat_preferences"], true)},
	}

	return cfg
}

// normalisePosition clamps the position string to the widget's accepted
// values. Returns "" for unknown values so the widget falls back to its own
// default ("bottom").
func normalisePosition(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "top", "bottom", "center":
		return strings.ToLower(strings.TrimSpace(p))
	default:
		return ""
	}
}
