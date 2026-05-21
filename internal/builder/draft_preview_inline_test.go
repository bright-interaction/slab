package builder

import (
	"strings"
	"testing"
)

// TestRenderRegisteredComponentInline_DottedGlobeHero locks the behaviour
// the admin preview iframe needs: registered Astro templates render to
// real HTML in the editor (no placeholder), with prop substitution,
// attribute context handled correctly, and embedded <style>/<script>
// blocks preserved verbatim so the canvas globe actually paints.
func TestRenderRegisteredComponentInline_DottedGlobeHero(t *testing.T) {
	tpl := `---
const { eyebrow, headline_first, headline_em, headline_last, subheading, cta_primary_text, cta_primary_href, cta_secondary_text, cta_secondary_href, aria_label } = Astro.props;
---
<section class="bi-globe-hero-3d">
  <p class="bi-globe-hero-3d__eyebrow">{eyebrow}</p>
  <h1>{headline_first}<em>{headline_em}</em>{headline_last}</h1>
  <p>{subheading}</p>
  <a href={cta_primary_href} class="btn-primary">{cta_primary_text}</a>
  <a href={cta_secondary_href} class="btn-secondary">{cta_secondary_text}</a>
  <canvas aria-label={aria_label || "Animated globe"}></canvas>
</section>
<style>.bi-globe-hero-3d { display: grid; }</style>
<script>(function(){ const x = ` + "`hello ${name}`" + `; })();</script>
`
	dataJSON := `{"component":"dotted-globe-hero","props":{"eyebrow":"EU JURISDICTION","headline_first":"Stop renting your business. Own ","headline_em":"it","headline_last":".","subheading":"European jurisdiction by default.","cta_primary_text":"Calculate your savings","cta_primary_href":"#calculate","cta_secondary_text":"See the swap","cta_secondary_href":"#swap","aria_label":"Globe with European hubs"}}`

	got, ok := renderRegisteredComponentInline(tpl, dataJSON)
	if !ok {
		t.Fatalf("renderRegisteredComponentInline returned ok=false")
	}

	// Prop interpolation in text position.
	mustContain(t, got, `>EU JURISDICTION<`, "eyebrow text not substituted")
	mustContain(t, got, `>Stop renting your business. Own `, "headline_first not substituted")
	mustContain(t, got, `<em>it</em>`, "headline_em not substituted")
	mustContain(t, got, `>European jurisdiction by default.<`, "subheading not substituted")
	mustContain(t, got, `>Calculate your savings<`, "cta_primary_text not substituted")

	// Attribute interpolation must be quoted so values with spaces survive.
	mustContain(t, got, `href="#calculate"`, "cta_primary_href attr not quoted")
	mustContain(t, got, `href="#swap"`, "cta_secondary_href attr not quoted")
	mustContain(t, got, `aria-label="Globe with European hubs"`, "aria_label attr not quoted")

	// Embedded <style> and <script> must survive verbatim so the preview
	// runs the real animation, not a stub.
	mustContain(t, got, `<style>.bi-globe-hero-3d { display: grid; }</style>`, "style block lost")
	mustContain(t, got, "<script>(function(){ const x = `hello ${name}`; })();</script>", "script block lost or template literal escaped")

	// JSX-only frontmatter MUST be stripped.
	if strings.Contains(got, "Astro.props") {
		t.Errorf("frontmatter leaked into rendered HTML: %s", got)
	}
}

// TestRenderRegisteredComponentInline_DefaultFallback confirms that the
// `{name || "default"}` JSX pattern falls back to the literal string when
// the prop is absent.
func TestRenderRegisteredComponentInline_DefaultFallback(t *testing.T) {
	tpl := `---
const { aria_label } = Astro.props;
---
<canvas aria-label={aria_label || "Default label"}></canvas>
`
	got, ok := renderRegisteredComponentInline(tpl, `{"props":{}}`)
	if !ok {
		t.Fatalf("ok=false")
	}
	mustContain(t, got, `aria-label="Default label"`, "default fallback not applied")
}

// TestRenderRegisteredComponentInline_HTMLEscaping confirms that an
// attacker-controlled prop string can't break out of the rendered
// markup. Critical for the admin preview because draft content is
// agent- and operator-authored.
func TestRenderRegisteredComponentInline_HTMLEscaping(t *testing.T) {
	tpl := `---
const { eyebrow } = Astro.props;
---
<p>{eyebrow}</p>
`
	got, ok := renderRegisteredComponentInline(tpl, `{"props":{"eyebrow":"<script>alert(1)</script>"}}`)
	if !ok {
		t.Fatalf("ok=false")
	}
	if strings.Contains(got, "<script>alert(1)</script>") {
		t.Errorf("unescaped <script> leaked: %s", got)
	}
	mustContain(t, got, `&lt;script&gt;alert(1)&lt;/script&gt;`, "expected HTML-escaped output")
}

// TestRenderRegisteredComponentInline_DataDirectAsProps confirms that
// when the block stores props flat (no nested "props" key) the renderer
// still picks them up. Legacy blocks predate the `props` envelope.
func TestRenderRegisteredComponentInline_DataDirectAsProps(t *testing.T) {
	tpl := `---
const { eyebrow } = Astro.props;
---
<p>{eyebrow}</p>
`
	got, ok := renderRegisteredComponentInline(tpl, `{"component":"x","eyebrow":"FLAT"}`)
	if !ok {
		t.Fatalf("ok=false")
	}
	mustContain(t, got, `>FLAT<`, "flat prop not picked up")
}

func mustContain(t *testing.T, haystack, needle, msg string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: expected substring %q in:\n%s", msg, needle, haystack)
	}
}
