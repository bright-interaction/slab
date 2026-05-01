package builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// SiteFontsCSS returns the @font-face block for one site's uploaded fonts.
// Self-hosted woff2 only — atomicsite does not load Google Fonts. Returns
// the empty string when the site has no uploaded fonts. Used by the
// per-block preview-html endpoint so the iframe matches the production
// font stack from layouts.go.
func SiteFontsCSS(ctx context.Context, queries *store.Queries, siteID string) string {
	rows, err := queries.ListSiteFonts(ctx, siteID)
	if err != nil || len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range rows {
		src := fmt.Sprintf("/atomicsite-fonts/%s/%s.woff2", siteID, f.ID)
		b.WriteString(fmt.Sprintf(
			"@font-face { font-family: %q; font-style: %s; font-weight: %d; font-display: swap; src: url(%q) format('woff2'); }\n",
			f.FamilyName, f.Style, f.Weight, src,
		))
	}
	return b.String()
}
