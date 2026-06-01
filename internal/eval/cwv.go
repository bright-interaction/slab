package eval

import (
	"context"
	"fmt"
	"math"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// cwvMinSamples is the floor below which a metric is reported as Info
// (no scoring impact). A brand-new site with no traffic should not be
// penalized; static markup grades it instead. Above the floor, p75 is
// trusted and graded against Google's published thresholds.
const cwvMinSamples = 10

// cwvWindow is the lookback for p75. Matches Google's PageSpeed Insights
// 28d window only loosely; 7d is what we collect and what the dashboard
// already surfaces.
const cwvWindow = "-7 days"

// cwvThresholds holds the good/poor breakpoints per metric. Source: Google,
// 2026-published values. Mirrors cwvRatingFor in handlers/analytics.go to
// keep the grader and the dashboard in lockstep.
type cwvThresholds struct {
	good   float64
	poor   float64
	unit   string
	weight int
}

var cwvMetricThresholds = map[string]cwvThresholds{
	"LCP":  {good: 2500, poor: 4000, unit: "ms", weight: 4},
	"INP":  {good: 200, poor: 500, unit: "ms", weight: 4},
	"CLS":  {good: 0.1, poor: 0.25, unit: "", weight: 3},
	"FCP":  {good: 1800, poor: 3000, unit: "ms", weight: 2},
	"TTFB": {good: 800, poor: 1800, unit: "ms", weight: 2},
}

// cwvMetricOrder fixes the emission order so the UI renders the most
// SEO-impactful metrics first (LCP/INP/CLS are the three official Core
// Web Vitals; FCP and TTFB are supplementary diagnostics).
var cwvMetricOrder = []string{"LCP", "INP", "CLS", "FCP", "TTFB"}

// RunCWVChecks queries measured Core Web Vitals from the analytics DB and
// emits one check per metric (LCP, INP, CLS, FCP, TTFB) based on the p75
// sample against Google's thresholds. Metrics with fewer than cwvMinSamples
// samples are emitted as Info, so a new site with no traffic keeps its
// static-markup-derived perf score and is not penalized for missing data.
//
// This is what closes the moat-hole vs Lovable: the static eval can call a
// site A+ on perf even when shipping a 500KB blocking font; folding the
// real-user beacon data in makes the grade reflect what users measured.
func RunCWVChecks(ctx context.Context, queries *store.Queries, siteID string) []CheckResult {
	if queries == nil || siteID == "" {
		return nil
	}
	var checks []CheckResult
	for _, metric := range cwvMetricOrder {
		rows, err := queries.ListCWVValuesForWindow(ctx, store.ListCWVValuesForWindowParams{
			SiteID:   siteID,
			Metric:   metric,
			Device:   "",
			Column4:  "",
			Datetime: cwvWindow,
		})
		if err != nil {
			continue
		}
		t := cwvMetricThresholds[metric]
		name := fmt.Sprintf("Measured %s (p75, 7d)", metric)
		if len(rows) < cwvMinSamples {
			checks = append(checks, Info(name, "cwv",
				fmt.Sprintf("%d samples (need %d for grading)", len(rows), cwvMinSamples)))
			continue
		}
		// Rows come back ORDER BY value ASC; p75 is nearest-rank.
		idx := int(math.Floor(0.75 * float64(len(rows))))
		if idx >= len(rows) {
			idx = len(rows) - 1
		}
		p75 := rows[idx].Value
		switch {
		case p75 <= t.good:
			checks = append(checks, Pass(name, "cwv", t.weight,
				fmt.Sprintf("p75 %s (good, n=%d)", formatCWVValue(metric, p75), len(rows))))
		case p75 <= t.poor:
			checks = append(checks, Fail(name, "cwv", t.weight, SeverityWarning,
				fmt.Sprintf("p75 %s (needs improvement, n=%d)", formatCWVValue(metric, p75), len(rows)),
				cwvFixHint(metric)))
		default:
			checks = append(checks, Fail(name, "cwv", t.weight, SeverityError,
				fmt.Sprintf("p75 %s (poor, n=%d)", formatCWVValue(metric, p75), len(rows)),
				cwvFixHint(metric)))
		}
	}
	return checks
}

func formatCWVValue(metric string, v float64) string {
	if cwvMetricThresholds[metric].unit == "ms" {
		return fmt.Sprintf("%.0fms", v)
	}
	return fmt.Sprintf("%.3f", v)
}

func cwvFixHint(metric string) string {
	switch metric {
	case "LCP":
		return "Preload the LCP image with fetchpriority=high. Serve AVIF/WebP. Reduce hero CSS and font-blocking work."
	case "INP":
		return "Reduce main-thread work. Defer non-critical JS. Split long tasks. Audit event handlers."
	case "CLS":
		return "Reserve space for images, iframes, and banners using width+height or aspect-ratio. Avoid late-injected DOM."
	case "FCP":
		return "Inline critical CSS. Defer non-critical scripts. Preconnect to required origins."
	case "TTFB":
		return "Cache HTML at the edge. Warm DB queries. Move heavy work off the request path."
	}
	return ""
}
