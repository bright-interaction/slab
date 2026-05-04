// Package knowledge owns the curriculum the AI agent reads through MCP
// resources. It exists so that an agent connecting to atomicsite via MCP
// can become a working expert in:
//
//   - the stack the builder emits (Astro, TypeScript, the CSS-variable
//     system in internal/builder/css.go, the 19 block renderers in
//     internal/blocks)
//   - UX/UI discipline that separates premium-feeling sites from
//     AI-generic output (typography, color, motion, accessibility,
//     performance budgets, dark mode, forms and nav patterns)
//
// The curriculum is shipped as embedded markdown for authoring ergonomics
// plus a Go-defined index that controls slug, category, and reading order.
// Both are versioned with the binary so an upgrade ships a curriculum
// update atomically.
//
// Surface: every Doc becomes one MCP resource at
// atomicsite://knowledge/<slug> with mimeType text/markdown. The
// generated catalog is served at atomicsite://knowledge/index. See
// internal/mcp/resources.go for the wiring.
package knowledge

import (
	"sort"
	"strings"
)

// Category groups docs in the catalog. Two halves: stack mastery (the
// concrete tech the builder emits) and ux mastery (the discipline that
// makes the output feel premium).
type Category string

const (
	CategoryStack Category = "stack"
	CategoryUX    Category = "ux"
)

// Doc is one curriculum entry. Body is the embedded markdown, parsed once
// at init via embed.go.
//
// Order controls reading sequence inside a category. Lower first. The
// master_the_stack prompt walks docs in (Category, Order) order so the
// agent reads foundational pieces before composition pieces.
type Doc struct {
	Slug     string   `json:"slug"`
	Title    string   `json:"title"`
	Category Category `json:"category"`
	Order    int      `json:"order"`
	Summary  string   `json:"summary"`
	Body     string   `json:"body,omitempty"`
}

// All returns every registered doc with body included, sorted by
// (Category, Order, Slug). Cheap; the slice is built once at init.
func All() []Doc {
	out := make([]Doc, 0, len(docs))
	for _, d := range docs {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].Slug < out[j].Slug
	})
	return out
}

// Catalog returns the same docs as All but with Body stripped, suitable
// for the index resource. Keeps the catalog payload small even if the
// curriculum grows to dozens of docs.
func Catalog() []Doc {
	full := All()
	out := make([]Doc, len(full))
	for i, d := range full {
		d.Body = ""
		out[i] = d
	}
	return out
}

// Get returns the doc matching slug, or (Doc{}, false) if absent.
func Get(slug string) (Doc, bool) {
	d, ok := docs[slug]
	return d, ok
}

// Slugs returns every registered slug. Used by the MCP resource
// registration loop and by tests that assert no embedded markdown is
// orphaned.
func Slugs() []string {
	out := make([]string, 0, len(docs))
	for s := range docs {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// firstHeading extracts the first markdown H1 from a body. Used at init
// to derive the doc title from the file content rather than duplicating
// it in the index. Falls back to the slug if the body has no H1 (which
// would be a test failure but we degrade gracefully at runtime).
func firstHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
	}
	return ""
}
