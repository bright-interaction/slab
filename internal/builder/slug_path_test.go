package builder

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestSlugToFilePathContainsTraversal is the audit C4 guard: a page slug
// must never resolve to a path outside <wsDir>/src/pages, even though the
// write-boundary validators (handlers.ValidatePageSlug) should already
// reject it. This is the last line of defense against a tenant overwriting
// another tenant's workspace via a crafted MCP slug.
func TestSlugToFilePathContainsTraversal(t *testing.T) {
	ws := t.TempDir()
	base := filepath.Join(ws, "src", "pages")

	// Legitimate slugs resolve inside the pages dir.
	for _, ok := range []struct{ slug, want string }{
		{"", filepath.Join(base, "index.astro")},
		{"about", filepath.Join(base, "about.astro")},
		{"blog/post-1", filepath.Join(base, "blog", "post-1.astro")},
		{"/about/", filepath.Join(base, "about.astro")}, // trimmed
	} {
		got, err := slugToFilePath(ok.slug, ws)
		if err != nil {
			t.Fatalf("slug %q returned error %v, want path", ok.slug, err)
		}
		if got != ok.want {
			t.Fatalf("slug %q = %q, want %q", ok.slug, got, ok.want)
		}
		if rel, _ := filepath.Rel(base, got); strings.HasPrefix(rel, "..") {
			t.Fatalf("slug %q escaped base: rel=%q", ok.slug, rel)
		}
	}

	// Slugs that escape <wsDir>/src/pages must fail closed. (Bare ".." is
	// not listed: with the ".astro" suffix it resolves to a contained file
	// "...astro", and the write-boundary validator rejects it anyway. The
	// guard's contract is specifically "never escape base".)
	for _, bad := range []string{
		"../evil",
		"../../../etc/passwd",
		"a/../../b",
		"../other-site-id/src/pages/index",
	} {
		if got, err := slugToFilePath(bad, ws); err == nil {
			t.Fatalf("traversal slug %q returned path %q with no error (escapes the workspace)", bad, got)
		}
	}
}
