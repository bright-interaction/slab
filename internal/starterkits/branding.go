package starterkits

import (
	"context"
	"fmt"

	"github.com/bright-interaction/slab/internal/store"
)

// Branding is the archetype palette + typography a starter kit installs
// during Apply(). It overrides the site's default colors and fonts so a
// freshly seeded site already looks aligned with the kit's intent
// instead of generic indigo + Inter.
//
// All colors are six-digit hex (#RRGGBB). FontHeading and FontBody must
// be one of the family names the runtime skeleton has @font-face for
// (Inter, Geist, Geist Mono, Space Grotesk, Space Mono are shipped).
type Branding struct {
	PrimaryColor   string
	SecondaryColor string
	BgColor        string
	TextColor      string
	FontHeading    string
	FontBody       string
}

// applyBranding patches the site's branding fields in-place inside the
// caller's transaction. Mirrors the read-then-write-all pattern the
// agent_branding.go UpdateBranding handler uses (sqlc's UpdateSite is
// a full-row update; partial updates require this pre-read).
func applyBranding(ctx context.Context, q *store.Queries, siteID string, b Branding) error {
	row, err := q.GetSiteByID(ctx, siteID)
	if err != nil {
		return fmt.Errorf("read site for branding override: %w", err)
	}
	params := store.UpdateSiteParams{
		Name:              row.Name,
		Slug:              row.Slug,
		Domain:            row.Domain,
		Status:            row.Status,
		PrimaryColor:      row.PrimaryColor,
		SecondaryColor:    row.SecondaryColor,
		BgColor:           row.BgColor,
		TextColor:         row.TextColor,
		SurfaceColor:      row.SurfaceColor,
		BorderColor:       row.BorderColor,
		MutedColor:        row.MutedColor,
		AccentColor:       row.AccentColor,
		OnPrimaryColor:    row.OnPrimaryColor,
		FontHeading:       row.FontHeading,
		FontBody:          row.FontBody,
		MetaTitle:         row.MetaTitle,
		MetaDescription:   row.MetaDescription,
		OgImageID:         row.OgImageID,
		FaviconID:         row.FaviconID,
		Ga4ID:             row.Ga4ID,
		UmamiID:           row.UmamiID,
		UmamiUrl:          row.UmamiUrl,
		CookieproofDomain: row.CookieproofDomain,
		Lang:              row.Lang,
		ID:                row.ID,
	}
	if b.PrimaryColor != "" {
		params.PrimaryColor = b.PrimaryColor
	}
	if b.SecondaryColor != "" {
		params.SecondaryColor = b.SecondaryColor
	}
	if b.BgColor != "" {
		params.BgColor = b.BgColor
	}
	if b.TextColor != "" {
		params.TextColor = b.TextColor
	}
	if b.FontHeading != "" {
		params.FontHeading = b.FontHeading
	}
	if b.FontBody != "" {
		params.FontBody = b.FontBody
	}
	if err := q.UpdateSite(ctx, params); err != nil {
		return fmt.Errorf("apply branding: %w", err)
	}
	return nil
}

// Archetype palettes. Each starter kit Apply() calls applyBranding with
// the one matching its intent so the first-build palette already reads
// as the right vibe instead of generic indigo.
var (
	BrandMesh = Branding{
		PrimaryColor: "#6366f1", SecondaryColor: "#8b5cf6",
		BgColor: "#fafaf9", TextColor: "#0f172a",
		FontHeading: "Geist", FontBody: "Inter",
	}
	BrandPulse = Branding{
		PrimaryColor: "#f97316", SecondaryColor: "#ec4899",
		BgColor: "#fff7ed", TextColor: "#1c1917",
		FontHeading: "Space Grotesk", FontBody: "Inter",
	}
	BrandAudit = Branding{
		PrimaryColor: "#0e7490", SecondaryColor: "#475569",
		BgColor: "#f8fafc", TextColor: "#0f172a",
		FontHeading: "Space Grotesk", FontBody: "Inter",
	}
	BrandMonogram = Branding{
		PrimaryColor: "#d4af37", SecondaryColor: "#1c1917",
		BgColor: "#fefdf8", TextColor: "#0c0a09",
		FontHeading: "Space Grotesk", FontBody: "Inter",
	}
)
