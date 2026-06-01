package mcp

import "strings"

// mcpValidSubsets mirrors handlers.validSubsets. Duplicated here so the MCP
// package does not import the handlers package (avoids the dependency cycle
// the wider tools surface already steers around). Drift between the two
// allowlists is caught by TestNormalizeMCPSubset_MatchesHandler.
var mcpValidSubsets = map[string]struct{}{
	"latin":      {},
	"latin-ext":  {},
	"cyrillic":   {},
	"greek":      {},
	"vietnamese": {},
	"arabic":     {},
	"hebrew":     {},
	"all":        {},
}

// normalizeMCPSubset coerces an agent-supplied subset name into one of the
// allowed values. Empty / unknown falls back to "latin", matching the REST
// upload handler behavior so the agent and a human operator produce
// byte-identical @font-face emissions on the rendered site.
func normalizeMCPSubset(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if _, ok := mcpValidSubsets[s]; ok {
		return s
	}
	return "latin"
}
