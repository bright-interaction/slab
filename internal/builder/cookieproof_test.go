package builder

import (
	"strings"
	"testing"
)

// TestRenderCookieProofSnippet_HappyPath confirms the snippet contains the
// loader script with the right data-domain, the relay's site_id, and the
// configured track path.
func TestRenderCookieProofSnippet_HappyPath(t *testing.T) {
	out := RenderCookieProofSnippet(
		"abcdef0123456789abcdef01",
		"example.com",
		"/t",
	)
	if !strings.Contains(out, `consent.example.com/loader.js`) {
		t.Errorf("missing CookieProof loader URL: %s", out)
	}
	if !strings.Contains(out, `data-domain="example.com"`) {
		t.Errorf("missing data-domain attribute: %s", out)
	}
	if !strings.Contains(out, `"abcdef0123456789abcdef01"`) {
		t.Errorf("missing site id in relay: %s", out)
	}
	if !strings.Contains(out, `"/t/consent"`) {
		t.Errorf("missing /t/consent track path: %s", out)
	}
	if !strings.Contains(out, `consent:update`) || !strings.Contains(out, `consent:gpc`) {
		t.Errorf("missing consent event listeners: %s", out)
	}
	if !strings.Contains(out, `sendBeacon`) || !strings.Contains(out, `fetch(TRACK`) {
		t.Errorf("missing transport (sendBeacon + fetch fallback): %s", out)
	}
}

// TestRenderCookieProofSnippet_DefaultsTrackPath confirms an empty trackPath
// falls back to /t.
func TestRenderCookieProofSnippet_DefaultsTrackPath(t *testing.T) {
	out := RenderCookieProofSnippet("aaaaaaaaaaaaaaaaaaaaaaaa", "example.com", "")
	if !strings.Contains(out, `"/t/consent"`) {
		t.Errorf("expected default /t/consent: %s", out)
	}
}

// TestRenderCookieProofSnippet_TrimsTrailingSlash makes sure /t/ doesn't
// produce //consent.
func TestRenderCookieProofSnippet_TrimsTrailingSlash(t *testing.T) {
	out := RenderCookieProofSnippet("aaaaaaaaaaaaaaaaaaaaaaaa", "example.com", "/t/")
	if strings.Contains(out, `"/t//consent"`) {
		t.Errorf("did not trim trailing slash: %s", out)
	}
	if !strings.Contains(out, `"/t/consent"`) {
		t.Errorf("expected /t/consent: %s", out)
	}
}

// TestRenderCookieProofSnippet_EmptyArgs returns "" when required args are
// missing so the layout doesn't emit a broken snippet.
func TestRenderCookieProofSnippet_EmptyArgs(t *testing.T) {
	if got := RenderCookieProofSnippet("", "example.com", "/t"); got != "" {
		t.Errorf("expected empty for missing siteID, got %q", got)
	}
	if got := RenderCookieProofSnippet("aaaa", "", "/t"); got != "" {
		t.Errorf("expected empty for missing domain, got %q", got)
	}
}
