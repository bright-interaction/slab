package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// validArchetypes is the curated set of vibe archetypes a page can be
// locked to. Mirrors the playbook's vibe_archetypes catalog; new
// archetypes added there should land here too so set_page_archetype
// validates against the same surface the playbook describes.
var validArchetypes = map[string]string{
	"mesh":          "Cool gradient, technical, tools / infra audience. hero_graphic = mesh | gradient-orb | globe-wire.",
	"pulse":         "Warm, kinetic, consumer / agency / local. hero_graphic = pulse | circuit | globe-wire.",
	"monogram":      "Editorial, founder-led, portfolio. hero_graphic = monogram, or text-only with monogram_char set.",
	"audit-receipt": "Audit / compliance / security. hero_graphic = audit-receipt; pair with stat_grid + visible audit scores.",
}

// registerArchetypeTools wires set_page_archetype + get_page_archetype.
// The lock makes every subsequent create_block / update_block on the
// page run the archetype_drift check; clearing the lock disables it.
func (s *Server) registerArchetypeTools() {
	register := func(t Tool) { s.tools[t.Name] = t }

	register(Tool{
		Name:        "set_page_archetype",
		Description: "Locks a page to one of the playbook's vibe archetypes (mesh | pulse | monogram | audit-receipt). Subsequent create_block / update_block calls on the page run an archetype_drift check that flags hero blocks whose hero_graphic does not fit the archetype. Pass archetype='' to clear the lock. Use this once near the start of authoring so the rest of the session keeps the vibe coherent. Required: page_slug, archetype.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"page_slug":{"type":"string"},
				"archetype":{"type":"string"}
			},
			"required":["page_slug","archetype"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				PageSlug  string `json:"page_slug"`
				Archetype string `json:"archetype"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			arch := strings.TrimSpace(args.Archetype)
			if arch != "" {
				if _, ok := validArchetypes[arch]; !ok {
					return "", fmt.Errorf("archetype must be one of mesh|pulse|monogram|audit-receipt or empty to clear; got %q", arch)
				}
			}
			page, err := s.queries.GetPageBySiteAndSlug(ctx, store.GetPageBySiteAndSlugParams{
				SiteID: agent.SiteID,
				Slug:   normalizeSlug(args.PageSlug),
			})
			if err != nil {
				return "", fmt.Errorf("page not found: %s", args.PageSlug)
			}
			if err := s.queries.SetPageArchetype(ctx, store.SetPageArchetypeParams{
				Archetype: arch,
				ID:        page.ID,
			}); err != nil {
				return "", err
			}
			updated, _ := s.queries.GetPageByID(ctx, page.ID)
			return mustJSON(map[string]any{
				"page_id":   updated.ID,
				"slug":      updated.Slug,
				"archetype": updated.Archetype,
				"rationale": validArchetypes[arch],
				"hint":      "Every create_block + update_block on this page now runs the archetype_drift check. Clear with set_page_archetype(page_slug, archetype='') if you want to author off-archetype variants.",
			}), nil
		},
	})

	register(Tool{
		Name:        "get_page_archetype",
		Description: "Returns the archetype lock on a page (gap 6). Empty archetype = no lock. Use this before authoring on an unfamiliar page so you know which archetype's tokens to obey.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"page_slug":{"type":"string"}
			},
			"required":["page_slug"]
		}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				PageSlug string `json:"page_slug"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			page, err := s.queries.GetPageBySiteAndSlug(ctx, store.GetPageBySiteAndSlugParams{
				SiteID: agent.SiteID,
				Slug:   normalizeSlug(args.PageSlug),
			})
			if err != nil {
				return "", errors.New("page not found")
			}
			catalog := make([]map[string]string, 0, len(validArchetypes))
			for name, desc := range validArchetypes {
				catalog = append(catalog, map[string]string{"name": name, "description": desc})
			}
			return mustJSON(map[string]any{
				"page_id":   page.ID,
				"slug":      page.Slug,
				"archetype": page.Archetype,
				"catalog":   catalog,
			}), nil
		},
	})
}
