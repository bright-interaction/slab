package builder

import (
	"strings"
	"testing"

	"github.com/bright-interaction/atomicsite/internal/store"
)

func TestRenderMegaMenu_SimpleLink(t *testing.T) {
	out := renderMegaMenuBlock(map[string]any{
		"items": []any{
			map[string]any{"label": "Pricing", "href": "/pricing"},
		},
	})
	if !strings.Contains(out, `<a href="/pricing">Pricing</a>`) {
		t.Errorf("simple link should render as plain anchor; got:\n%s", out)
	}
	if !strings.Contains(out, `aria-label="Primary navigation"`) {
		t.Errorf("expected nav aria-label; got:\n%s", out)
	}
}

func TestRenderMegaMenu_DropdownWithColumns(t *testing.T) {
	out := renderMegaMenuBlock(map[string]any{
		"items": []any{
			map[string]any{
				"label": "Product",
				"columns": []any{
					map[string]any{
						"heading": "Features",
						"items": []any{
							map[string]any{"label": "Analytics", "href": "/analytics", "description": "Real-user CWV"},
							map[string]any{"label": "CMS", "href": "/cms"},
						},
					},
				},
			},
		},
	})

	wants := []string{
		`mega-menu-item--dropdown`,
		`<details>`,
		`<summary>Product`,
		`<p class="mega-column-heading">Features</p>`,
		`<a href="/analytics">`,
		`<span class="mega-link-label">Analytics</span>`,
		`<span class="mega-link-desc">Real-user CWV</span>`,
		`<a href="/cms">`,
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in output:\n%s", w, out)
		}
	}
}

func TestRenderMegaMenu_FeaturedSlot(t *testing.T) {
	out := renderMegaMenuBlock(map[string]any{
		"items": []any{
			map[string]any{
				"label": "Resources",
				"columns": []any{
					map[string]any{"items": []any{map[string]any{"label": "Docs", "href": "/docs"}}},
				},
				"featured_heading":  "What's new",
				"featured_body":     "We shipped 5 premium blocks.",
				"featured_cta_text": "Read",
				"featured_cta_url":  "/changelog",
			},
		},
	})
	wants := []string{
		`<aside class="mega-featured">`,
		`<p class="mega-featured-heading">What's new</p>`,
		`We shipped 5 premium blocks.`,
		`<a class="mega-featured-cta" href="/changelog">Read</a>`,
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in output:\n%s", w, out)
		}
	}
}

func TestRenderMegaMenu_EmptyItemsRendersNothing(t *testing.T) {
	out := renderMegaMenuBlock(map[string]any{})
	if out != "" {
		t.Errorf("empty items should produce no output; got: %s", out)
	}
}

func TestRenderMegaMenu_LabelOnlyFallsBackToText(t *testing.T) {
	out := renderMegaMenuBlock(map[string]any{
		"items": []any{
			map[string]any{"label": "Coming soon"},
		},
	})
	if !strings.Contains(out, `mega-menu-item--text`) {
		t.Errorf("label-only item should render as text; got:\n%s", out)
	}
	if !strings.Contains(out, `<span>Coming soon</span>`) {
		t.Errorf("expected plain span; got:\n%s", out)
	}
}

func TestRenderMegaMenu_DispatchedViaRenderDataBlock(t *testing.T) {
	out := renderDataBlock(
		"mega_menu",
		map[string]any{"items": []any{map[string]any{"label": "X", "href": "/x"}}},
		map[string]store.Medium{},
		"s",
	)
	if !strings.Contains(out, "block--mega_menu") {
		t.Errorf("renderDataBlock did not dispatch mega_menu; got:\n%s", out)
	}
}

func TestRenderMegaMenu_SkipsItemsWithoutLabel(t *testing.T) {
	out := renderMegaMenuBlock(map[string]any{
		"items": []any{
			map[string]any{"href": "/no-label"},
			map[string]any{"label": "Valid", "href": "/valid"},
		},
	})
	if strings.Contains(out, "/no-label") {
		t.Errorf("item without label must be skipped; got:\n%s", out)
	}
	if !strings.Contains(out, "/valid") {
		t.Errorf("valid item must render; got:\n%s", out)
	}
}
