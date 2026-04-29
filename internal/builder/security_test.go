package builder

import (
	"strings"
	"testing"
)

// TestBuildCSP_DefaultsKeepFrameSrcAbsent confirms a site with no allowed
// embeds doesn't get an explicit frame-src directive (browsers fall back to
// default-src 'self', which is the safe default). Adding an unused frame-src
// directive would trip the "weak CSP" eval check.
func TestBuildCSP_DefaultsKeepFrameSrcAbsent(t *testing.T) {
	out := buildCSP(CSPOptions{})
	if strings.Contains(out, "frame-src") {
		t.Errorf("expected no frame-src on default CSP, got: %s", out)
	}
	if !strings.Contains(out, "frame-ancestors 'none'") {
		t.Errorf("expected frame-ancestors 'none' default, got: %s", out)
	}
	if !strings.Contains(out, "upgrade-insecure-requests") {
		t.Errorf("expected upgrade-insecure-requests, got: %s", out)
	}
}

// TestBuildCSP_FrameKindGoesToFrameSrc is the cal.com / YouTube case the
// user explicitly asked for. A domain registered with kind=frame must end
// up in frame-src and NOT in script-src.
func TestBuildCSP_FrameKindGoesToFrameSrc(t *testing.T) {
	out := buildCSP(CSPOptions{
		Allow: AllowedDomains{
			Frame: []string{"https://cal.com", "https://youtube.com"},
		},
	})
	if !strings.Contains(out, "frame-src 'self' https://cal.com https://youtube.com") {
		t.Errorf("expected frame-src with cal.com + youtube.com, got: %s", out)
	}
	if strings.Contains(out, "script-src 'self' https://cal.com") {
		t.Errorf("frame-only domain leaked into script-src, got: %s", out)
	}
}

// TestBuildCSP_ScriptKindFanOutToConnectSrc confirms the historical
// behaviour: a script-kind domain feeds both script-src and connect-src so
// loading a third-party script that XHRs back to its origin still works.
func TestBuildCSP_ScriptKindFanOutToConnectSrc(t *testing.T) {
	out := buildCSP(CSPOptions{
		Allow: AllowedDomains{
			Script:  []string{"https://js.stripe.com"},
			Connect: []string{"https://js.stripe.com"},
		},
	})
	if !strings.Contains(out, "script-src 'self' https://js.stripe.com") {
		t.Errorf("expected script-src with js.stripe.com, got: %s", out)
	}
	if !strings.Contains(out, "connect-src 'self' https://js.stripe.com") {
		t.Errorf("expected connect-src with js.stripe.com, got: %s", out)
	}
}

// TestBuildCSP_ImageKindGoesToImgSrc covers the CDN host case. The default
// img-src already wildcards https:, so we still want named hosts to land in
// the directive (otherwise an operator-set restrictive img-src override
// would silently lose the host). Append-only.
func TestBuildCSP_ImageKindGoesToImgSrc(t *testing.T) {
	out := buildCSP(CSPOptions{
		Allow: AllowedDomains{
			Image: []string{"https://res.cloudinary.com"},
		},
	})
	if !strings.Contains(out, "img-src 'self' data: https: https://res.cloudinary.com") {
		t.Errorf("expected img-src with cloudinary appended, got: %s", out)
	}
}

// TestBuildCSP_MediaKindOnlyEmitsWhenSet mirrors the frame-src behaviour:
// no entries -> no directive emitted, so simple sites stay clean.
func TestBuildCSP_MediaKindOnlyEmitsWhenSet(t *testing.T) {
	noMedia := buildCSP(CSPOptions{})
	if strings.Contains(noMedia, "media-src") {
		t.Errorf("expected no media-src on default, got: %s", noMedia)
	}
	withMedia := buildCSP(CSPOptions{Allow: AllowedDomains{Media: []string{"https://vimeo.com"}}})
	if !strings.Contains(withMedia, "media-src 'self' https://vimeo.com") {
		t.Errorf("expected media-src with vimeo, got: %s", withMedia)
	}
}

// TestBuildCSP_FrameAncestorsOverride confirms operators can flip from the
// strict 'none' default to allow embedding. The eval engine warns when the
// header is missing entirely; this lets a site opt into being framed by a
// portal app without losing the rest of the CSP.
func TestBuildCSP_FrameAncestorsOverride(t *testing.T) {
	out := buildCSP(CSPOptions{FrameAncestors: "'self' https://portal.example.com"})
	if !strings.Contains(out, "frame-ancestors 'self' https://portal.example.com") {
		t.Errorf("expected operator-supplied frame-ancestors, got: %s", out)
	}
	if strings.Contains(out, "frame-ancestors 'none'") {
		t.Errorf("default frame-ancestors leaked through override, got: %s", out)
	}
}

// TestBuildCSP_ExtraDirectivesAppendedAndTrimmed confirms operator paste
// that ends in a semicolon doesn't double up. Browsers tolerate "a; b; ;"
// but it looks broken in inspector tools.
func TestBuildCSP_ExtraDirectivesAppendedAndTrimmed(t *testing.T) {
	out := buildCSP(CSPOptions{ExtraDirectives: "report-uri /csp-report;"})
	if !strings.Contains(out, "; report-uri /csp-report") {
		t.Errorf("expected report-uri appended, got: %s", out)
	}
	if strings.HasSuffix(out, "; ") || strings.HasSuffix(out, ";") {
		t.Errorf("trailing semicolon leaked into output: %s", out)
	}
}

// TestBuildCSP_AllKindFansOut is the convenience case for "trust everything
// at this domain". A single row with kind=all should reach script + connect
// + frame + image + media simultaneously.
func TestBuildCSP_AllKindFansOut(t *testing.T) {
	allow := AllowedDomains{}
	host := "https://embed.example.com"
	allow.Script = append(allow.Script, host)
	allow.Connect = append(allow.Connect, host)
	allow.Frame = append(allow.Frame, host)
	allow.Image = append(allow.Image, host)
	allow.Media = append(allow.Media, host)
	out := buildCSP(CSPOptions{Allow: allow})

	for _, want := range []string{
		"script-src 'self' " + host,
		"connect-src 'self' " + host,
		"frame-src 'self' " + host,
		"img-src 'self' data: https: " + host,
		"media-src 'self' " + host,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("kind=all missed %q in output: %s", want, out)
		}
	}
}

// TestCSPForMeta_StripsHeaderOnlyDirectives makes sure the meta-tag
// fallback still strips frame-ancestors / sandbox / report-uri / report-to
// / upgrade-insecure-requests now that the per-kind directives flow through
// the same string. Browsers ignore those in <meta http-equiv> form so
// leaving them in produces noisy console warnings.
func TestCSPForMeta_StripsHeaderOnlyDirectives(t *testing.T) {
	full := buildCSP(CSPOptions{
		Allow:          AllowedDomains{Frame: []string{"https://cal.com"}},
		FrameAncestors: "'self'",
	})
	meta := CSPForMeta(full)

	mustHave := []string{"script-src", "frame-src 'self' https://cal.com", "img-src"}
	for _, w := range mustHave {
		if !strings.Contains(meta, w) {
			t.Errorf("meta CSP missing %q: %s", w, meta)
		}
	}
	mustNotHave := []string{"frame-ancestors", "upgrade-insecure-requests", "sandbox", "report-uri"}
	for _, w := range mustNotHave {
		if strings.Contains(meta, w) {
			t.Errorf("meta CSP should not include %q: %s", w, meta)
		}
	}
}
