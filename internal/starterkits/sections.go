package starterkits

import (
	"encoding/json"
	"fmt"
)

// Section builders construct typed-block JSON payloads for the four
// "above-the-fold trust" elements every upgraded kit ships: hero with a
// curated graphic, logo strip, stat grid, FAQ accordion. They keep the
// kit Apply() functions readable by hiding the JSON escape ceremony.
//
// The renderer side lives in internal/builder (renderHeroGraphic,
// blocks_extra.go, pages.go). Block schemas live in internal/blocks/
// registry.go. Whenever a new field is added to a schema, update the
// matching builder here so the section keeps emitting the full set.

// HeroPayload mirrors the fields the `hero` block schema accepts. Empty
// fields are omitted from JSON so the renderer falls back to its default.
type HeroPayload struct {
	Eyebrow        string `json:"eyebrow,omitempty"`
	Headline       string `json:"headline,omitempty"`
	HeadlineAccent string `json:"headline_accent,omitempty"`
	Subheading     string `json:"subheading,omitempty"`
	CTAText        string `json:"cta_text,omitempty"`
	CTAUrl         string `json:"cta_url,omitempty"`
	SecondaryLabel string `json:"secondary_label,omitempty"`
	SecondaryUrl   string `json:"secondary_url,omitempty"`
	Background     string `json:"bg,omitempty"`
	HeroGraphic    string `json:"hero_graphic,omitempty"`
	MonogramChar   string `json:"monogram_char,omitempty"`
	AuditScore     string `json:"audit_score,omitempty"`
	AuditScoreMax  string `json:"audit_score_max,omitempty"`
	AuditBaseline  string `json:"audit_baseline,omitempty"`
	AuditLabel     string `json:"audit_label,omitempty"`
}

// HeroBlock wraps a HeroPayload into the blockDef shape seedBlocks
// expects. Use this instead of hand-writing JSON strings to ensure
// every hero ships with hero_graphic set (the kit audit found NONE of
// the existing kits wired hero_graphic, which is the single biggest
// "first impression looks generic" lever).
func HeroBlock(h HeroPayload) blockDef {
	return blockDef{BlockType: "hero", Data: marshalJSON(h)}
}

// LogoStripBlock builds a trust-bar of text-mark logos. Each item only
// needs a label and (optional) href; image_id is left blank so the kit
// works before the user has uploaded any media. The renderer falls back
// to a typographic mark per item.
func LogoStripBlock(label string, items []LogoItem) blockDef {
	type stripItem struct {
		Label string `json:"label"`
		Alt   string `json:"alt,omitempty"`
		Href  string `json:"href,omitempty"`
	}
	type stripData struct {
		Label string      `json:"label,omitempty"`
		Items []stripItem `json:"items"`
	}
	out := stripData{Label: label, Items: make([]stripItem, 0, len(items))}
	for _, it := range items {
		out.Items = append(out.Items, stripItem{Label: it.Label, Alt: it.Label, Href: it.Href})
	}
	return blockDef{BlockType: "logo_carousel", Data: marshalJSON(out)}
}

// LogoItem is the user-facing input for LogoStripBlock. Keeping the
// API tight so kits don't have to know about image_id vs label fallback.
type LogoItem struct {
	Label string
	Href  string
}

// StatGridBlock renders three or four big numbers above the fold. Every
// upgraded kit ships one because "we have customers, we have outcomes,
// we have years in market" is the strongest single trust signal you can
// put in 100ms of attention.
func StatGridBlock(heading string, items []StatItem) blockDef {
	type statItem struct {
		Value   string `json:"value"`
		Label   string `json:"label"`
		Context string `json:"context,omitempty"`
	}
	type statData struct {
		Heading string     `json:"heading,omitempty"`
		Items   []statItem `json:"items"`
	}
	out := statData{Heading: heading, Items: make([]statItem, 0, len(items))}
	for _, it := range items {
		out.Items = append(out.Items, statItem{Value: it.Value, Label: it.Label, Context: it.Context})
	}
	return blockDef{BlockType: "stat_grid", Data: marshalJSON(out)}
}

// StatItem is one (value, label, context) row in a StatGridBlock.
type StatItem struct {
	Value   string
	Label   string
	Context string
}

// AccordionFAQBlock renders a native <details>/<summary> Q&A list. The
// renderer auto-emits FAQPage JSON-LD when this block is present, so
// every upgraded kit gets a free SEO win simply by including five real
// questions. Five is the sweet spot: enough to look thorough, few
// enough that they're all worth answering.
func AccordionFAQBlock(heading string, items []FAQItem) blockDef {
	type faqItem struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	type faqData struct {
		Heading string    `json:"heading,omitempty"`
		Items   []faqItem `json:"items"`
	}
	out := faqData{Heading: heading, Items: make([]faqItem, 0, len(items))}
	for _, it := range items {
		out.Items = append(out.Items, faqItem{Question: it.Question, Answer: it.Answer})
	}
	return blockDef{BlockType: "accordion_faq", Data: marshalJSON(out)}
}

// FAQItem is one (question, answer) pair in an AccordionFAQBlock.
type FAQItem struct {
	Question string
	Answer   string
}

// marshalJSON wraps json.Marshal with a panic-on-error fallback. Used
// here because the inputs are all literal Go structs the developer just
// wrote; if Marshal fails the developer hand-wrote something pathological
// (channel, function, cycle) and a panic at init/test time is the right
// signal. Run starterkits tests after editing a section payload.
func marshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("starterkits: marshal section payload: %v", err))
	}
	return string(b)
}
