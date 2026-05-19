package critique

import (
	"strings"
	"testing"

	"github.com/bright-interaction/slab/internal/agent"
)

// TestHeroQualityChecks_FailsOnPlainHero asserts that a landing page
// with a hero block but no hero_graphic / circuit bg / image emits a
// hero_quality Fail. This is the protection against shipping "AI
// builder demo" looking pages.
func TestHeroQualityChecks_FailsOnPlainHero(t *testing.T) {
	html := minimalHeadedHTML(`<section class="block block--hero"><h1>Plain hero</h1><p>No graphic, no image.</p></section>`)
	checks := RunChecks(fakeSite(html), agent.DefaultDesignPlaybook())
	if !hasFailNamed(checks, "hero_quality") {
		t.Fatalf("expected hero_quality Fail for plain hero; got: %s", debugChecks(checks))
	}
}

// TestHeroQualityChecks_PassesOnGraphicHero asserts that a hero with a
// rendered hero_graphic does NOT fire the check.
func TestHeroQualityChecks_PassesOnGraphicHero(t *testing.T) {
	html := minimalHeadedHTML(`<section class="block block--hero has-graphic"><div class="hero-graphic hero-graphic--mesh" aria-hidden="true"></div><h1>Graphic hero</h1></section>`)
	checks := RunChecks(fakeSite(html), agent.DefaultDesignPlaybook())
	if hasFailNamed(checks, "hero_quality") {
		t.Fatalf("hero_quality should not fail when hero_graphic is present; got: %s", debugChecks(checks))
	}
}

// TestHeroQualityChecks_PassesOnCircuitBg asserts that bg=circuit is
// counted as a quality hero asset (same precedent the b2c-local + saas
// kits relied on before hero_graphic).
func TestHeroQualityChecks_PassesOnCircuitBg(t *testing.T) {
	html := minimalHeadedHTML(`<section class="block block--hero has-circuit-bg"><canvas></canvas><h1>Circuit hero</h1></section>`)
	checks := RunChecks(fakeSite(html), agent.DefaultDesignPlaybook())
	if hasFailNamed(checks, "hero_quality") {
		t.Fatalf("hero_quality should not fail when bg=circuit; got: %s", debugChecks(checks))
	}
}

// TestHeroQualityChecks_PassesWhenNoHero asserts that a page without
// any hero block is not penalised here. The eval engine's "Has H1"
// check covers the "must have a hero" requirement; this check is only
// about hero QUALITY, not presence.
func TestHeroQualityChecks_PassesWhenNoHero(t *testing.T) {
	html := minimalHeadedHTML(`<section class="block block--text"><h1>Just text</h1></section>`)
	checks := RunChecks(fakeSite(html), agent.DefaultDesignPlaybook())
	if hasFailNamed(checks, "hero_quality") {
		t.Fatalf("hero_quality should not fail when no hero block exists; got: %s", debugChecks(checks))
	}
}

// TestAboveTheFoldTrustChecks_FailsOnThinTop asserts a landing page
// with hero + 3 feature blocks + cta (no logos, no stats, no quote)
// triggers the trust-signal Fail.
func TestAboveTheFoldTrustChecks_FailsOnThinTop(t *testing.T) {
	body := strings.Join([]string{
		`<section class="block block--hero has-graphic"><div class="hero-graphic hero-graphic--mesh"></div><h1>Hero</h1></section>`,
		`<section class="block block--feature_grid"><h2>Features</h2></section>`,
		`<section class="block block--feature_grid"><h2>More</h2></section>`,
		`<section class="block block--feature_grid"><h2>Even more</h2></section>`,
		`<section class="block block--cta"><h2>CTA</h2></section>`,
	}, "")
	html := minimalHeadedHTML(body)
	checks := RunChecks(fakeSite(html), agent.DefaultDesignPlaybook())
	if !hasFailNamed(checks, "above_the_fold_trust") {
		t.Fatalf("expected above_the_fold_trust Fail when first 5 blocks ship no trust signal; got: %s", debugChecks(checks))
	}
}

// TestAboveTheFoldTrustChecks_PassesWithLogos asserts that adding a
// logo_strip in the first 5 blocks clears the check.
func TestAboveTheFoldTrustChecks_PassesWithLogos(t *testing.T) {
	body := strings.Join([]string{
		`<section class="block block--hero has-graphic"><div class="hero-graphic hero-graphic--mesh"></div><h1>Hero</h1></section>`,
		`<section class="block block--logo_carousel"><p>Trusted by</p></section>`,
		`<section class="block block--stat_grid"><h2>Numbers</h2></section>`,
		`<section class="block block--feature_grid"><h2>Features</h2></section>`,
		`<section class="block block--cta"><h2>CTA</h2></section>`,
	}, "")
	html := minimalHeadedHTML(body)
	checks := RunChecks(fakeSite(html), agent.DefaultDesignPlaybook())
	if hasFailNamed(checks, "above_the_fold_trust") {
		t.Fatalf("above_the_fold_trust should pass with logo_carousel + stat_grid in first 5 blocks; got: %s", debugChecks(checks))
	}
}

// TestAboveTheFoldTrustChecks_PassesWhenNoBlocks asserts a page with
// no block sections at all (legal pages, etc.) is not penalised.
func TestAboveTheFoldTrustChecks_PassesWhenNoBlocks(t *testing.T) {
	html := minimalHeadedHTML(`<h1>Cookies</h1><p>Plain content.</p>`)
	checks := RunChecks(fakeSite(html), agent.DefaultDesignPlaybook())
	if hasFailNamed(checks, "above_the_fold_trust") {
		t.Fatalf("above_the_fold_trust should pass when no block sections are present; got: %s", debugChecks(checks))
	}
}

// TestHeroQualityChecks_SeverityIsWarning confirms the new check uses
// warning severity (per Tom's direction: don't over-gate design).
func TestHeroQualityChecks_SeverityIsWarning(t *testing.T) {
	html := minimalHeadedHTML(`<section class="block block--hero"><h1>Plain</h1></section>`)
	checks := RunChecks(fakeSite(html), agent.DefaultDesignPlaybook())
	for _, c := range checks {
		if c.Name == "hero_quality" && c.Passed != nil && !*c.Passed {
			if c.Severity != "warning" {
				t.Errorf("hero_quality severity = %q, want warning (per Tom: design checks must not block builds)", c.Severity)
			}
			return
		}
	}
	t.Fatalf("expected a hero_quality Fail in output; got: %s", debugChecks(checks))
}
