package builder

import (
	"strings"
	"testing"
)

func TestRenderHeroGraphic_AllVariants(t *testing.T) {
	cases := []struct {
		name      string
		data      map[string]any
		wantClass string
		wantBody  []string
	}{
		{name: "mesh", wantClass: "hero-graphic hero-graphic--mesh", wantBody: []string{`aria-hidden="true"`}},
		{name: "pulse", wantClass: "hero-graphic hero-graphic--pulse", wantBody: []string{`aria-hidden="true"`}},
		{name: "gradient-orb", wantClass: "hero-graphic hero-graphic--gradient-orb", wantBody: []string{`aria-hidden="true"`}},
		{name: "monogram", data: map[string]any{"monogram_char": "N"}, wantClass: "hero-graphic--monogram", wantBody: []string{">N<"}},
		{name: "monogram-fallback-to-headline", data: map[string]any{"headline": "Sovereign"}, wantClass: "hero-graphic--monogram", wantBody: []string{">S<"}},
		{name: "audit-receipt", data: map[string]any{"audit_score": "98", "audit_score_max": "100", "audit_label": "Site audit"}, wantClass: "hero-graphic--audit-receipt", wantBody: []string{"<strong>98</strong>", "/100", "Site audit", "Industry average"}},
		{name: "globe-wire", wantClass: "hero-graphic hero-graphic--globe-wire", wantBody: []string{`<svg`, `class="hero-graphic__globe"`, `class="hero-graphic__grid"`, `class="hero-graphic__arcs"`, `class="hero-graphic__hub"`, `viewBox="0 0 400 400"`}},
	}
	for _, tc := range cases {
		variant := strings.SplitN(tc.name, "-fallback-", 2)[0]
		got := renderHeroGraphic(variant, tc.data)
		if !strings.Contains(got, tc.wantClass) {
			t.Errorf("[%s] missing class %q in: %s", tc.name, tc.wantClass, got)
		}
		for _, want := range tc.wantBody {
			if !strings.Contains(got, want) {
				t.Errorf("[%s] missing %q in: %s", tc.name, want, got)
			}
		}
	}
}

func TestRenderHeroGraphic_EmptyName(t *testing.T) {
	if got := renderHeroGraphic("", nil); got != "" {
		t.Errorf("renderHeroGraphic(\"\") = %q, want empty", got)
	}
}

func TestRenderHeroGraphic_UnknownName(t *testing.T) {
	if got := renderHeroGraphic("not-a-real-variant", nil); got != "" {
		t.Errorf("renderHeroGraphic(unknown) = %q, want empty so the build doesn't ship a broken graphic", got)
	}
}
