package builder

import (
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// cookieProofThemeFromBranding maps a site row's branding palette to the CSS
// custom properties the CookieProof widget reads at render time. Every key in
// the returned map is on the widget's allowlist (see ALLOWED_CSS_VARS in
// CookieProof/src/embed.ts) so the widget will accept them.
//
// Goal: a tenant who customises their Atomic Site branding gets a cookie
// banner that matches their site without touching a separate configurator.
//
// Mapping:
//
//	bg              -> --cc-bg               banner panel surface
//	surface         -> --cc-bg-secondary     preference modal panels
//	text            -> --cc-text             headlines and body
//	muted           -> --cc-text-secondary   descriptions
//	border          -> --cc-border           hairlines
//	primary         -> --cc-btn-primary-bg   accept-all background
//	on_primary      -> --cc-btn-primary-text accept-all label
//	bg              -> --cc-btn-secondary-bg settings/customise button
//	text            -> --cc-btn-secondary-text settings label
//	muted           -> --cc-btn-reject-bg    reject background
//	on_primary      -> --cc-btn-reject-text  reject label
//	primary         -> --cc-toggle-on        active toggle
//	muted           -> --cc-toggle-off       inactive toggle
//	font_body       -> --cc-font             stack
//
// Empty branding fields are skipped .  the widget falls back to its own
// defaults for any cssVar it does not see.
func cookieProofThemeFromBranding(site store.Site) map[string]string {
	vars := map[string]string{}

	put := func(key, value string) {
		v := strings.TrimSpace(value)
		if v == "" {
			return
		}
		vars[key] = v
	}

	put("--cc-bg", site.BgColor)
	put("--cc-bg-secondary", site.SurfaceColor)
	put("--cc-text", site.TextColor)
	put("--cc-text-secondary", site.MutedColor)
	put("--cc-border", site.BorderColor)

	put("--cc-btn-primary-bg", site.PrimaryColor)
	put("--cc-btn-primary-text", site.OnPrimaryColor)

	// Secondary button (Customize) .  uses surface tone so it reads as a
	// quieter CTA next to the primary Accept.
	put("--cc-btn-secondary-bg", site.SurfaceColor)
	put("--cc-btn-secondary-text", site.TextColor)

	// Reject button .  neutral, distinct from accept. Map to muted/text so
	// it's visible but never as loud as Accept.
	put("--cc-btn-reject-bg", site.SurfaceColor)
	put("--cc-btn-reject-text", site.TextColor)

	put("--cc-toggle-on", site.PrimaryColor)
	put("--cc-toggle-off", site.MutedColor)

	if family := strings.TrimSpace(site.FontBody); family != "" {
		put("--cc-font", quoteFontFamily(family)+", system-ui, -apple-system, BlinkMacSystemFont, sans-serif")
	}

	return vars
}

// quoteFontFamily wraps a family name in quotes when it contains a space, to
// match CSS font-family syntax. Single-token names ("Inter") are emitted bare;
// multi-token names ("Source Sans Pro") get quoted.
func quoteFontFamily(name string) string {
	if name == "" {
		return ""
	}
	if strings.ContainsAny(name, " \t") {
		return `"` + strings.ReplaceAll(name, `"`, `\"`) + `"`
	}
	return name
}
