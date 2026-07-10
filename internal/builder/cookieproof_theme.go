package builder

import (
	"strings"

	"github.com/bright-interaction/atomicsite/internal/store"
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
//	primary         -> --cc-btn-reject-bg    reject background (matches Accept; IMY 2026 equal-prominence)
//	on_primary      -> --cc-btn-reject-text  reject label
//	secondary       -> --cc-btn-secondary-bg save-preferences button (falls back to surface)
//	(computed)      -> --cc-btn-secondary-text contrasting label for the secondary fill
//	primary         -> --cc-toggle-on        active toggle
//	muted           -> --cc-toggle-off       inactive toggle
//	font_body       -> --cc-font             stack
//
// Empty branding fields are skipped — the widget falls back to its own
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

	// Reject button — same fill as Accept so both CTAs carry equal visual
	// weight. The user's brand intent and IMY 2026 guidance both say
	// "reject must be as prominent as accept" — surfacing them in the same
	// color is the simplest interpretation.
	put("--cc-btn-reject-bg", site.PrimaryColor)
	put("--cc-btn-reject-text", site.OnPrimaryColor)

	// Save Preferences button — uses the site's secondary brand color so it
	// reads as its own CTA next to the primary pair. Tenants who haven't
	// configured a secondary fall back to the surface tone (the previous
	// behaviour, a quiet near-white button).
	secondaryBg := strings.TrimSpace(site.SecondaryColor)
	if secondaryBg == "" {
		secondaryBg = strings.TrimSpace(site.SurfaceColor)
	}
	if secondaryBg != "" {
		put("--cc-btn-secondary-bg", secondaryBg)
		put("--cc-btn-secondary-text", contrastTextOn(secondaryBg, site.TextColor))
	}

	put("--cc-toggle-on", site.PrimaryColor)
	put("--cc-toggle-off", site.MutedColor)

	if family := strings.TrimSpace(site.FontBody); family != "" {
		put("--cc-font", quoteFontFamily(family)+", system-ui, -apple-system, BlinkMacSystemFont, sans-serif")
	}

	return vars
}

// contrastTextOn returns a text color (#FFFFFF or #111111) that reads
// legibly on the given background hex. Falls back to the supplied dark
// default when the background is empty or unparseable, so the existing
// site.TextColor still wins on light backgrounds.
//
// Uses the WCAG relative-luminance formula. The cutoff at 0.5 is the
// commonly cited "switch to white text" threshold; for the small sample
// of brand palettes we ship, that's enough.
func contrastTextOn(bgHex, fallbackDark string) string {
	r, g, b, ok := parseHexColor(bgHex)
	if !ok {
		if strings.TrimSpace(fallbackDark) != "" {
			return fallbackDark
		}
		return "#111111"
	}
	// Relative luminance, sRGB. Approximated linearly — close enough for a
	// readability switch.
	lum := 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
	if lum < 140 { // out of 255
		return "#FFFFFF"
	}
	if strings.TrimSpace(fallbackDark) != "" {
		return fallbackDark
	}
	return "#111111"
}

// parseHexColor accepts "#RGB", "#RRGGBB", or the same without the leading
// hash, and returns 0-255 channel values. Returns ok=false on anything else.
func parseHexColor(s string) (r, g, b int, ok bool) {
	v := strings.TrimSpace(s)
	v = strings.TrimPrefix(v, "#")
	switch len(v) {
	case 3:
		v = string([]byte{v[0], v[0], v[1], v[1], v[2], v[2]})
	case 6:
		// ok
	default:
		return 0, 0, 0, false
	}
	parse := func(hi, lo byte) (int, bool) {
		h, ok1 := hexNibble(hi)
		l, ok2 := hexNibble(lo)
		if !ok1 || !ok2 {
			return 0, false
		}
		return h*16 + l, true
	}
	rv, ok1 := parse(v[0], v[1])
	gv, ok2 := parse(v[2], v[3])
	bv, ok3 := parse(v[4], v[5])
	if !ok1 || !ok2 || !ok3 {
		return 0, 0, 0, false
	}
	return rv, gv, bv, true
}

func hexNibble(c byte) (int, bool) {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), true
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10, true
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10, true
	}
	return 0, false
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
