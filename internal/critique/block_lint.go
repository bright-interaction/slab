package critique

import (
	"fmt"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/agent"
)

// LintFinding is one warning the block-level linter emits at write time
// (gap 4 of the design-quality push, 2026-05-21). Distinct from the
// Inspector's post-build Check struct because this fires synchronously
// on every create_block / update_block before HTML exists, so the
// shape is intentionally smaller: just a name, severity, message,
// optional fix hint. The agent reads these inline in the tool response
// and self-corrects in the same turn instead of waiting for the next
// build's Inspector run.
type LintFinding struct {
	Name     string `json:"name"`
	Severity string `json:"severity"` // warning | info
	Message  string `json:"message"`
	Field    string `json:"field,omitempty"`
	Fix      string `json:"fix,omitempty"`
}

// LintBlockData runs the block-level write-time checks for one block.
// Pass the block_type and the data map the agent is about to commit;
// findings come back ordered by severity then name. Empty result =
// nothing to flag. The function is allocation-light and called
// synchronously from create_block + update_block, so additions here
// must stay fast (no DB hits, no template rendering).
func LintBlockData(blockType string, data map[string]any, playbook agent.DesignPlaybookInfo) []LintFinding {
	return LintBlockDataWithArchetype(blockType, data, "", playbook)
}

// LintBlockDataWithArchetype is the archetype-aware variant: when the
// page carries an archetype lock (gap 6, 2026-05-21), block tokens
// that don't fit the archetype get flagged with archetype_drift.
// Empty archetype = same behaviour as LintBlockData.
func LintBlockDataWithArchetype(blockType string, data map[string]any, archetype string, playbook agent.DesignPlaybookInfo) []LintFinding {
	var out []LintFinding

	add := func(f LintFinding) { out = append(out, f) }

	switch blockType {
	case "hero", "split_hero":
		out = append(out, lintHeroQualityField(data)...)
		out = append(out, lintHeroHeadlineLength(data)...)
		out = append(out, lintArchetypeHeroGraphic(archetype, data)...)
	case "custom":
		out = append(out, lintCustomDuplicateEyebrow(data)...)
	}

	// Slop scan: walk every string-valued field in data and flag any
	// match against the playbook's BannedPhrases / BannedNames /
	// BannedCompanies / BannedNumbers. Case-insensitive substring match
	// is intentional: the Inspector's slop check uses the same rule so
	// the agent gets the same warning at write time it would after
	// build.
	bp := lowerSet(playbook.ContentAuthenticity.BannedPhrases)
	bn := lowerSet(playbook.ContentAuthenticity.BannedNames)
	bc := lowerSet(playbook.ContentAuthenticity.BannedCompanies)
	bnum := lowerSet(playbook.ContentAuthenticity.BannedNumbers)
	walkStringFields(data, func(field, value string) {
		lower := strings.ToLower(value)
		for term := range bp {
			if strings.Contains(lower, term) {
				add(LintFinding{
					Name:     "slop_term",
					Severity: "warning",
					Field:    field,
					Message:  fmt.Sprintf("Banned phrase %q in %q. Rewrite with a specific noun the visitor is shopping for.", term, field),
					Fix:      "Replace generic language with a concrete category, transformation verb, or measurable outcome. See get_design_playbook anti_patterns + copy_voice.",
				})
				return
			}
		}
		for term := range bn {
			if strings.Contains(lower, term) {
				add(LintFinding{
					Name:     "slop_name",
					Severity: "warning",
					Field:    field,
					Message:  fmt.Sprintf("Banned placeholder name %q in %q. Use a real attribution or remove the field.", term, field),
				})
				return
			}
		}
		for term := range bc {
			if strings.Contains(lower, term) {
				add(LintFinding{
					Name:     "slop_company",
					Severity: "warning",
					Field:    field,
					Message:  fmt.Sprintf("Banned placeholder company %q in %q. Use a real company name or omit the logo strip until you have one.", term, field),
				})
				return
			}
		}
		for term := range bnum {
			if strings.Contains(lower, term) {
				add(LintFinding{
					Name:     "slop_number",
					Severity: "warning",
					Field:    field,
					Message:  fmt.Sprintf("Round / suspicious number %q in %q. Use an actual measured value or remove the claim.", term, field),
				})
				return
			}
		}
	})

	return out
}

// lintHeroQualityField mirrors the post-build hero_quality Inspector
// check but at write time: a hero with no hero_graphic AND no
// bg=circuit AND no image_id ships as a plain text block, which is the
// AI-builder-demo look the design playbook explicitly bans.
func lintHeroQualityField(data map[string]any) []LintFinding {
	heroGraphic := strings.TrimSpace(stringOf(data, "hero_graphic"))
	bg := strings.TrimSpace(stringOf(data, "bg"))
	imageID := strings.TrimSpace(stringOf(data, "image_id"))
	if heroGraphic != "" || bg == "circuit" || imageID != "" {
		return nil
	}
	return []LintFinding{{
		Name:     "hero_quality",
		Severity: "warning",
		Field:    "hero_graphic",
		Message:  "Hero ships with no hero_graphic, no bg=circuit, and no image_id. Plain text heroes read as AI-builder-demo output.",
		Fix:      "Pick one of: hero_graphic=mesh|pulse|monogram|audit-receipt|globe-wire|gradient-orb, or bg=circuit (tech/infra vibe), or upload an image and set image_id.",
	}}
}

// lintHeroHeadlineLength flags overly long headlines. Playbook rule:
// headlines should land the transformation in <=12 words. The
// Inspector's eye-test check is slow; this synchronous warning lets
// the agent shorten before committing.
func lintHeroHeadlineLength(data map[string]any) []LintFinding {
	headline := strings.TrimSpace(stringOf(data, "headline"))
	if headline == "" {
		return nil
	}
	words := strings.Fields(headline)
	if len(words) <= 12 {
		return nil
	}
	return []LintFinding{{
		Name:     "headline_length",
		Severity: "warning",
		Field:    "headline",
		Message:  fmt.Sprintf("Hero headline is %d words. Premium marketing sites land the headline in <=12 words.", len(words)),
		Fix:      "Drop adjectives, drop hedge words, lead with a transformation verb (Stop / Own / Replace / Audit). Land the proposition in <=12 words. Subheading carries the rest.",
	}}
}

// archetypeHeroGraphics maps each vibe archetype to the hero_graphic
// values that fit it. The mapping is curated from the playbook's
// vibe_archetypes catalog AND the renderer's hero_graphic catalog;
// every entry has been verified to render in-archetype.
var archetypeHeroGraphics = map[string]map[string]struct{}{
	"mesh": {
		"mesh":         {},
		"gradient-orb": {},
		"globe-wire":   {},
	},
	"pulse": {
		"pulse":      {},
		"circuit":    {},
		"globe-wire": {},
	},
	"monogram": {
		"monogram": {},
		"":         {}, // monogram archetype permits text-only hero with monogram_char
	},
	"audit-receipt": {
		"audit-receipt": {},
	},
}

// lintArchetypeHeroGraphic enforces the page-level archetype lock
// (gap 6 of 6, 2026-05-21). When the page carries an archetype, the
// hero block's hero_graphic must come from the archetype's curated
// set; off-archetype graphics produce a warning. The check is
// scoped to hero / split_hero because that's where the visual
// archetype most strongly reads.
func lintArchetypeHeroGraphic(archetype string, data map[string]any) []LintFinding {
	archetype = strings.TrimSpace(archetype)
	if archetype == "" {
		return nil
	}
	allowed, ok := archetypeHeroGraphics[archetype]
	if !ok {
		return nil
	}
	heroGraphic := strings.TrimSpace(stringOf(data, "hero_graphic"))
	if _, fits := allowed[heroGraphic]; fits {
		return nil
	}
	listed := make([]string, 0, len(allowed))
	for k := range allowed {
		if k == "" {
			continue
		}
		listed = append(listed, k)
	}
	return []LintFinding{{
		Name:     "archetype_drift",
		Severity: "warning",
		Field:    "hero_graphic",
		Message:  fmt.Sprintf("Page archetype=%q expects hero_graphic in {%s}; current value %q drifts off-archetype.", archetype, strings.Join(listed, ", "), heroGraphic),
		Fix:      fmt.Sprintf("Pick one of {%s} for hero_graphic, or change the page's archetype via set_page_archetype if you want to commit to a different vibe.", strings.Join(listed, ", ")),
	}}
}

// lintCustomDuplicateEyebrow catches the documented MCP gotcha #1: when
// a custom block sets data.eyebrow AND the markup already contains its
// own eyebrow, renderCustomBlock prepends a duplicate. See
// Hive/concepts/atomicsite-mcp-mistakes.md item 1.
func lintCustomDuplicateEyebrow(data map[string]any) []LintFinding {
	eyebrow := strings.TrimSpace(stringOf(data, "eyebrow"))
	if eyebrow == "" {
		return nil
	}
	markup := stringOf(data, "markup")
	if markup == "" {
		return nil
	}
	lowerMarkup := strings.ToLower(markup)
	if !strings.Contains(lowerMarkup, `class="eyebrow"`) && !strings.Contains(lowerMarkup, "eyebrow__") && !strings.Contains(lowerMarkup, "bi-globe-hero-3d__eyebrow") {
		return nil
	}
	return []LintFinding{{
		Name:     "custom_block_duplicate_eyebrow",
		Severity: "warning",
		Field:    "eyebrow",
		Message:  "data.eyebrow is set AND the markup already contains its own eyebrow markup; the renderer will emit both, producing a duplicate eyebrow line.",
		Fix:      "Drop data.eyebrow from the create_block / update_block call when the markup ships its own eyebrow. Documented in Hive concept atomicsite-mcp-mistakes item 1.",
	}}
}

// walkStringFields visits every leaf string in the data map and calls
// visit with the dotted field path + value. Used by the slop scan so
// every authored string is in scope, not just a hand-picked subset.
func walkStringFields(data map[string]any, visit func(field, value string)) {
	for k, v := range data {
		switch val := v.(type) {
		case string:
			visit(k, val)
		case map[string]any:
			walkStringFields(val, func(sub, value string) {
				visit(k+"."+sub, value)
			})
		case []any:
			for i, item := range val {
				if s, ok := item.(string); ok {
					visit(fmt.Sprintf("%s[%d]", k, i), s)
					continue
				}
				if m, ok := item.(map[string]any); ok {
					walkStringFields(m, func(sub, value string) {
						visit(fmt.Sprintf("%s[%d].%s", k, i, sub), value)
					})
				}
			}
		}
	}
}

func stringOf(data map[string]any, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func lowerSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, s := range items {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		out[s] = struct{}{}
	}
	return out
}
