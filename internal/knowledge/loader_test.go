package knowledge

import (
	"strings"
	"testing"
)

// TestAllReturnsAllRegisteredDocs guards against init drift: every entry
// in docMetadataTable must round-trip through All().
func TestAllReturnsAllRegisteredDocs(t *testing.T) {
	got := All()
	if len(got) != len(docMetadataTable) {
		t.Fatalf("All returned %d docs, want %d", len(got), len(docMetadataTable))
	}
	seen := map[string]bool{}
	for _, d := range got {
		seen[d.Slug] = true
	}
	for _, m := range docMetadataTable {
		if !seen[m.slug] {
			t.Errorf("All missing registered slug %q", m.slug)
		}
	}
}

// TestAllSortedByCategoryThenOrder asserts the sort contract that
// master_the_stack and the catalog UI rely on.
func TestAllSortedByCategoryThenOrder(t *testing.T) {
	got := All()
	for i := 1; i < len(got); i++ {
		prev, cur := got[i-1], got[i]
		if prev.Category > cur.Category {
			t.Errorf("docs[%d].Category=%q after docs[%d].Category=%q", i, cur.Category, i-1, prev.Category)
			continue
		}
		if prev.Category == cur.Category && prev.Order > cur.Order {
			t.Errorf("docs[%d].Order=%d after docs[%d].Order=%d in category %q",
				i, cur.Order, i-1, prev.Order, cur.Category)
		}
	}
}

// TestCatalogStripsBody keeps the index resource cheap to send.
func TestCatalogStripsBody(t *testing.T) {
	for _, d := range Catalog() {
		if d.Body != "" {
			t.Errorf("Catalog entry %q includes Body; should be stripped", d.Slug)
		}
		if d.Title == "" || d.Slug == "" {
			t.Errorf("Catalog entry %q has empty title/slug", d.Slug)
		}
	}
}

// TestGetReturnsBody asserts the per-doc resource path returns full body.
func TestGetReturnsBody(t *testing.T) {
	for _, slug := range Slugs() {
		d, ok := Get(slug)
		if !ok {
			t.Errorf("Get(%q) missing", slug)
			continue
		}
		if d.Body == "" {
			t.Errorf("Get(%q) returned empty body", slug)
		}
	}
}

// TestEveryDocStartsWithH1 enforces the title-derivation invariant from
// embed.go::init. If this fails, init would have panicked at startup.
func TestEveryDocStartsWithH1(t *testing.T) {
	for _, d := range All() {
		if !strings.Contains(d.Body, "\n# ") && !strings.HasPrefix(d.Body, "# ") {
			t.Errorf("doc %q missing top-level # heading", d.Slug)
		}
		if d.Title == "" {
			t.Errorf("doc %q has empty title", d.Slug)
		}
	}
}

// TestNoEmDashes enforces the user's writing-voice rule (no em dashes
// ever, see the project's CLAUDE.md). Catches authoring drift.
func TestNoEmDashes(t *testing.T) {
	for _, d := range All() {
		if strings.Contains(d.Body, "—") {
			t.Errorf("doc %q contains an em dash; rewrite with colon or comma", d.Slug)
		}
	}
}

// TestKnownCategories prevents typos in docMetadataTable.
func TestKnownCategories(t *testing.T) {
	for _, d := range All() {
		switch d.Category {
		case CategoryStack, CategoryUX:
		default:
			t.Errorf("doc %q has unknown category %q", d.Slug, d.Category)
		}
	}
}
