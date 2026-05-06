package builder

import "testing"

func TestCaddyQuote(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"default-src 'self'", `"default-src 'self'"`},
		{`max-age=63072000; includeSubDomains`, `"max-age=63072000; includeSubDomains"`},
		{"DENY", `"DENY"`},
		{`he said "hi"`, `"he said \"hi\""`},
		{`a\b`, `"a\\b"`},
		{"", `""`},
	}
	for _, c := range cases {
		if got := caddyQuote(c.in); got != c.want {
			t.Errorf("caddyQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
