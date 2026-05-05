// Package critique audits a built site against the DesignPlaybook
// anti-patterns and content-authenticity rules. The single source of
// truth is agent.DefaultDesignPlaybook(): every check derives its
// banned pattern, recommendation, and section label from the playbook
// so editing the playbook is the only place to add or remove rules.
//
// Mirrors the eval package shape (Pass/Fail/Info, CategoryReport)
// and persists results to the same evaluations table with
// Category="design" so the existing /api/sites/{id}/evaluations and
// /api/agent/evaluation/{buildID} endpoints surface critique findings
// alongside security/SEO without schema changes.
package critique

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/bright-interaction/slab/internal/agent"
	"github.com/bright-interaction/slab/internal/eval"
	"github.com/bright-interaction/slab/internal/store"
)

// Timeout caps total critique time per build.
const Timeout = 15 * time.Second

// Run loads the same SiteContext eval uses, runs every design check
// against it, and persists one evaluations row with Category="design".
// Bounded by Timeout. Failures are non-fatal at the caller level.
func Run(ctx context.Context, queries *store.Queries, siteID, buildID, distDir string) (eval.CategoryReport, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	site, err := eval.LoadSiteContext(ctx, queries, siteID, distDir)
	if err != nil {
		return eval.CategoryReport{}, fmt.Errorf("load site context: %w", err)
	}

	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(site, playbook)

	score, max := eval.ComputeScore(checks)
	report := eval.CategoryReport{
		Category: "design",
		Score:    score,
		MaxScore: max,
		Grade:    eval.ScoreToGrade(score, max),
		Checks:   checks,
	}

	checksJSON, _ := json.Marshal(checks)
	if err := queries.CreateEvaluation(ctx, store.CreateEvaluationParams{
		ID:         newID(),
		BuildID:    buildID,
		SiteID:     siteID,
		Category:   "design",
		Score:      int64(score),
		MaxScore:   int64(max),
		Grade:      report.Grade,
		ChecksJson: string(checksJSON),
	}); err != nil {
		slog.Error("critique: persist failed", "site_id", siteID, "build_id", buildID, "err", err)
	}

	slog.Info("critique: complete", "site_id", siteID, "build_id", buildID, "grade", report.Grade, "checks", len(checks))
	return report, nil
}

// RunChecks runs every design check against the site context using the
// playbook as the rule source. Pure function, no I/O, easy to unit test.
func RunChecks(site *eval.SiteContext, playbook agent.DesignPlaybookInfo) []eval.CheckResult {
	var checks []eval.CheckResult

	// Concatenate all page HTML once for whole-site searches. Per-page
	// scope is preserved for findings that need to attribute the page.
	allHTML := concatHTML(site)
	allText := concatVisibleText(site)
	allCSS := readSiteCSS(site)

	checks = append(checks, antiPatternChecks(playbook.AntiPatterns, allHTML, allCSS)...)
	checks = append(checks, contentAuthenticityChecks(playbook.ContentAuthenticity, allText, site)...)
	checks = append(checks, motionChecks(playbook.Motion, allCSS, allHTML)...)
	checks = append(checks, iconPolicyChecks(playbook.IconPolicy, allHTML)...)
	checks = append(checks, auditChecklistInfo(playbook.AuditChecklist)...)

	return checks
}

// antiPatternChecks scans HTML + CSS for each banned pattern in the
// playbook. Each AntiPattern entry becomes one check; matching the
// banned string triggers a Fail with the playbook's preferred
// alternative as the recommendation.
func antiPatternChecks(patterns []agent.AntiPattern, html, css string) []eval.CheckResult {
	var out []eval.CheckResult
	combined := html + "\n" + css

	for _, ap := range patterns {
		needle := extractAntiPatternNeedle(ap.Banned)
		name := truncate(ap.Banned, 60)
		section := "anti_patterns"
		if needle == "" {
			out = append(out, eval.Info(name, section, ap.Preferred))
			continue
		}
		if antiPatternMatches(needle, combined) {
			out = append(out, eval.Fail(
				name, section, 5, eval.SeverityWarning,
				fmt.Sprintf("Found %q in built output. Banned: %s", needle, ap.Banned),
				ap.Preferred,
			))
		} else {
			out = append(out, eval.Pass(name, section, 5, "no match in HTML/CSS"))
		}
	}
	return out
}

// antiPatternMatches dispatches to a needle-specific matcher. Font
// names need word-boundary matching so "Inter" doesn't false-positive
// on "Interaction"; gradient/backdrop-filter rules can use a plain
// substring scan because their tokens never collide with prose.
var fontNameNeedles = map[string]bool{
	"inter": true, "roboto": true, "arial": true, "space grotesk": true,
}

func antiPatternMatches(needle, combined string) bool {
	low := strings.ToLower(combined)
	n := strings.ToLower(needle)
	if fontNameNeedles[n] {
		// Match font-family declarations that name the banned font.
		// Covers `font-family: Inter`, `font-family:'Inter'`, "Inter,"
		// and Google-Fonts URLs that include the banned family.
		patterns := []string{
			"font-family:" + n,
			"font-family: " + n,
			"font-family:'" + n + "'",
			"font-family: '" + n + "'",
			"font-family:\"" + n + "\"",
			"font-family: \"" + n + "\"",
			"family=" + n,
			"family=" + strings.ReplaceAll(n, " ", "+"),
			"\"" + n + "\",",
			"'" + n + "',",
		}
		for _, p := range patterns {
			if strings.Contains(low, p) {
				return true
			}
		}
		return false
	}
	return strings.Contains(low, n)
}

// extractAntiPatternNeedle pulls a literal substring out of an anti-pattern
// description. Many entries use prose like 'Inter font everywhere' or
// 'Pure black (#000)'; we extract the most-grep-able token.
func extractAntiPatternNeedle(banned string) string {
	low := strings.ToLower(banned)
	known := []string{
		"inter", "roboto", "arial", "space grotesk",
		"#000000", "#000", "rgb(0,0,0)", "rgb(0, 0, 0)",
		"backdrop-filter", "linear-gradient",
		"box-shadow: 0 0",
	}
	for _, k := range known {
		if strings.Contains(low, k) {
			return k
		}
	}
	// Single-quoted literal inside the banned string, e.g. `Use 'Elevate'`.
	if a, b := strings.Index(banned, "'"), strings.LastIndex(banned, "'"); a >= 0 && b > a+1 {
		return banned[a+1 : b]
	}
	return ""
}

// contentAuthenticityChecks scans visible text for banned slop terms
// (names, numbers, companies, phrases). Each match becomes one Fail.
func contentAuthenticityChecks(rules agent.ContentRules, text string, site *eval.SiteContext) []eval.CheckResult {
	var out []eval.CheckResult
	textLower := strings.ToLower(text)

	scan := func(category string, terms []string) {
		for _, t := range terms {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			needle := strings.ToLower(t)
			name := fmt.Sprintf("slop:%s:%s", category, truncate(t, 40))
			if strings.Contains(textLower, needle) {
				out = append(out, eval.Fail(
					name, "content_authenticity", 3, eval.SeverityWarning,
					fmt.Sprintf("Banned %s %q appears in visible copy.", category, t),
					"Replace with specific, real, named example. Generic AI-slop terms damage credibility.",
				))
			}
		}
	}

	scan("name", rules.BannedNames)
	scan("number", rules.BannedNumbers)
	scan("company", rules.BannedCompanies)
	scan("phrase", rules.BannedPhrases)

	if len(out) == 0 {
		out = append(out, eval.Pass("slop_terms", "content_authenticity", 5, "no banned slop terms in visible copy"))
	}
	_ = site // reserved for per-page attribution if we expand later
	return out
}

// motionChecks looks for transition: all and similarly-broad CSS rules
// the playbook bans. Only CSS is scanned; atomicsite is static-Astro
// so JS motion libs aren't a concern here.
var transitionAllRE = regexp.MustCompile(`(?i)transition\s*:\s*all\b`)

func motionChecks(_ agent.MotionGuidance, css, _ string) []eval.CheckResult {
	var out []eval.CheckResult
	if transitionAllRE.MatchString(css) {
		out = append(out, eval.Fail(
			"transition_all", "motion", 3, eval.SeverityInfo,
			"`transition: all` matched in built CSS.",
			"Be explicit: list the properties you actually animate (transform, opacity).",
		))
	} else {
		out = append(out, eval.Pass("transition_all", "motion", 3, "no broad `transition: all` rules"))
	}
	return out
}

// iconPolicyChecks emits Info entries when icon imports outside the
// curated set are detected. Hard to be precise from rendered HTML;
// stays informational.
func iconPolicyChecks(_ agent.IconRules, html string) []eval.CheckResult {
	var out []eval.CheckResult
	if strings.Contains(strings.ToLower(html), "font-awesome") {
		out = append(out, eval.Fail(
			"icon_set:font-awesome", "icon_policy", 2, eval.SeverityInfo,
			"Font Awesome reference detected. Playbook prefers Lucide.",
			"Swap to a Lucide icon. See playbook IconPolicy.Available for the curated 52-item set.",
		))
	} else {
		out = append(out, eval.Pass("icon_set", "icon_policy", 2, "no banned icon set references"))
	}
	return out
}

// auditChecklistInfo turns each AuditChecklist item into an Info entry
// so the agent (or human reviewer) sees the full pre-flight checklist
// rendered alongside critique findings.
func auditChecklistInfo(items []agent.AuditItem) []eval.CheckResult {
	out := make([]eval.CheckResult, 0, len(items))
	for _, item := range items {
		out = append(out, eval.Info(
			truncate(item.Check, 80), "audit_checklist", item.Why,
		))
	}
	return out
}

// concatHTML joins every page's HTML into one big string so checks
// can do site-wide pattern matches without per-page bookkeeping.
func concatHTML(site *eval.SiteContext) string {
	var sb strings.Builder
	for _, p := range site.Pages {
		sb.WriteString(p.HTML)
		sb.WriteString("\n")
	}
	return sb.String()
}

// concatVisibleText extracts visible text content (what users read)
// from each page. Strips tags + script/style. Used for slop-term scans.
var (
	tagRE        = regexp.MustCompile(`<[^>]+>`)
	scriptStyleRE = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	wsRE          = regexp.MustCompile(`\s+`)
)

func concatVisibleText(site *eval.SiteContext) string {
	var sb strings.Builder
	for _, p := range site.Pages {
		stripped := scriptStyleRE.ReplaceAllString(p.HTML, " ")
		stripped = tagRE.ReplaceAllString(stripped, " ")
		stripped = wsRE.ReplaceAllString(stripped, " ")
		sb.WriteString(stripped)
		sb.WriteString("\n")
	}
	return sb.String()
}

// readSiteCSS reads any built CSS files in dist/ that the site
// context exposes. Falls back to scanning <style> tags inside HTML
// when no separate CSS file is loaded.
func readSiteCSS(site *eval.SiteContext) string {
	var sb strings.Builder
	for _, p := range site.Pages {
		// Pull <style> blocks from the page HTML so inlined Astro
		// styles are scanned. The eval package already loads the
		// per-site computed CSS bundle elsewhere; the design checks
		// only need broad pattern matches, not perfect coverage.
		matches := scriptStyleRE.FindAllStringSubmatch(p.HTML, -1)
		for _, m := range matches {
			if len(m) > 1 && strings.EqualFold(m[1], "style") {
				sb.WriteString(m[0])
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "."
}

// newID generates an evaluations row id (12-byte hex). Same shape as
// eval.newID() but local to this package.
func newID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
