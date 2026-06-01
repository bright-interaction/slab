package builder

import (
	"strings"
	"testing"

	"github.com/brightinteraction/atomicsite/internal/store"
)

func TestRenderVideoHero_YouTubeUsesNoCookieDomain(t *testing.T) {
	out := renderVideoHeroBlock(map[string]any{
		"headline":     "Stop renting your stack",
		"video_source": "youtube",
		"video_id":     "abc123XYZ",
	}, nil)

	if !strings.Contains(out, "youtube-nocookie.com") {
		t.Errorf("YouTube embed must use youtube-nocookie.com (no pre-consent cookies); got:\n%s", out)
	}
	if !strings.Contains(out, "modestbranding=1") {
		t.Errorf("expected modestbranding parameter; got:\n%s", out)
	}
	if !strings.Contains(out, `<h1>Stop renting your stack</h1>`) {
		t.Errorf("expected headline in content layer; got:\n%s", out)
	}
}

func TestRenderVideoHero_VimeoBackgroundMode(t *testing.T) {
	out := renderVideoHeroBlock(map[string]any{
		"headline":     "Test",
		"video_source": "vimeo",
		"video_id":     "1234567",
	}, nil)
	if !strings.Contains(out, "player.vimeo.com/video/1234567") {
		t.Errorf("expected Vimeo embed URL; got:\n%s", out)
	}
	if !strings.Contains(out, "background=1") {
		t.Errorf("expected Vimeo background mode; got:\n%s", out)
	}
}

func TestRenderVideoHero_SelfHostedEmitsVideoTag(t *testing.T) {
	out := renderVideoHeroBlock(map[string]any{
		"headline":     "Self hosted",
		"video_source": "self_hosted",
		"video_id":     "mediaXYZ",
	}, nil)
	if !strings.Contains(out, "<video") {
		t.Errorf("expected <video> tag for self_hosted; got:\n%s", out)
	}
	if !strings.Contains(out, `<source src="/media/mediaXYZ/original.mp4"`) {
		t.Errorf("expected media URL pattern; got:\n%s", out)
	}
	if !strings.Contains(out, "muted") || !strings.Contains(out, "loop") || !strings.Contains(out, "playsinline") {
		t.Errorf("expected muted/loop/playsinline attrs for autoplay compliance; got:\n%s", out)
	}
}

func TestRenderVideoHero_OverlayClampedAndSerialized(t *testing.T) {
	out := renderVideoHeroBlock(map[string]any{
		"headline":        "Test",
		"video_source":    "youtube",
		"video_id":        "id",
		"overlay_opacity": "75",
	}, nil)
	if !strings.Contains(out, "--video-hero-overlay: 0.75") {
		t.Errorf("expected unitless 0.75 overlay var; got:\n%s", out)
	}

	// Out-of-range gets clamped to 100 (1.0).
	out2 := renderVideoHeroBlock(map[string]any{
		"headline":        "Test",
		"video_source":    "youtube",
		"video_id":        "id",
		"overlay_opacity": "999",
	}, nil)
	if !strings.Contains(out2, "--video-hero-overlay: 1.00") {
		t.Errorf("expected clamp to 1.00; got:\n%s", out2)
	}
}

func TestParseOverlayOpacity(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 35},
		{"0", 0},
		{"35", 35},
		{"100", 100},
		{"200", 100},
		{"-5", 35},
		{"abc", 35},
		{"45.5", 45},
	}
	for _, c := range cases {
		if got := parseOverlayOpacity(c.in); got != c.want {
			t.Errorf("parseOverlayOpacity(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRenderVideoHero_PosterImageIsLCPPriority(t *testing.T) {
	medium := store.Medium{
		ID:           "post1",
		SiteID:       "s",
		AltText:      "Poster",
		VariantsJson: `[{"path":"s/post1/1600.webp","width":1600,"height":900,"format":"webp"}]`,
	}
	out := renderVideoHeroBlock(map[string]any{
		"headline":        "T",
		"video_source":    "youtube",
		"video_id":        "id",
		"poster_image_id": "post1",
	}, map[string]store.Medium{"post1": medium})

	if !strings.Contains(out, `fetchpriority="high"`) {
		t.Errorf("poster image must be LCP-priority; got:\n%s", out)
	}
	if !strings.Contains(out, `loading="eager"`) {
		t.Errorf("poster image must be loading=eager; got:\n%s", out)
	}
}

func TestRenderVideoHero_DispatchedViaRenderDataBlock(t *testing.T) {
	out := renderDataBlock(
		"video_hero",
		map[string]any{"headline": "T", "video_source": "youtube", "video_id": "id"},
		map[string]store.Medium{},
		"s",
	)
	if !strings.Contains(out, "block--video_hero") {
		t.Errorf("renderDataBlock did not dispatch video_hero; got:\n%s", out)
	}
}
