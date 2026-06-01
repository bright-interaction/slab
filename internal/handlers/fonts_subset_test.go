package handlers

import "testing"

func TestNormalizeSubset(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "latin"},
		{"latin", "latin"},
		{"  latin  ", "latin"},
		{"LATIN", "latin"},
		{"latin-ext", "latin-ext"},
		{"cyrillic", "cyrillic"},
		{"all", "all"},
		{"unknown", "latin"},
		{"foo-bar", "latin"},
	}
	for _, c := range cases {
		if got := normalizeSubset(c.in); got != c.want {
			t.Errorf("normalizeSubset(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidSubsetsAllowlist(t *testing.T) {
	// Lock the allowlist so a future drift between the REST handler and
	// the MCP tool / Go builder fails CI rather than silently shipping a
	// subset the renderer doesn't know about.
	want := []string{"latin", "latin-ext", "cyrillic", "greek", "vietnamese", "arabic", "hebrew", "all"}
	if len(validSubsets) != len(want) {
		t.Fatalf("validSubsets has %d entries; want %d", len(validSubsets), len(want))
	}
	for _, name := range want {
		if _, ok := validSubsets[name]; !ok {
			t.Errorf("validSubsets missing %q", name)
		}
	}
}
