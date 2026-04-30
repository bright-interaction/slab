package builder

import (
	"strings"
	"testing"

	"github.com/bright-interaction/slab/internal/store"
)

// TestRenderCookieProofSnippet_HappyPath verifies the snippet emits an inline
// config blob with the right siteId/domain/proofEndpoint, references the
// content-addressed same-origin asset, and includes the synchronous GCM stub.
// Crucially, it must NOT reference consent.example.com .  Atomic
// Site is standalone for cookies after the fold-in.
func TestRenderCookieProofSnippet_HappyPath(t *testing.T) {
	out := RenderCookieProofSnippet(CookieProofConfig{
		SiteID:    "abcdef0123456789abcdef01",
		Domain:    "example.com",
		Locale:    "sv",
		TrackPath: "/t",
		CssVars: map[string]string{
			"--cc-bg":             "#FAFAFA",
			"--cc-btn-primary-bg": "#0066CC",
		},
	})

	if strings.Contains(out, "consent.example.com") {
		t.Errorf("snippet must not reference consent.example.com: %s", out)
	}
	if !strings.Contains(out, `"siteId":"abcdef0123456789abcdef01"`) {
		t.Errorf("missing siteId in inline config: %s", out)
	}
	if !strings.Contains(out, `"domain":"example.com"`) {
		t.Errorf("missing domain in inline config: %s", out)
	}
	if !strings.Contains(out, `"proofEndpoint":"/t/consent"`) {
		t.Errorf("missing proofEndpoint in inline config: %s", out)
	}
	if !strings.Contains(out, `"--cc-bg":"#FAFAFA"`) {
		t.Errorf("missing cssVars in inline config: %s", out)
	}
	expectedAsset := "/" + CookieProofWidgetFilename()
	if !strings.Contains(out, expectedAsset) {
		t.Errorf("missing same-origin widget asset reference %q: %s", expectedAsset, out)
	}
	if !strings.Contains(out, "gtag('consent','default'") {
		t.Errorf("missing GCM default-denied stub: %s", out)
	}
}

// TestRenderCookieProofSnippet_DefaultsTrackPath confirms an empty trackPath
// falls back to /t.
func TestRenderCookieProofSnippet_DefaultsTrackPath(t *testing.T) {
	out := RenderCookieProofSnippet(CookieProofConfig{
		SiteID: "aaaaaaaaaaaaaaaaaaaaaaaa",
		Domain: "example.com",
	})
	if !strings.Contains(out, `"proofEndpoint":"/t/consent"`) {
		t.Errorf("expected default /t/consent: %s", out)
	}
}

// TestRenderCookieProofSnippet_TrimsTrailingSlash ensures /t/ doesn't produce
// /t//consent.
func TestRenderCookieProofSnippet_TrimsTrailingSlash(t *testing.T) {
	out := RenderCookieProofSnippet(CookieProofConfig{
		SiteID:    "aaaaaaaaaaaaaaaaaaaaaaaa",
		Domain:    "example.com",
		TrackPath: "/t/",
	})
	if strings.Contains(out, `"proofEndpoint":"/t//consent"`) {
		t.Errorf("did not trim trailing slash: %s", out)
	}
	if !strings.Contains(out, `"proofEndpoint":"/t/consent"`) {
		t.Errorf("expected /t/consent: %s", out)
	}
}

// TestRenderCookieProofSnippet_EmptyArgs returns "" when required args are
// missing so the layout doesn't emit a broken snippet.
func TestRenderCookieProofSnippet_EmptyArgs(t *testing.T) {
	if got := RenderCookieProofSnippet(CookieProofConfig{Domain: "example.com"}); got != "" {
		t.Errorf("expected empty for missing siteID, got %q", got)
	}
	if got := RenderCookieProofSnippet(CookieProofConfig{SiteID: "aaaa"}); got != "" {
		t.Errorf("expected empty for missing domain, got %q", got)
	}
}

// TestRenderCookieProofSnippet_CopyOverrides confirms title/description/etc.
// land in the inline config.copy object so the widget can use them as
// translation overrides.
func TestRenderCookieProofSnippet_CopyOverrides(t *testing.T) {
	out := RenderCookieProofSnippet(CookieProofConfig{
		SiteID:        "aaaaaaaaaaaaaaaaaaaaaaaa",
		Domain:        "example.com",
		Title:         "Vi använder kakor",
		AcceptLabel:   "Acceptera",
		RejectLabel:   "Avvisa",
		SettingsLabel: "Anpassa",
	})
	if !strings.Contains(out, `"title":"Vi använder kakor"`) {
		t.Errorf("missing title override: %s", out)
	}
	if !strings.Contains(out, `"accept":"Acceptera"`) {
		t.Errorf("missing accept override: %s", out)
	}
	if !strings.Contains(out, `"reject":"Avvisa"`) {
		t.Errorf("missing reject override: %s", out)
	}
	if !strings.Contains(out, `"customize":"Anpassa"`) {
		t.Errorf("missing customize override: %s", out)
	}
}

// TestCookieProofWidgetHash_StableAcrossCalls makes sure the cache-bust hash
// is deterministic for a given embedded bundle, so tenant builds reuse the
// same filename across rebuilds (and CDN caches stay warm).
func TestCookieProofWidgetHash_StableAcrossCalls(t *testing.T) {
	a := CookieProofWidgetHash()
	b := CookieProofWidgetHash()
	if a != b {
		t.Errorf("hash should be stable, got %q vs %q", a, b)
	}
	if len(a) != 8 {
		t.Errorf("expected 8-char hash, got %d", len(a))
	}
}

// TestCookieProofThemeFromBranding_MapsPalette confirms the branding palette
// fills in the cssVars the widget exposes, and skips empty fields.
func TestCookieProofThemeFromBranding_MapsPalette(t *testing.T) {
	site := store.Site{
		BgColor:        "#FAFAFA",
		TextColor:      "#1A1A1A",
		PrimaryColor:   "#0066CC",
		OnPrimaryColor: "#FFFFFF",
		MutedColor:     "#6B7280",
		BorderColor:    "#E5E7EB",
		SurfaceColor:   "#FFFFFF",
		FontBody:       "Source Sans Pro",
	}
	vars := cookieProofThemeFromBranding(site)
	if vars["--cc-bg"] != "#FAFAFA" {
		t.Errorf("--cc-bg = %q want #FAFAFA", vars["--cc-bg"])
	}
	if vars["--cc-btn-primary-bg"] != "#0066CC" {
		t.Errorf("--cc-btn-primary-bg = %q want #0066CC", vars["--cc-btn-primary-bg"])
	}
	if vars["--cc-btn-primary-text"] != "#FFFFFF" {
		t.Errorf("--cc-btn-primary-text = %q want #FFFFFF", vars["--cc-btn-primary-text"])
	}
	font := vars["--cc-font"]
	if !strings.HasPrefix(font, `"Source Sans Pro"`) {
		t.Errorf("multi-word font family should be quoted, got %q", font)
	}
}

func TestCookieProofThemeFromBranding_SkipsEmpty(t *testing.T) {
	site := store.Site{} // every field empty
	vars := cookieProofThemeFromBranding(site)
	if len(vars) != 0 {
		t.Errorf("expected no vars for empty branding, got %d: %v", len(vars), vars)
	}
}
