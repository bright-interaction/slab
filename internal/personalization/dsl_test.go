package personalization

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"present", "name present", false},
		{"eq string", `lifecycle == "SQL"`, false},
		{"eq number", "lead_score == 60", false},
		{"gte number", "lead_score >= 60", false},
		{"in list", "last_topic in compliance,gdpr,nis2", false},
		{"in single", "lifecycle in SQL", false},
		{"empty", "", true},
		{"missing op", "lead_score 60", true},
		{"unknown op", "lead_score ~~ 60", true},
		{"semicolon", "name == \"a\"; alert(1)", true},
		{"script tag", `name == "<script>"`, true},
		{"javascript:", `redirect == "javascript:foo"`, true},
		{"bad key", "1bad == 2", true},
		{"missing value", "name == ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.expr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate(%q) err=%v want=%v", tc.expr, err, tc.wantErr)
			}
		})
	}
}

func TestEval(t *testing.T) {
	vars := map[string]any{
		"name":       "Joe",
		"email":      "",
		"lead_score": float64(72),
		"lifecycle":  "SQL",
		"last_topic": "compliance",
		"signed_up":  true,
	}
	cases := []struct {
		expr string
		want bool
	}{
		{"name present", true},
		{"email present", false},
		{"missing present", false},
		{`name == "Joe"`, true},
		{`name != "Mary"`, true},
		{"lead_score >= 60", true},
		{"lead_score >= 80", false},
		{"lead_score > 72", false},
		{"lead_score < 100", true},
		{"lead_score == 72", true},
		{`lifecycle == "SQL"`, true},
		{"last_topic in compliance,gdpr,nis2", true},
		{"last_topic in foo,bar", false},
		{"missing == 1", false},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			got, err := Eval(tc.expr, vars)
			if err != nil {
				t.Fatalf("Eval(%q) err=%v", tc.expr, err)
			}
			if got != tc.want {
				t.Fatalf("Eval(%q) = %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}
