package builder

import (
	"strings"
	"testing"

	"github.com/bright-interaction/atomicsite/internal/store"
)

func TestRenderTestimonialWall_WallLayoutDefault(t *testing.T) {
	data := map[string]any{
		"heading": "What teams say",
		"items": []any{
			map[string]any{
				"quote":          "Saved us a quarter of agency budget.",
				"author_name":    "Erika Lindqvist",
				"author_role":    "COO",
				"author_company": "Volta Logistics",
				"rating":         "5",
			},
		},
	}
	out := renderTestimonialWallBlock(data, nil)

	wants := []string{
		`class="block block--testimonial_wall block--testimonial_wall--wall"`,
		`<h2>What teams say</h2>`,
		`<ul class="testimonial-wall-list">`,
		`<blockquote>Saved us a quarter of agency budget.</blockquote>`,
		`Erika Lindqvist`,
		`COO, Volta Logistics`,
		`aria-label="Rated 5 out of 5"`,
		`<span class="is-filled" aria-hidden="true">★</span>`,
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in output:\n%s", w, out)
		}
	}
}

func TestRenderTestimonialWall_CarouselLayout(t *testing.T) {
	data := map[string]any{
		"layout": "carousel",
		"items": []any{
			map[string]any{"quote": "Q1", "author_name": "A1"},
			map[string]any{"quote": "Q2", "author_name": "A2"},
		},
	}
	out := renderTestimonialWallBlock(data, nil)
	if !strings.Contains(out, `block--testimonial_wall--carousel`) {
		t.Errorf("expected carousel modifier class; got:\n%s", out)
	}
	if !strings.Contains(out, `<ul class="testimonial-carousel-list">`) {
		t.Errorf("expected carousel list class; got:\n%s", out)
	}
}

func TestRenderTestimonialWall_SingleFeaturedLayout(t *testing.T) {
	data := map[string]any{
		"layout": "single_featured",
		"items": []any{
			map[string]any{"quote": "Q", "author_name": "A"},
		},
	}
	out := renderTestimonialWallBlock(data, nil)
	if !strings.Contains(out, `block--testimonial_wall--single_featured`) {
		t.Errorf("expected single_featured modifier; got:\n%s", out)
	}
	if !strings.Contains(out, `<ul class="testimonial-featured-list">`) {
		t.Errorf("expected single-featured list class; got:\n%s", out)
	}
}

func TestRenderTestimonialWall_UnknownLayoutFallsBackToWall(t *testing.T) {
	data := map[string]any{
		"layout": "rocket-ship",
		"items":  []any{map[string]any{"quote": "Q", "author_name": "A"}},
	}
	out := renderTestimonialWallBlock(data, nil)
	if !strings.Contains(out, `block--testimonial_wall--wall`) {
		t.Errorf("expected fallback to wall layout; got:\n%s", out)
	}
}

func TestRenderTestimonialWall_NoQuoteSkipsCard(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"author_name": "A"},
			map[string]any{"quote": "Real one", "author_name": "B"},
		},
	}
	out := renderTestimonialWallBlock(data, nil)
	if strings.Contains(out, "A") {
		t.Errorf("quote-less card should not render its author; got:\n%s", out)
	}
	if !strings.Contains(out, "Real one") {
		t.Errorf("valid card should render; got:\n%s", out)
	}
}

func TestRenderTestimonialWall_FooterLinkOptional(t *testing.T) {
	data := map[string]any{
		"items":      []any{map[string]any{"quote": "Q", "author_name": "A"}},
		"footer":     "Read 200+ more reviews",
		"footer_url": "/reviews",
	}
	out := renderTestimonialWallBlock(data, nil)
	if !strings.Contains(out, `<a class="testimonial-footer" href="/reviews">Read 200+ more reviews</a>`) {
		t.Errorf("expected footer anchor; got:\n%s", out)
	}

	data2 := map[string]any{
		"items":  []any{map[string]any{"quote": "Q", "author_name": "A"}},
		"footer": "Just text, no link",
	}
	out2 := renderTestimonialWallBlock(data2, nil)
	if !strings.Contains(out2, `<p class="testimonial-footer">Just text, no link</p>`) {
		t.Errorf("expected footer paragraph when no URL; got:\n%s", out2)
	}
}

func TestRenderTestimonialWall_DispatchedViaRenderDataBlock(t *testing.T) {
	out := renderDataBlock(
		"testimonial_wall",
		map[string]any{"items": []any{map[string]any{"quote": "Q", "author_name": "A"}}},
		map[string]store.Medium{},
		"site_id",
	)
	if !strings.Contains(out, `block--testimonial_wall`) {
		t.Errorf("renderDataBlock did not dispatch testimonial_wall; got:\n%s", out)
	}
}

func TestParseRating(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"   ", 0},
		{"5", 5},
		{"4", 4},
		{"0", 0},
		{"9", 5},
		{"5 stars", 5},
		{"4/5", 4},
		{"abc", 0},
	}
	for _, c := range cases {
		if got := parseRating(c.in); got != c.want {
			t.Errorf("parseRating(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
