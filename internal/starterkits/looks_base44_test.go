package starterkits

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// TestLooksBase44_Quality asserts each kit ships a Base44-quality home
// page on first build: hero_graphic set (or bg=circuit for globe vibe),
// real trust signals above the fold (logo strip + stat grid), an FAQ
// block (auto-emits FAQPage JSON-LD), and a branding override on the
// site row (not generic indigo).
//
// This is the contract that protects "first build looks ship-ready"
// from drift. New kits MUST pass this. Existing kits will fail until
// upgraded; the test names which.
func TestLooksBase44_Quality(t *testing.T) {
	cases := []struct {
		kitID         string
		wantHeroAsset string // one of "hero_graphic" or "bg" must be set; this is the expected hero_graphic value or "" if bg=circuit
		wantBg        string // expected hero bg value if hero_graphic is empty
	}{
		{kitID: "saas-landing", wantHeroAsset: "mesh"},
		{kitID: "b2b-services", wantHeroAsset: "audit-receipt"},
		{kitID: "law-firm", wantHeroAsset: "audit-receipt"},
		{kitID: "b2c-local", wantHeroAsset: "pulse"},
		{kitID: "portfolio", wantHeroAsset: "monogram"},
		{kitID: "ecommerce", wantHeroAsset: "pulse"},
	}

	for _, tc := range cases {
		t.Run(tc.kitID, func(t *testing.T) {
			sqlDB, q := setupKitTestDB(t)
			siteID := "site-" + tc.kitID
			insertSite(t, sqlDB, siteID)

			kit, ok := Default.Get(tc.kitID)
			if !ok {
				t.Fatalf("kit %q not registered", tc.kitID)
			}
			if err := kit.Apply(context.Background(), q, siteID); err != nil {
				t.Fatalf("Apply: %v", err)
			}

			homeBlocks := homePageBlocks(t, sqlDB, siteID)
			if len(homeBlocks) < 5 {
				t.Fatalf("home page should ship >= 5 blocks, got %d", len(homeBlocks))
			}

			// Block 0 should be a hero with hero_graphic or bg set.
			first := homeBlocks[0]
			if first.blockType != "hero" && first.blockType != "split_hero" {
				t.Errorf("first block should be hero or split_hero, got %q", first.blockType)
			}
			if tc.wantHeroAsset != "" {
				if !strings.Contains(first.dataJson, `"hero_graphic":"`+tc.wantHeroAsset+`"`) {
					t.Errorf("first hero block missing hero_graphic=%q. data=%s", tc.wantHeroAsset, first.dataJson)
				}
			} else if tc.wantBg != "" {
				if !strings.Contains(first.dataJson, `"bg":"`+tc.wantBg+`"`) {
					t.Errorf("first hero block missing bg=%q. data=%s", tc.wantBg, first.dataJson)
				}
			}

			// Logo strip (logo_strip or logo_carousel) must appear in
			// the first three blocks. Trust above the fold.
			if !hasTypeInFirstN(homeBlocks, 3, "logo_strip", "logo_carousel") {
				t.Errorf("no logo strip in first 3 blocks; this is the single biggest trust signal a site can ship above the fold")
			}

			// Stat grid must appear in the first 4 blocks.
			if !hasTypeInFirstN(homeBlocks, 4, "stat_grid") {
				t.Errorf("no stat_grid in first 4 blocks; numbers above the fold are how visitors decide you're credible in 100ms")
			}

			// Accordion FAQ must appear somewhere. Auto-emits FAQPage
			// JSON-LD which is a free SEO win we should never skip.
			if !hasTypeAnywhere(homeBlocks, "accordion_faq") {
				t.Errorf("no accordion_faq block; missing FAQPage JSON-LD = free SEO loss")
			}

			// Branding must be overridden. Test fixture inserts the
			// site with empty primary_color; if branding wasn't
			// applied the column stays NULL/empty.
			var primary, fontHeading string
			row := sqlDB.QueryRowContext(context.Background(),
				"SELECT primary_color, font_heading FROM sites WHERE id=?", siteID)
			if err := row.Scan(&primary, &fontHeading); err != nil {
				t.Fatalf("scan site branding: %v", err)
			}
			if primary == "" || !strings.HasPrefix(primary, "#") {
				t.Errorf("primary_color not overridden by kit; want non-empty hex, got %q", primary)
			}
			if fontHeading == "" {
				t.Errorf("font_heading not overridden by kit; want non-empty, got %q", fontHeading)
			}
		})
	}
}

type homeBlock struct {
	blockType string
	dataJson  string
	sortOrder int64
}

func homePageBlocks(t *testing.T, sqlDB *sql.DB, siteID string) []homeBlock {
	t.Helper()
	rows, err := sqlDB.QueryContext(context.Background(),
		`SELECT b.block_type, b.data_json, b.sort_order
		 FROM blocks b JOIN pages p ON b.page_id = p.id
		 WHERE p.site_id = ? AND p.slug = '/'
		 ORDER BY b.sort_order ASC, b.created_at ASC`, siteID)
	if err != nil {
		t.Fatalf("query home blocks: %v", err)
	}
	defer rows.Close()
	var out []homeBlock
	for rows.Next() {
		var b homeBlock
		if err := rows.Scan(&b.blockType, &b.dataJson, &b.sortOrder); err != nil {
			t.Fatalf("scan block: %v", err)
		}
		out = append(out, b)
	}
	return out
}

func hasTypeInFirstN(blocks []homeBlock, n int, types ...string) bool {
	if n > len(blocks) {
		n = len(blocks)
	}
	for i := 0; i < n; i++ {
		for _, t := range types {
			if blocks[i].blockType == t {
				return true
			}
		}
	}
	return false
}

func hasTypeAnywhere(blocks []homeBlock, t string) bool {
	for _, b := range blocks {
		if b.blockType == t {
			return true
		}
	}
	return false
}
