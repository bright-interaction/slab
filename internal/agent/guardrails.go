package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// GuardrailEngine validates agent operations against site rules.
type GuardrailEngine struct {
	queries *store.Queries
}

func NewGuardrailEngine(queries *store.Queries) *GuardrailEngine {
	return &GuardrailEngine{queries: queries}
}

// Violation represents a guardrail rule that was violated.
type Violation struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Severity string `json:"severity"`
}

// ValidateBlock checks a block creation/update against guardrail rules.
// dataJSON should be the raw JSON string. rawData is the unescaped content for pattern matching.
func (g *GuardrailEngine) ValidateBlock(ctx context.Context, siteID string, blockType string, dataJSON string) []Violation {
	// Unescape HTML entities that Go's json.Marshal produces
	unescaped := strings.ReplaceAll(dataJSON, "\\u003c", "<")
	unescaped = strings.ReplaceAll(unescaped, "\\u003e", ">")
	unescaped = strings.ReplaceAll(unescaped, "\\u0026", "&")
	rules, _ := g.queries.ListActiveGuardrailsBySite(ctx, siteID)
	var violations []Violation

	allowedTypes := map[string]bool{}
	hasTypeRules := false

	for _, r := range rules {
		switch r.RuleType {
		case "allow_block_type":
			hasTypeRules = true
			allowedTypes[r.Value] = true

		case "forbid_pattern":
			if strings.Contains(unescaped, r.Value) || strings.Contains(dataJSON, r.Value) {
				violations = append(violations, Violation{
					Rule:     "forbid_pattern",
					Message:  fmt.Sprintf("Block data contains forbidden pattern: %q. Remove it and try again.", r.Value),
					Severity: r.Severity,
				})
			}
		}
	}

	if hasTypeRules && !allowedTypes[blockType] {
		violations = append(violations, Violation{
			Rule:     "allow_block_type",
			Message:  fmt.Sprintf("Block type %q is not allowed. Allowed types: %s", blockType, joinKeys(allowedTypes)),
			Severity: "error",
		})
	}

	// Check for common bad patterns regardless of rules
	if strings.Contains(unescaped, "!important") {
		violations = append(violations, Violation{
			Rule:     "best_practice",
			Message:  "Avoid using !important in CSS. Use specific CSS classes instead.",
			Severity: "warning",
		})
	}
	if strings.Contains(unescaped, "<script") {
		violations = append(violations, Violation{
			Rule:     "security",
			Message:  "Inline <script> tags are not allowed in block data. Use components or allowed external scripts.",
			Severity: "error",
		})
	}

	return violations
}

// ValidatePageSlug checks a page slug against architecture constraints.
func (g *GuardrailEngine) ValidatePageSlug(ctx context.Context, siteID string, slug string) []Violation {
	var violations []Violation

	// Check URL depth
	arch, _ := g.queries.GetSiteArchitecture(ctx, siteID)
	maxDepth := int64(3)
	if arch.ID != "" {
		maxDepth = arch.MaxDepth
	}

	depth := strings.Count(strings.Trim(slug, "/"), "/") + 1
	if int64(depth) > maxDepth {
		violations = append(violations, Violation{
			Rule:     "max_url_depth",
			Message:  fmt.Sprintf("URL depth %d exceeds maximum %d. Keep URLs shallow for crawl efficiency.", depth, maxDepth),
			Severity: "error",
		})
	}

	// Check URL conventions
	if slug != strings.ToLower(slug) {
		violations = append(violations, Violation{
			Rule:     "url_convention",
			Message:  "URLs must be lowercase.",
			Severity: "error",
		})
	}
	if strings.Contains(slug, "_") {
		violations = append(violations, Violation{
			Rule:     "url_convention",
			Message:  "Use hyphens (-) instead of underscores (_) in URLs.",
			Severity: "error",
		})
	}
	if len(slug) > 75 {
		violations = append(violations, Violation{
			Rule:     "url_convention",
			Message:  fmt.Sprintf("URL length %d exceeds recommended maximum of 75 characters.", len(slug)),
			Severity: "warning",
		})
	}

	return violations
}

// ValidateCapability checks if an agent has the required capability.
func ValidateCapability(caps []string, required string) bool {
	for _, c := range caps {
		if c == required {
			return true
		}
	}
	return false
}

// HasErrors returns true if any violation has "error" severity.
func HasErrors(violations []Violation) bool {
	for _, v := range violations {
		if v.Severity == "error" {
			return true
		}
	}
	return false
}

func joinKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

// SeedDefaultGuardrails creates default guardrail rules for a new site.
func (g *GuardrailEngine) SeedDefaultGuardrails(ctx context.Context, siteID string) error {
	defaults := []struct {
		ruleType string
		target   string
		value    string
		severity string
	}{
		{"forbid_pattern", "*", "style=", "warning"},
		{"forbid_pattern", "*", "<script", "error"},
		{"forbid_pattern", "*", "<iframe", "warning"},
		{"forbid_pattern", "*", "javascript:", "error"},
		{"forbid_pattern", "*", "onclick=", "error"},
		{"forbid_pattern", "*", "onerror=", "error"},
	}

	for _, d := range defaults {
		_ = g.queries.CreateGuardrail(ctx, store.CreateGuardrailParams{
			ID:       newAgentID(),
			SiteID:   siteID,
			RuleType: d.ruleType,
			Target:   d.target,
			Value:    d.value,
			Severity: d.severity,
		})
	}

	return nil
}

func newAgentID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
