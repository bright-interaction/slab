package builder

import (
	"strings"
	"testing"

	"github.com/bright-interaction/slab/internal/store"
)

// TestRenderPictureWithPriority_HeroEmitsEagerFetchPriority confirms hero
// callers get fetchpriority=high + loading=eager so the LCP element is not
// deferred. Closes the markup-author dependency that left LCP at the mercy
// of whoever wrote the block JSON.
func TestRenderPictureWithPriority_HeroEmitsEagerFetchPriority(t *testing.T) {
	m := store.Medium{
		ID:           "med1",
		SiteID:       "site1",
		AltText:      "Hero image",
		VariantsJson: `[{"path":"site1/med1/1600.webp","width":1600,"height":900,"format":"webp"},{"path":"site1/med1/1600.jpg","width":1600,"height":900,"format":"jpeg"}]`,
	}

	priority := RenderPictureWithPriority(m, "hero-image", true)
	if !strings.Contains(priority, `fetchpriority="high"`) {
		t.Errorf("priority render missing fetchpriority=high: %s", priority)
	}
	if !strings.Contains(priority, `loading="eager"`) {
		t.Errorf("priority render missing loading=eager: %s", priority)
	}
	if strings.Contains(priority, `loading="lazy"`) {
		t.Errorf("priority render unexpectedly contains loading=lazy: %s", priority)
	}

	lazy := RenderPictureWithPriority(m, "card-image", false)
	if !strings.Contains(lazy, `loading="lazy"`) {
		t.Errorf("non-priority render missing loading=lazy: %s", lazy)
	}
	if strings.Contains(lazy, `fetchpriority`) {
		t.Errorf("non-priority render unexpectedly contains fetchpriority: %s", lazy)
	}
}

// TestRenderPicture_DefaultsToLazy keeps the bare RenderPicture API
// byte-identical with its pre-priority behavior for every existing callsite.
func TestRenderPicture_DefaultsToLazy(t *testing.T) {
	m := store.Medium{
		ID:           "med2",
		SiteID:       "s2",
		AltText:      "Body image",
		VariantsJson: `[{"path":"s2/med2/800.webp","width":800,"height":600,"format":"webp"}]`,
	}
	out := RenderPicture(m, "")
	if !strings.Contains(out, `loading="lazy"`) {
		t.Errorf("default RenderPicture should be lazy: %s", out)
	}
	if strings.Contains(out, `fetchpriority`) {
		t.Errorf("default RenderPicture should not set fetchpriority: %s", out)
	}
}

// TestRenderMediaImg_FallbackRespectsePriority makes sure the fallback path
// (media row missing in the lookup map) also gets the right loading attrs,
// not just the variant-rich happy path.
func TestRenderMediaImg_FallbackRespectsePriority(t *testing.T) {
	priority := renderMediaImgWithPriority("missing-media", "alt", "fb", "", nil, true)
	if !strings.Contains(priority, `fetchpriority="high"`) {
		t.Errorf("fallback priority missing fetchpriority=high: %s", priority)
	}
	if !strings.Contains(priority, `loading="eager"`) {
		t.Errorf("fallback priority missing loading=eager: %s", priority)
	}

	lazy := renderMediaImg("missing-media", "alt", "fb", "", nil)
	if !strings.Contains(lazy, `loading="lazy"`) {
		t.Errorf("fallback default missing loading=lazy: %s", lazy)
	}
}
