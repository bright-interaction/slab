package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/builder"
	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// CookiesPreviewHandler renders an authenticated HTML page that mounts the
// real CookieProof widget configured for a given site, immediately opens
// the preferences modal, and runs in the widget's previewMode (no focus
// trap, banner stays visible). The admin SPA's Settings -> Cookies page
// embeds this as an iframe so editors see the exact production banner
// (cookie tables, language toggle, privacy policy link, branding colors)
// while they edit instead of a static 3-button mockup.
//
// Mounted behind the standard site_access middleware; query params let the
// admin SPA pass through unsaved form state so the preview reflects edits
// before the user clicks Save.
type CookiesPreviewHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewCookiesPreviewHandler(cfg *config.Config, queries *store.Queries) *CookiesPreviewHandler {
	return &CookiesPreviewHandler{cfg: cfg, queries: queries}
}

func (h *CookiesPreviewHandler) Render(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	site, err := h.queries.GetSiteByID(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}

	settings, _ := h.queries.ListSettingsBySite(r.Context(), siteID)
	settingsMap := make(map[string]string, len(settings))
	for _, s := range settings {
		settingsMap[s.Category+"."+s.Key] = s.Value
	}

	// Apply query-string overrides for in-progress form state. The admin
	// SPA passes the live edited values so the preview reflects them
	// before the user hits Save.
	q := r.URL.Query()
	override := func(key, qparam string) {
		if q.Has(qparam) {
			settingsMap[key] = q.Get(qparam)
		}
	}
	override("analytics.cookie_banner_title", "title")
	override("analytics.cookie_banner_description", "description")
	override("analytics.cookie_banner_accept", "accept_label")
	override("analytics.cookie_banner_reject", "reject_label")
	override("analytics.cookie_banner_customize", "customize_label")
	override("analytics.cookie_language_selector", "language_selector")
	override("analytics.cookie_languages", "languages")
	override("analytics.cookie_theme", "theme")
	override("analytics.cookie_floating_trigger", "floating_trigger")
	override("analytics.cookie_banner_position", "position")
	override("seo.privacy_policy_url", "privacy_policy_url")
	override("analytics.ccpa_enabled", "ccpa_enabled")
	override("analytics.ccpa_url", "ccpa_url")

	// Branding overrides on the site row itself (e.g. when the user is
	// editing colors on the Branding page; the admin can pass them
	// through to see the cookie banner respond live).
	if v := q.Get("primary_color"); v != "" {
		site.PrimaryColor = v
	}
	if v := q.Get("secondary_color"); v != "" {
		site.SecondaryColor = v
	}
	if v := q.Get("on_primary_color"); v != "" {
		site.OnPrimaryColor = v
	}
	if v := q.Get("text_color"); v != "" {
		site.TextColor = v
	}
	if v := q.Get("bg_color"); v != "" {
		site.BgColor = v
	}
	if v := q.Get("surface_color"); v != "" {
		site.SurfaceColor = v
	}
	if v := q.Get("muted_color"); v != "" {
		site.MutedColor = v
	}
	if v := q.Get("border_color"); v != "" {
		site.BorderColor = v
	}
	if v := q.Get("font_body"); v != "" {
		site.FontBody = v
	}

	cpCfg := builder.BuildCookieProofConfig(site, settingsMap)

	// Force language selector ON for preview when requested. Lets editors
	// verify the dropdown UI even when the saved setting is off.
	if q.Get("force_language_selector") == "1" {
		cpCfg.LanguageSelector = true
	}

	prefix, err := builder.RenderCookieProofConfigPrefix(cpCfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to build preview config")
		return
	}

	bgColor := orDefaultStr(site.BgColor, "#FAFAFA")
	textColor := orDefaultStr(site.TextColor, "#1A1A1A")
	fontBody := orDefaultStr(site.FontBody, "system-ui, -apple-system, sans-serif")

	// Permissive CSP scoped to this preview route only. The page inlines
	// window.__CCB__ + the widget bundle bytes; both are server-generated
	// and the endpoint is admin-auth so the surface is trusted. The admin
	// SPA loads this as a same-origin iframe.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self' 'unsafe-inline'; "+
			"style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; "+
			"font-src 'self' data:; connect-src 'self'; "+
			"frame-ancestors 'self'")
	w.Header().Set("Cache-Control", "no-store")

	tpl := template.Must(template.New("preview").Parse(previewHTMLTemplate))
	_ = tpl.Execute(w, map[string]any{
		"BgColor":   bgColor,
		"TextColor": textColor,
		"FontBody":  fontBody,
		"Prefix":    template.JS(prefix),
		"WidgetJS":  template.JS(builder.CookieProofWidget),
	})
}

const previewHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Cookie banner preview</title>
<style>
  html, body { margin: 0; padding: 0; min-height: 100%; background: {{.BgColor}}; color: {{.TextColor}}; font-family: {{.FontBody}}; }
  .preview-hint { position: fixed; bottom: 12px; right: 12px; opacity: 0.4; font-size: 11px; pointer-events: none; user-select: none; }
</style>
</head>
<body>
<div class="preview-hint">Preview · live banner</div>
<script>{{.Prefix}}</script>
<script>{{.WidgetJS}}</script>
<script>
customElements.whenDefined('cookie-consent').then(function () {
  var tries = 0;
  var iv = setInterval(function () {
    var el = document.querySelector('cookie-consent');
    if (!el) {
      if (++tries > 50) clearInterval(iv);
      return;
    }
    clearInterval(iv);
    el.previewMode = true;
    if (typeof el.showPreferences === 'function') {
      el.showPreferences();
    }
  }, 30);
});
</script>
</body>
</html>`

func orDefaultStr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
