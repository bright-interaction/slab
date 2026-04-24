package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/store"
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
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// ValidateBlock checks a block creation/update against guardrail rules.
func (g *GuardrailEngine) ValidateBlock(ctx context.Context, siteID string, blockType string, dataJSON string) []Violation {
	unescaped := unescapeJSON(dataJSON)
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

	// Built-in security checks (always active, regardless of rules)
	if strings.Contains(unescaped, "<script") {
		violations = append(violations, Violation{
			Rule:     "security",
			Message:  "Inline <script> tags are not allowed in block data. Use components or allowed external scripts.",
			Severity: "error",
		})
	}
	if strings.Contains(unescaped, "javascript:") {
		violations = append(violations, Violation{
			Rule:     "security",
			Message:  "javascript: URLs are not allowed. Use proper href links.",
			Severity: "error",
		})
	}
	for _, evt := range []string{"onclick=", "onerror=", "onload=", "onmouseover=", "onfocus="} {
		if strings.Contains(unescaped, evt) {
			violations = append(violations, Violation{
				Rule:     "security",
				Message:  fmt.Sprintf("Inline event handler %q is not allowed. Use components with proper event handling.", evt),
				Severity: "error",
			})
			break
		}
	}

	// Best practice warnings
	if strings.Contains(unescaped, "!important") {
		violations = append(violations, Violation{
			Rule:     "best_practice",
			Message:  "Avoid using !important in CSS. Use specific CSS classes instead.",
			Severity: "warning",
		})
	}
	if strings.Contains(unescaped, "style=") {
		violations = append(violations, Violation{
			Rule:     "best_practice",
			Message:  "Inline styles detected. Prefer using global CSS classes for consistency.",
			Severity: "warning",
		})
	}

	return violations
}

// ValidateBlockCount checks if adding a block would exceed the max_blocks limit.
func (g *GuardrailEngine) ValidateBlockCount(ctx context.Context, siteID string, pageID string) []Violation {
	rules, _ := g.queries.ListActiveGuardrailsBySite(ctx, siteID)
	maxBlocks := 50 // default

	for _, r := range rules {
		if r.RuleType == "max_blocks" {
			var n int
			if err := json.Unmarshal([]byte(r.Value), &n); err == nil && n > 0 {
				maxBlocks = n
			}
		}
	}

	blocks, _ := g.queries.ListBlocksByPage(ctx, pageID)
	if len(blocks) >= maxBlocks {
		return []Violation{{
			Rule:     "max_blocks",
			Message:  fmt.Sprintf("Page already has %d blocks (maximum %d). Remove blocks before adding more.", len(blocks), maxBlocks),
			Severity: "error",
		}}
	}

	return nil
}

// ValidateRequiredBlocks checks that required blocks are still present after a delete.
func (g *GuardrailEngine) ValidateRequiredBlocks(ctx context.Context, siteID string, pageSlug string, pageID string, deletingBlockID string) []Violation {
	rules, _ := g.queries.ListActiveGuardrailsBySite(ctx, siteID)

	for _, r := range rules {
		if r.RuleType != "require_block" {
			continue
		}
		// Target is the page slug (or "*" for all pages)
		if r.Target != "*" && r.Target != pageSlug {
			continue
		}

		var requiredTypes []string
		if err := json.Unmarshal([]byte(r.Value), &requiredTypes); err != nil {
			continue
		}

		blocks, _ := g.queries.ListBlocksByPage(ctx, pageID)
		remainingTypes := map[string]bool{}
		for _, b := range blocks {
			if b.ID != deletingBlockID {
				remainingTypes[b.BlockType] = true
			}
		}

		for _, rt := range requiredTypes {
			if !remainingTypes[rt] {
				return []Violation{{
					Rule:     "require_block",
					Message:  fmt.Sprintf("Page %q requires a %q block. Cannot delete the last one.", pageSlug, rt),
					Severity: "error",
				}}
			}
		}
	}

	return nil
}

// ValidatePageSlug checks a page slug against architecture constraints.
func (g *GuardrailEngine) ValidatePageSlug(ctx context.Context, siteID string, slug string) []Violation {
	var violations []Violation

	// Root slug is always valid
	if slug == "/" {
		return nil
	}

	arch, _ := g.queries.GetSiteArchitecture(ctx, siteID)
	maxDepth := int64(3)
	if arch.ID != "" {
		maxDepth = arch.MaxDepth
	}

	trimmed := strings.Trim(slug, "/")
	depth := strings.Count(trimmed, "/") + 1
	if int64(depth) > maxDepth {
		violations = append(violations, Violation{
			Rule:     "max_url_depth",
			Message:  fmt.Sprintf("URL depth %d exceeds maximum %d. Keep URLs shallow for crawl efficiency.", depth, maxDepth),
			Severity: "error",
		})
	}

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
	if strings.Contains(slug, " ") {
		violations = append(violations, Violation{
			Rule:     "url_convention",
			Message:  "URLs must not contain spaces.",
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
	if !strings.HasPrefix(slug, "/") {
		violations = append(violations, Violation{
			Rule:     "url_convention",
			Message:  "URLs must start with /.",
			Severity: "error",
		})
	}

	return violations
}

// ShouldNoindex returns true if a page is an orphan (not in any silo) and should default to noindex.
func (g *GuardrailEngine) ShouldNoindex(ctx context.Context, siteID string, pageSlug string) bool {
	// Root page is never an orphan
	if pageSlug == "/" {
		return false
	}
	// Legal pages are never orphans
	legalPrefixes := []string{"/privacy", "/terms", "/cookie", "/imprint", "/legal"}
	for _, lp := range legalPrefixes {
		if strings.HasPrefix(pageSlug, lp) {
			return false
		}
	}

	silos, _ := g.queries.ListSilosBySite(ctx, siteID)
	if len(silos) == 0 {
		return false // No silos defined, all pages are valid
	}

	for _, s := range silos {
		if strings.HasPrefix(pageSlug, s.SlugPrefix) {
			return false // Page belongs to a silo
		}
	}

	// Top-level pages like /about, /contact are not orphans
	trimmed := strings.Trim(pageSlug, "/")
	if !strings.Contains(trimmed, "/") {
		return false
	}

	// Multi-level page not in any silo = orphan
	return true
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

// SeedDefaultGuardrails creates default guardrail rules for a new site.
func (g *GuardrailEngine) SeedDefaultGuardrails(ctx context.Context, siteID string) error {
	defaults := []struct {
		ruleType string
		target   string
		value    string
		severity string
	}{
		// Forbidden patterns
		{"forbid_pattern", "*", "<iframe", "warning"},

		// Block count limit
		{"max_blocks", "*", "30", "error"},

		// Required blocks on homepage
		{"require_block", "/", `["hero"]`, "warning"},
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

// SeedDefaultKnowledgebase creates default knowledgebase entries for a new site.
func (g *GuardrailEngine) SeedDefaultKnowledgebase(ctx context.Context, siteID string) error {
	defaults := []struct {
		category string
		title    string
		content  string
	}{
		{
			"accessibility",
			"WCAG AA Compliance",
			"All pages must meet WCAG 2.1 AA standards:\n- Color contrast ratio >= 4.5:1 for normal text, >= 3:1 for large text\n- All images must have descriptive alt text (not 'image', 'photo', or empty)\n- Use semantic HTML (main, nav, header, footer landmarks)\n- Include skip navigation link\n- Form inputs must have associated labels\n- Heading hierarchy must be sequential (H1 -> H2 -> H3, no skipping)",
		},
		{
			"technical",
			"No Inline Styles or Scripts",
			"Use global CSS classes for all styling. Never use inline style= attributes or <script> tags in page content. Components should reference CSS classes by name. This ensures consistency across the site and makes theming possible.",
		},
		{
			"seo",
			"Meta Tag Requirements",
			"Every page must have:\n- Title tag: 30-60 characters, page-specific (not site name alone)\n- Meta description: 120-160 characters, includes a call to action\n- Canonical URL\n- Open Graph tags (og:title, og:description, og:image, og:type)\n- Single H1 per page\n- Heading hierarchy (H1 -> H2 -> H3)\n- At least 2 internal links per page",
		},
		{
			"seo",
			"URL Structure Rules",
			"URLs must be:\n- Lowercase only\n- Hyphens for word separation (no underscores)\n- Maximum 3 levels deep (e.g., /services/scanning/details)\n- Under 75 characters\n- Descriptive and human-readable\n- No trailing slashes except root",
		},
		{
			"security",
			"External Resource Policy",
			"Only load external resources (scripts, styles, fonts, iframes) from explicitly allowlisted domains. Never include tracking scripts without consent management. All external scripts must be loaded with the integrity attribute (SRI) when available.",
		},
	}

	for i, d := range defaults {
		_ = g.queries.CreateKnowledgebaseEntry(ctx, store.CreateKnowledgebaseEntryParams{
			ID:        newAgentID(),
			SiteID:    siteID,
			Category:  d.category,
			Title:     d.title,
			Content:   d.content,
			SortOrder: int64(i),
		})
	}

	return nil
}

func unescapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\u003c", "<")
	s = strings.ReplaceAll(s, "\\u003e", ">")
	s = strings.ReplaceAll(s, "\\u0026", "&")
	return s
}

func joinKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func newAgentID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
