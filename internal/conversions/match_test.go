package conversions

import "testing"

func TestMatchURLPattern(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"", "/", false},
		{"*", "/anything", true},
		{"*", "", true},
		{"/", "/", true},
		{"/", "", false},
		{"/thank-you", "/thank-you", true},
		{"/thank-you", "/thank-you/", false},
		{"/thank-you/*", "/thank-you/lead", true},
		{"/thank-you/*", "/thank-you/abc-def", true},
		{"/thank-you/*", "/thank-you", false},
		{"/thank-you/*", "/thank-you/", true},
		{"/thank-you/*", "/blog/lead", false},
		{"/blog/*/comments", "/blog/x/comments", true},
		{"/blog/*/comments", "/blog/abc-def/comments", true},
		{"/blog/*/comments", "/blog/x/y/comments", false}, // wildcard does not cross /
		{"/foo*", "/foobar", true},
		{"/foo*", "/foo", true},
		{"/foo*", "/fobar", false},
		{"/case-Sensitive", "/case-sensitive", false},
		{"/sub/*/end", "/sub/a/end", true},
		{"/sub/*/end", "/sub//end", true}, // empty segment is allowed in glob form
	}
	for _, c := range cases {
		got := MatchURLPattern(c.pattern, c.path)
		if got != c.want {
			t.Errorf("MatchURLPattern(%q,%q)=%v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestIsValidMatchType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"url_pattern", true},
		{"event_name", true},
		{"form_submit", true},
		{"", false},
		{"URL_PATTERN", false},
		{"unknown", false},
	}
	for _, c := range cases {
		if got := IsValidMatchType(c.in); got != c.want {
			t.Errorf("IsValidMatchType(%q)=%v, want %v", c.in, got, c.want)
		}
	}
}

func TestMatchGoals(t *testing.T) {
	t.Parallel()
	goals := []Goal{
		{ID: "g1", Slug: "lead", MatchType: MatchTypeURLPattern, MatchValue: "/thank-you/*"},
		{ID: "g2", Slug: "demo", MatchType: MatchTypeURLPattern, MatchValue: "/demo-booked"},
		{ID: "g3", Slug: "signup", MatchType: MatchTypeEventName, MatchValue: "signup"},
		{ID: "g4", Slug: "contact-form", MatchType: MatchTypeFormSubmit, MatchValue: "form_abc"},
		{ID: "g5", Slug: "any", MatchType: MatchTypeURLPattern, MatchValue: "*"},
	}

	t.Run("url match returns multi", func(t *testing.T) {
		got := MatchGoals(goals, MatchTypeURLPattern, "/thank-you/lead")
		if len(got) != 2 {
			t.Fatalf("got %d goals, want 2 (lead+any)", len(got))
		}
		if got[0].ID != "g1" || got[1].ID != "g5" {
			t.Errorf("order wrong: %s,%s", got[0].ID, got[1].ID)
		}
	})

	t.Run("url no match", func(t *testing.T) {
		got := MatchGoals(goals, MatchTypeURLPattern, "/random")
		if len(got) != 1 || got[0].ID != "g5" {
			t.Errorf("expected only catch-all, got %v", goalIDs(got))
		}
	})

	t.Run("event exact match", func(t *testing.T) {
		got := MatchGoals(goals, MatchTypeEventName, "signup")
		if len(got) != 1 || got[0].ID != "g3" {
			t.Errorf("got %v", goalIDs(got))
		}
	})

	t.Run("event no match", func(t *testing.T) {
		if got := MatchGoals(goals, MatchTypeEventName, "Signup"); len(got) != 0 {
			t.Errorf("case-sensitivity broken: %v", goalIDs(got))
		}
	})

	t.Run("form match", func(t *testing.T) {
		got := MatchGoals(goals, MatchTypeFormSubmit, "form_abc")
		if len(got) != 1 || got[0].ID != "g4" {
			t.Errorf("got %v", goalIDs(got))
		}
	})

	t.Run("form no match", func(t *testing.T) {
		if got := MatchGoals(goals, MatchTypeFormSubmit, "form_xyz"); len(got) != 0 {
			t.Errorf("got %v", goalIDs(got))
		}
	})
}

func goalIDs(gs []Goal) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = g.ID
	}
	return out
}
