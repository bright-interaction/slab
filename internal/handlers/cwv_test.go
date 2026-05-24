package handlers

import (
	"testing"
)

// Sprint 5 quick-win (2026-05-24): CWV threshold tests. Pure-function
// rating helper; locks Google's published bands so a publisher-side
// web-vitals.js library bumping the bands later doesn't drift the
// dashboard view away from the documented thresholds.

func TestCWVRatingFor_LCP(t *testing.T) {
	cases := []struct {
		v    float64
		want string
	}{
		{1000, "good"},
		{2500, "good"},
		{2501, "needs-improvement"},
		{4000, "needs-improvement"},
		{4001, "poor"},
		{8000, "poor"},
	}
	for _, c := range cases {
		if got := cwvRatingFor("LCP", c.v); got != c.want {
			t.Errorf("LCP %v: got %q, want %q", c.v, got, c.want)
		}
	}
}

func TestCWVRatingFor_INP(t *testing.T) {
	if cwvRatingFor("INP", 150) != "good" {
		t.Errorf("INP 150 should be good")
	}
	if cwvRatingFor("INP", 300) != "needs-improvement" {
		t.Errorf("INP 300 should be needs-improvement")
	}
	if cwvRatingFor("INP", 700) != "poor" {
		t.Errorf("INP 700 should be poor")
	}
}

func TestCWVRatingFor_CLS(t *testing.T) {
	if cwvRatingFor("CLS", 0.05) != "good" {
		t.Errorf("CLS 0.05 should be good")
	}
	if cwvRatingFor("CLS", 0.2) != "needs-improvement" {
		t.Errorf("CLS 0.2 should be needs-improvement")
	}
	if cwvRatingFor("CLS", 0.5) != "poor" {
		t.Errorf("CLS 0.5 should be poor")
	}
}

func TestCWVRatingFor_FCPAndTTFB(t *testing.T) {
	if cwvRatingFor("FCP", 1500) != "good" {
		t.Errorf("FCP 1500 should be good")
	}
	if cwvRatingFor("TTFB", 700) != "good" {
		t.Errorf("TTFB 700 should be good")
	}
	if cwvRatingFor("TTFB", 2000) != "poor" {
		t.Errorf("TTFB 2000 should be poor")
	}
}

func TestCWVRatingFor_UnknownMetric(t *testing.T) {
	if cwvRatingFor("XYZ", 100) != "" {
		t.Errorf("unknown metric should return empty rating")
	}
}

func TestCWVValidMetricsEnum(t *testing.T) {
	for _, m := range []string{"LCP", "INP", "CLS", "FCP", "TTFB"} {
		if !validCWVMetrics[m] {
			t.Errorf("expected %q to be a valid metric", m)
		}
	}
	if validCWVMetrics["xyz"] {
		t.Errorf("unknown metric should be rejected")
	}
}

func TestCWVValidRatingsEnum(t *testing.T) {
	for _, r := range []string{"", "good", "needs-improvement", "poor"} {
		if !validCWVRatings[r] {
			t.Errorf("expected %q to be a valid rating", r)
		}
	}
	if validCWVRatings["unknown"] {
		t.Errorf("unknown rating should be rejected (the handler maps to empty)")
	}
}
