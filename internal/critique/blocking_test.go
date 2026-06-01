package critique

import "testing"

// TestHasBlockingFindings covers the curated allowlist of finding names that
// are tripwires in strict mode. Soft warnings stay non-blocking.
func TestHasBlockingFindings(t *testing.T) {
	cases := []struct {
		name     string
		findings []LintFinding
		want     bool
	}{
		{"empty", nil, false},
		{"only soft warning", []LintFinding{
			{Name: "hero_quality", Severity: "warning"},
			{Name: "headline_length", Severity: "warning"},
		}, false},
		{"slop term blocks", []LintFinding{
			{Name: "hero_quality", Severity: "warning"},
			{Name: "slop_term", Severity: "warning"},
		}, true},
		{"slop name blocks", []LintFinding{{Name: "slop_name"}}, true},
		{"slop company blocks", []LintFinding{{Name: "slop_company"}}, true},
		{"slop number blocks", []LintFinding{{Name: "slop_number"}}, true},
		{"archetype drift blocks", []LintFinding{{Name: "archetype_drift"}}, true},
		{"unknown name does not block", []LintFinding{{Name: "future_rule_we_did_not_curate"}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasBlockingFindings(c.findings); got != c.want {
				t.Errorf("HasBlockingFindings(%+v) = %v, want %v", c.findings, got, c.want)
			}
		})
	}
}

func TestFilterBlocking(t *testing.T) {
	input := []LintFinding{
		{Name: "hero_quality"},
		{Name: "slop_term", Field: "headline"},
		{Name: "headline_length"},
		{Name: "archetype_drift", Field: "hero_graphic"},
	}
	got := FilterBlocking(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 blocking findings, got %d", len(got))
	}
	if got[0].Name != "slop_term" || got[1].Name != "archetype_drift" {
		t.Errorf("blocking order changed: %+v", got)
	}
}
