package builder

import (
	"strings"
	"testing"

	"github.com/bright-interaction/slab/internal/store"
)

func TestRenderTabs_HappyPath(t *testing.T) {
	data := map[string]any{
		"heading": "Two ways to use it",
		"items": []any{
			map[string]any{"label": "Marketing teams", "heading": "Ship landing pages", "body": "Body for marketing."},
			map[string]any{"label": "Product teams", "heading": "Document features", "body": "Body for product."},
		},
	}
	out := renderTabsBlock(data, nil)

	wants := []string{
		`class="block block--tabs"`,
		`<h2>Two ways to use it</h2>`,
		`<input type="radio" class="tab-radio"`,
		` checked `,
		`role="tablist"`,
		`role="tab"`,
		`role="tabpanel"`,
		`>Marketing teams</label>`,
		`>Product teams</label>`,
		`<h3>Ship landing pages</h3>`,
		`<p>Body for marketing.</p>`,
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in output:\n%s", w, out)
		}
	}
}

func TestRenderTabs_FirstTabCheckedByDefault(t *testing.T) {
	out := renderTabsBlock(map[string]any{
		"items": []any{
			map[string]any{"label": "A"},
			map[string]any{"label": "B"},
			map[string]any{"label": "C"},
		},
	}, nil)
	// First radio carries " checked", later ones do not.
	firstIdx := strings.Index(out, ` id="`)
	if firstIdx == -1 {
		t.Fatalf("no radio id rendered: %s", out)
	}
	// The first radio should have checked; count occurrences of checked.
	if strings.Count(out, ` checked `) != 1 {
		t.Errorf("expected exactly one checked radio; got %d in:\n%s", strings.Count(out, ` checked `), out)
	}
}

func TestRenderTabs_RespectsDefaultTab(t *testing.T) {
	out := renderTabsBlock(map[string]any{
		"default_tab": "1",
		"items": []any{
			map[string]any{"label": "A"},
			map[string]any{"label": "B"},
		},
	}, nil)
	// Second radio (index 1) carries checked.
	if !strings.Contains(out, `id="tabs-`) {
		t.Fatalf("group id missing: %s", out)
	}
	// Pull out the two radio lines; check the second has checked.
	lines := strings.Split(out, "\n")
	radioLines := []string{}
	for _, l := range lines {
		if strings.Contains(l, `class="tab-radio"`) {
			radioLines = append(radioLines, l)
		}
	}
	if len(radioLines) != 2 {
		t.Fatalf("expected 2 radio lines, got %d", len(radioLines))
	}
	if strings.Contains(radioLines[0], "checked") {
		t.Errorf("first radio should not be checked when default_tab=1; got: %s", radioLines[0])
	}
	if !strings.Contains(radioLines[1], "checked") {
		t.Errorf("second radio should be checked when default_tab=1; got: %s", radioLines[1])
	}
}

func TestRenderTabs_EmptyItemsRendersNothing(t *testing.T) {
	out := renderTabsBlock(map[string]any{"heading": "Empty"}, nil)
	if out != "" {
		t.Errorf("expected empty output for no items; got: %s", out)
	}
}

func TestRenderTabs_StableGroupID(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"label": "Alpha"},
			map[string]any{"label": "Beta"},
		},
	}
	a := tabsGroupID(data)
	b := tabsGroupID(data)
	if a != b {
		t.Errorf("group ID must be stable for same content: %q vs %q", a, b)
	}

	// Different labels -> different ID.
	data2 := map[string]any{
		"items": []any{
			map[string]any{"label": "Alpha"},
			map[string]any{"label": "Gamma"},
		},
	}
	c := tabsGroupID(data2)
	if a == c {
		t.Errorf("different content should hash to different IDs: both got %q", a)
	}
}

func TestRenderTabs_DispatchedViaRenderDataBlock(t *testing.T) {
	out := renderDataBlock(
		"tabs",
		map[string]any{"items": []any{map[string]any{"label": "A", "body": "B"}}},
		map[string]store.Medium{},
		"s",
	)
	if !strings.Contains(out, "block--tabs") {
		t.Errorf("renderDataBlock did not dispatch tabs; got:\n%s", out)
	}
}

func TestFnv1aHash_Stable(t *testing.T) {
	// Smoke test that the constants compile correctly and produce
	// non-trivial output for typical inputs.
	if fnv1aHash("") == 0 {
		t.Error("FNV-1a of empty string should equal the offset (non-zero)")
	}
	if fnv1aHash("hello") == fnv1aHash("world") {
		t.Error("FNV-1a collision on hello/world: constants are wrong")
	}
}
