package builder

import (
	"strings"
	"testing"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// TestRenderCookieProofSnippet_HappyPath verifies the snippet emits the
// expected single same-origin script tag and that the per-site config
// prefix carries siteId/domain/proofEndpoint/cssVars + GCM default-denied
// stub. After 2026-05-01 the snippet became one tag; the config + GCM
// stub moved into the per-site bundle so CSP `script-src 'self'` doesn't
// have to allow inline.
func TestRenderCookieProofSnippet_HappyPath(t *testing.T) {
	cfg := CookieProofConfig{
		SiteID:    "abcdef0123456789abcdef01",
		Domain:    "example.com",
		Locale:    "sv",
		TrackPath: "/t",
		CssVars: map[string]string{
			"--cc-bg":             "#FAFAFA",
			"--cc-btn-primary-bg": "#0066CC",
		},
	}
	out := RenderCookieProofSnippet(cfg)
	expectedAsset := "/" + CookieProofWidgetFilename()
	if !strings.Contains(out, expectedAsset) {
		t.Errorf("missing same-origin widget asset reference %q: %s", expectedAsset, out)
	}
	if strings.Contains(out, "consent.example.com") {
		t.Errorf("snippet must not reference consent.example.com: %s", out)
	}
	// Inline scripts moved to the per-site bundle. Validate the bundle
	// prefix carries the config + GCM stub.
	prefix, err := renderCookieProofConfigPrefix(cfg)
	if err != nil {
		t.Fatalf("config prefix: %v", err)
	}
	prefixStr := string(prefix)
	if !strings.Contains(prefixStr, `"siteId":"abcdef0123456789abcdef01"`) {
		t.Errorf("missing siteId in bundle prefix: %s", prefixStr)
	}
	if !strings.Contains(prefixStr, `"domain":"example.com"`) {
		t.Errorf("missing domain in bundle prefix: %s", prefixStr)
	}
	if !strings.Contains(prefixStr, `"proofEndpoint":"/t/consent"`) {
		t.Errorf("missing proofEndpoint in bundle prefix: %s", prefixStr)
	}
	if !strings.Contains(prefixStr, `"--cc-bg":"#FAFAFA"`) {
		t.Errorf("missing cssVars in bundle prefix: %s", prefixStr)
	}
	if !strings.Contains(prefixStr, "gtag('consent','default'") {
		t.Errorf("missing GCM default-denied stub in bundle prefix: %s", prefixStr)
	}
}

// TestRenderCookieProofSnippet_DefaultsTrackPath confirms an empty trackPath
// falls back to /t in the per-site bundle prefix.
func TestRenderCookieProofSnippet_DefaultsTrackPath(t *testing.T) {
	prefix, err := renderCookieProofConfigPrefix(CookieProofConfig{
		SiteID: "aaaaaaaaaaaaaaaaaaaaaaaa",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("config prefix: %v", err)
	}
	if !strings.Contains(string(prefix), `"proofEndpoint":"/t/consent"`) {
		t.Errorf("expected default /t/consent: %s", string(prefix))
	}
}

// TestRenderCookieProofSnippet_TrimsTrailingSlash ensures /t/ doesn't produce
// /t//consent.
func TestRenderCookieProofSnippet_TrimsTrailingSlash(t *testing.T) {
	prefix, err := renderCookieProofConfigPrefix(CookieProofConfig{
		SiteID:    "aaaaaaaaaaaaaaaaaaaaaaaa",
		Domain:    "example.com",
		TrackPath: "/t/",
	})
	if err != nil {
		t.Fatalf("config prefix: %v", err)
	}
	if strings.Contains(string(prefix), `"proofEndpoint":"/t//consent"`) {
		t.Errorf("did not trim trailing slash: %s", string(prefix))
	}
	if !strings.Contains(string(prefix), `"proofEndpoint":"/t/consent"`) {
		t.Errorf("expected /t/consent: %s", string(prefix))
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

// TestCookieProofConfigPrefix_CopyOverrides confirms title/description/etc.
// land in the bundle-prefix config.copy object so the widget uses them
// as translation overrides.
func TestCookieProofConfigPrefix_CopyOverrides(t *testing.T) {
	prefix, err := renderCookieProofConfigPrefix(CookieProofConfig{
		SiteID:        "aaaaaaaaaaaaaaaaaaaaaaaa",
		Domain:        "example.com",
		Title:         "Vi använder kakor",
		AcceptLabel:   "Acceptera",
		RejectLabel:   "Avvisa",
		SettingsLabel: "Anpassa",
	})
	if err != nil {
		t.Fatalf("config prefix: %v", err)
	}
	out := string(prefix)
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
