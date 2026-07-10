package builder

import (
	"strings"
	"testing"

	"github.com/bright-interaction/atomicsite/internal/store"
)

func TestRenderScrollReveal_HappyPath(t *testing.T) {
	out := renderScrollRevealBlock(map[string]any{
		"heading": "How it ships",
		"items": []any{
			map[string]any{"heading": "Step 1", "body": "First paragraph.\n\nSecond paragraph.", "image_position": "right"},
			map[string]any{"heading": "Step 2", "body": "Body 2", "image_position": "left"},
		},
	}, nil)

	wants := []string{
		`class="block block--scroll_reveal"`,
		`<h2>How it ships</h2>`,
		`<ol class="scroll-reveal-list">`,
		`is-image-right`,
		`is-image-left`,
		`<h3>Step 1</h3>`,
		`<p>First paragraph.</p>`,
		`<p>Second paragraph.</p>`,
		`<h3>Step 2</h3>`,
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in output:\n%s", w, out)
		}
	}
}

func TestRenderScrollReveal_NoItemsSkipsList(t *testing.T) {
	out := renderScrollRevealBlock(map[string]any{
		"heading": "Just a heading",
	}, nil)
	if strings.Contains(out, "scroll-reveal-list") {
		t.Errorf("empty items should skip the <ol>; got:\n%s", out)
	}
	if !strings.Contains(out, "<h2>Just a heading</h2>") {
		t.Errorf("heading should still render; got:\n%s", out)
	}
}

func TestRenderScrollReveal_MissingHeadingSkipsItem(t *testing.T) {
	out := renderScrollRevealBlock(map[string]any{
		"items": []any{
			map[string]any{"body": "no heading"},
			map[string]any{"heading": "Real one", "body": "ok"},
		},
	}, nil)
	if strings.Contains(out, "no heading") {
		t.Errorf("item without heading should be skipped; got:\n%s", out)
	}
	if !strings.Contains(out, "<h3>Real one</h3>") {
		t.Errorf("valid item should render; got:\n%s", out)
	}
}

func TestRenderScrollReveal_DefaultsToImageRight(t *testing.T) {
	out := renderScrollRevealBlock(map[string]any{
		"items": []any{map[string]any{"heading": "S"}},
	}, nil)
	if !strings.Contains(out, "is-image-right") {
		t.Errorf("default image_position should be right; got:\n%s", out)
	}
}

func TestRenderScrollReveal_DispatchedViaRenderDataBlock(t *testing.T) {
	out := renderDataBlock(
		"scroll_reveal",
		map[string]any{"items": []any{map[string]any{"heading": "S"}}},
		map[string]store.Medium{},
		"s",
	)
	if !strings.Contains(out, "block--scroll_reveal") {
		t.Errorf("renderDataBlock did not dispatch scroll_reveal; got:\n%s", out)
	}
}
