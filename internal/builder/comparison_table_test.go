package builder

import (
	"strings"
	"testing"
)

func TestRenderComparisonTableBlock(t *testing.T) {
	data := map[string]any{
		"heading":    "Slab vs the obvious alternatives",
		"subheading": "What you get out of the box, no plugin marketplace required.",
		"us_label":   "Slab",
		"columns":    []any{"Lovable", "Webflow"},
		"rows": []any{
			map[string]any{
				"feature": "SSR Astro output",
				"us":      true,
				"values":  []any{false, true},
			},
			map[string]any{
				"feature": "BYO AI key",
				"us":      true,
				"values":  []any{false, false},
			},
			map[string]any{
				"feature": "Inspector built-in",
				"us":      "100/100",
				"values":  []any{"-", "-"},
			},
			map[string]any{
				// Row with fewer values than columns; missing cells render as dash.
				"feature": "EU-only data residency",
				"us":      true,
				"values":  []any{},
			},
		},
	}
	got := renderComparisonTableBlock(data)

	wantContains := []string{
		`class="block block--comparison_table"`,
		`<h2>Slab vs the obvious alternatives</h2>`,
		`<p class="subheading">What you get out of the box, no plugin marketplace required.</p>`,
		`<table class="comparison-table">`,
		`class="comparison-us"`,
		`>Slab<`,
		`>Lovable<`,
		`>Webflow<`,
		`>SSR Astro output<`,
		`comparison-cell--yes`,
		`comparison-cell--no`,
		`100/100`,
		`comparison-cell--text`,
		`aria-label="included"`,
		`aria-label="not included"`,
	}
	for _, w := range wantContains {
		if !strings.Contains(got, w) {
			t.Errorf("comparison_table output missing %q. got:\n%s", w, got)
		}
	}
}

func TestRenderComparisonTableBlock_SkipsRowsWithoutFeature(t *testing.T) {
	data := map[string]any{
		"columns": []any{"Lovable"},
		"rows": []any{
			map[string]any{"feature": "Valid", "us": true, "values": []any{false}},
			map[string]any{"us": true}, // no feature, must be skipped
		},
	}
	got := renderComparisonTableBlock(data)
	if !strings.Contains(got, "Valid") {
		t.Error("valid row missing")
	}
	// The render must not have an extra empty <th scope="row"></th> from the skipped row.
	if strings.Count(got, `<th scope="row">`) != 1 {
		t.Errorf("want exactly 1 row header, got %d. output:\n%s", strings.Count(got, `<th scope="row">`), got)
	}
}

func TestRenderComparisonTableBlock_NoColumns_StillRendersUs(t *testing.T) {
	data := map[string]any{
		"us_label": "Slab",
		"rows": []any{
			map[string]any{"feature": "Feature A", "us": true},
		},
	}
	got := renderComparisonTableBlock(data)
	if !strings.Contains(got, `>Slab<`) {
		t.Error("us column header missing when no competitors are listed")
	}
	if !strings.Contains(got, `>Feature A<`) {
		t.Error("feature row missing")
	}
}
