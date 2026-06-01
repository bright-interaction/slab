package mcp

import "testing"

func TestNormalizeMCPSubset(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "latin"},
		{"latin", "latin"},
		{"  latin-ext  ", "latin-ext"},
		{"CYRILLIC", "cyrillic"},
		{"all", "all"},
		{"unknown", "latin"},
	}
	for _, c := range cases {
		if got := normalizeMCPSubset(c.in); got != c.want {
			t.Errorf("normalizeMCPSubset(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestMCPSubsetsMatchHandlerAllowlist catches drift between the MCP-side
// validSubsets map and the REST handler's. If a future PR adds "thai"
// to one and forgets the other, an agent upload and a human upload would
// produce different @font-face emissions on the same site.
func TestMCPSubsetsMatchHandlerAllowlist(t *testing.T) {
	want := []string{"latin", "latin-ext", "cyrillic", "greek", "vietnamese", "arabic", "hebrew", "all"}
	if len(mcpValidSubsets) != len(want) {
		t.Fatalf("mcpValidSubsets has %d entries; want %d", len(mcpValidSubsets), len(want))
	}
	for _, name := range want {
		if _, ok := mcpValidSubsets[name]; !ok {
			t.Errorf("mcpValidSubsets missing %q", name)
		}
	}
}
