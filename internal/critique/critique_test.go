package critique

import (
	"strings"
	"testing"

	"github.com/bright-interaction/slab/internal/agent"
	"github.com/bright-interaction/slab/internal/eval"
)

// fakeSite builds a minimal SiteContext with one page so RunChecks can
// exercise antiPatternChecks / contentAuthenticityChecks / motionChecks
// against a known HTML+CSS blob.
func fakeSite(html string) *eval.SiteContext {
	return &eval.SiteContext{
		Pages: []eval.PageContext{{URL: "/index.html", Slug: "/", HTML: html}},
	}
}

func TestRunChecks_DetectsBannedFontInCSS(t *testing.T) {
	html := `<!doctype html><html><head><style>body{font-family:Inter,sans-serif}</style></head><body><h1>Hi</h1></body></html>`
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(html), playbook)

	if !hasFail(checks, "anti_patterns", "Inter") {
		t.Fatalf("expected an anti_patterns Fail mentioning Inter; got: %s", debugChecks(checks))
	}
}

func TestRunChecks_DetectsTransitionAll(t *testing.T) {
	html := `<style>.btn{transition:all 200ms ease}</style><div>x</div>`
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(html), playbook)

	if !hasFailNamed(checks, "transition_all") {
		t.Fatalf("expected transition_all Fail; got: %s", debugChecks(checks))
	}
}

func TestRunChecks_FlagsBannedSlopName(t *testing.T) {
	// John Doe is in BannedNames per ContentAuthenticity rules.
	html := `<p>Trusted by John Doe and Acme Corp.</p>`
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(html), playbook)

	if !hasFailIn(checks, "content_authenticity") {
		t.Fatalf("expected at least one content_authenticity Fail; got: %s", debugChecks(checks))
	}
}

func TestRunChecks_AuditChecklistIsInfo(t *testing.T) {
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(`<p>clean copy here, no slop.</p>`), playbook)

	hasInfo := false
	for _, c := range checks {
		if c.Section == "audit_checklist" && c.Passed == nil {
			hasInfo = true
			break
		}
	}
	if !hasInfo {
		t.Fatalf("expected at least one audit_checklist Info entry; got: %s", debugChecks(checks))
	}
}

func TestRunChecks_CleanSiteHasNoFails(t *testing.T) {
	// "specific" copy, no banned slop terms, no Inter, no transition: all,
	// + the standard meta a real Astro layout always emits (title,
	// description, og:image, canonical). Real pages always carry these;
	// the metaCompletenessChecks rightly fails when they're missing.
	html := `<!doctype html><html><head>` +
		`<title>Bright Interaction, GDPR audits for Swedish law firms</title>` +
		`<meta name="description" content="We scanned 599 Swedish advokatbyråer in Q1 2026 and surfaced 412 GDPR gaps. Run your own free audit in 90 seconds.">` +
		`<meta property="og:image" content="https://example.com/og.png">` +
		`<link rel="canonical" href="https://example.com/">` +
		`</head><body>` +
		`<style>.btn{transition:transform 150ms ease}</style>` +
		`<h1>Bright Interaction scans Swedish law firms</h1>` +
		`<p>We scanned 599 advokatbyråer in Q1 2026 and surfaced 412 GDPR gaps.</p>` +
		`</body></html>`
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(html), playbook)

	for _, c := range checks {
		if c.Passed != nil && !*c.Passed && c.Section != "audit_checklist" {
			t.Errorf("unexpected Fail on clean site: %s/%s | %s", c.Section, c.Name, c.Detail)
		}
	}
}

func TestRunChecks_DetectsFontAwesome(t *testing.T) {
	html := `<head><link rel="stylesheet" href="https://cdnjs.cloudflare.com/font-awesome/4.7/css/font-awesome.min.css"></head>`
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(html), playbook)

	if !hasFailNamed(checks, "icon_set:font-awesome") {
		t.Fatalf("expected font-awesome icon_policy Fail; got: %s", debugChecks(checks))
	}
}

func TestRunChecks_DetectsTokenDriftInCustomBlock(t *testing.T) {
	// A custom block hard-codes a non-canonical radius + an
	// untinted shadow + a literal font-family list. The new
	// designTokenCoherenceChecks should emit warnings for each.
	html := minimalHeadedHTML(`<section class="block block--custom"><style>` +
		`.weird { border-radius: 7px; box-shadow: 0 4px 8px #000; font-family: 'Comic Sans MS', cursive; }` +
		`</style><div class="weird">hi</div></section>`)
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(html), playbook)

	if !hasFailIn(checks, "design_tokens") {
		t.Fatalf("expected design_tokens Fail for the drift; got: %s", debugChecks(checks))
	}
}

func TestRunChecks_DetectsMotionDensityStacking(t *testing.T) {
	// Three perpetual animations on non-hover selectors. Critique should
	// emit a motion_density Fail (severity error at 3+).
	html := minimalHeadedHTML(`<style>` +
		`@keyframes spin { 0% { transform: rotate(0); } 100% { transform: rotate(360deg); } }` +
		`@keyframes drift { 0% { transform: translateX(0); } 100% { transform: translateX(10px); } }` +
		`@keyframes pulse { 0% { opacity: 0.5; } 100% { opacity: 1; } }` +
		`.a { animation: spin 4s infinite; }` +
		`.b { animation: drift 8s infinite; }` +
		`.c { animation: pulse 2s infinite; }` +
		`</style>`)
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(html), playbook)

	if !hasFailNamed(checks, "motion_density") {
		t.Fatalf("expected motion_density Fail for 3 stacked animations; got: %s", debugChecks(checks))
	}
}

func TestRunChecks_DetectsMissingMeta(t *testing.T) {
	// No <title>, no description, no og:image, no canonical. All four
	// metaCompletenessChecks should fire.
	html := `<!doctype html><html><head></head><body><h1>Hi</h1></body></html>`
	playbook := agent.DefaultDesignPlaybook()
	checks := RunChecks(fakeSite(html), playbook)

	if !hasFailIn(checks, "meta_audit") {
		t.Fatalf("expected meta_audit Fails for missing meta; got: %s", debugChecks(checks))
	}
}

// minimalHeadedHTML wraps a body fragment with the meta tags
// metaCompletenessChecks requires, so other tests can focus on the
// check under test without tripping the meta gate.
func minimalHeadedHTML(body string) string {
	return `<!doctype html><html><head>` +
		`<title>Atomicsite Inspector, 599 sites audited, A+ on 47.2%</title>` +
		`<meta name="description" content="Atomicsite inspector grades 128 checks across security, performance, accessibility, SEO, privacy. Industry average is 62 of 100. Run yours.">` +
		`<meta property="og:image" content="https://example.com/og.png">` +
		`<link rel="canonical" href="https://example.com/">` +
		`</head><body>` + body + `</body></html>`
}

// helpers

func hasFail(checks []eval.CheckResult, section, contains string) bool {
	for _, c := range checks {
		if c.Passed != nil && !*c.Passed && c.Section == section && strings.Contains(c.Name+c.Detail, contains) {
			return true
		}
	}
	return false
}

func hasFailIn(checks []eval.CheckResult, section string) bool {
	for _, c := range checks {
		if c.Passed != nil && !*c.Passed && c.Section == section {
			return true
		}
	}
	return false
}

func hasFailNamed(checks []eval.CheckResult, name string) bool {
	for _, c := range checks {
		if c.Passed != nil && !*c.Passed && c.Name == name {
			return true
		}
	}
	return false
}

func debugChecks(checks []eval.CheckResult) string {
	var b strings.Builder
	for _, c := range checks {
		state := "info"
		if c.Passed != nil {
			if *c.Passed {
				state = "pass"
			} else {
				state = "FAIL"
			}
		}
		b.WriteString(state + " " + c.Section + "/" + c.Name + " " + c.Detail + "\n")
	}
	return b.String()
}
