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
	// "specific" copy, no banned slop terms, no Inter, no transition: all.
	html := `<style>.btn{transition:transform 150ms ease}</style>` +
		`<h1>Bright Interaction scans Swedish law firms</h1>` +
		`<p>We scanned 599 advokatbyråer in Q1 2026 and surfaced 412 GDPR gaps.</p>`
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
