package builder

import (
	"fmt"
	"strings"
)

// Sprint 5 quick-win (2026-05-24): search_box block renderer.
//
// Pairs with the post-build Pagefind index pass in RunPagefind +
// shouldRunPagefind. The block emits a `<div id="search">` mount +
// a same-origin loader script that pulls the Pagefind UI bundle from
// /pagefind/pagefind-ui.js (written into dist/pagefind/ by the
// post-build pagefind run). CSP-safe (same-origin, no third-party
// CDN); when search.pagefind_enabled is off, the dist/pagefind/
// directory doesn't exist and the script 404s silently, leaving an
// inert mount in place. Tenants flipping the setting on later just
// need a rebuild; existing site bytes never change until then.

func renderSearchBoxBlock(data map[string]any) string {
	placeholder := dataString(data, "placeholder")
	if placeholder == "" {
		placeholder = "Search the site"
	}
	showEmpty := true
	if v, ok := data["show_empty_state"].(bool); ok {
		showEmpty = v
	}
	resultsMax := int64(6)
	if n := numberFromData(data, "results_max"); n > 0 {
		resultsMax = n
		if resultsMax > 50 {
			resultsMax = 50
		}
	}

	var b strings.Builder
	b.WriteString("  <section class=\"block block--search_box\">\n")
	b.WriteString("    <div id=\"search\" data-pagefind-mount></div>\n")
	b.WriteString("    <link rel=\"stylesheet\" href=\"/pagefind/pagefind-ui.css\" />\n")
	b.WriteString("    <script src=\"/pagefind/pagefind-ui.js\" defer></script>\n")
	// Init script. Inline + small + same-origin matches the
	// RenderVisitorHydration / RenderStorefrontIslandTag pattern; CSP
	// allows it via 'self' since the dist file lives same-origin.
	b.WriteString("    <script defer>\n")
	b.WriteString("      window.addEventListener('DOMContentLoaded', function () {\n")
	b.WriteString("        if (typeof PagefindUI !== 'function') return;\n")
	b.WriteString(fmt.Sprintf("        new PagefindUI({ element: '#search', showImages: false, resetStyles: false, translations: { placeholder: %q, zero_results: %q } });\n",
		placeholder, zeroResultsCopy(showEmpty)))
	b.WriteString(fmt.Sprintf("        var sr = document.querySelector('#search .pagefind-ui__results-area'); if (sr) sr.style.maxHeight = 'none';\n"))
	_ = resultsMax
	b.WriteString("      });\n")
	b.WriteString("    </script>\n")
	b.WriteString("  </section>\n")
	return b.String()
}

func zeroResultsCopy(show bool) string {
	if !show {
		return ""
	}
	return "No results for [SEARCH_TERM]."
}
