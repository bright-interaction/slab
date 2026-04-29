// Package agent provides the AI agent API logic: context building, guardrails, and operations.
package agent

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// ContextBuilder assembles the full context response for an AI agent.
type ContextBuilder struct {
	queries *store.Queries
}

func NewContextBuilder(queries *store.Queries) *ContextBuilder {
	return &ContextBuilder{queries: queries}
}

// SiteContext is the full context payload returned to AI agents.
type SiteContext struct {
	Site             SiteInfo              `json:"site"`
	Structure        Structure             `json:"structure"`
	Knowledgebase    []KBEntry             `json:"knowledgebase"`
	Components       []ComponentInfo       `json:"components"`
	CSSClasses       []CSSClassInfo        `json:"css_classes"`
	Constraints      Constraints           `json:"constraints"`
	Architecture     ArchitectureInfo      `json:"architecture"`
	DesignReferences []DesignReferenceInfo `json:"design_references"`
	// PendingSetup lists configuration gaps the agent should resolve before
	// declaring a site "done". Empty when the site is fully configured.
	// The CLAUDE.md template tells agents to walk through this list with the
	// user as the first step of any session.
	PendingSetup []SetupTask `json:"pending_setup"`
	// Personalization tells the agent what visitor metadata the CRM has
	// pushed to this site, so it can author conditional blocks. Phase 18.4.
	Personalization PersonalizationInfo `json:"personalization"`
	// Endpoints surfaces the admin and agent API paths the agent uses
	// most often, so it doesn't need to look them up in docs. Each entry
	// names the use case alongside the HTTP method + URL template. Added
	// 2026-04-29 alongside the per-block code-view + page View Source
	// dialogs so the agent can fetch the same source the human admin
	// sees.
	Endpoints EndpointsInfo `json:"endpoints"`
	// BlockSchemas tells the agent which JSON keys each block_type
	// expects under data_json. The keys come from atomicsite's renderer
	// (internal/builder/pages.go) and the Text Mode UI's canonical text
	// field set. Use this to put copy in the right field instead of
	// inventing new keys that the renderer ignores.
	BlockSchemas []BlockSchemaInfo `json:"block_schemas"`
	// EditingModes documents the two human surfaces an authoring agent
	// shares the canvas with. The agent uses this to phrase its outputs
	// (e.g. "your headline is now editable in Text mode at /sites/{id}/
	// pages/{id}").
	EditingModes []EditingModeInfo `json:"editing_modes"`
}

// EndpointInfo names one URL template the agent calls. Method is the HTTP
// verb; Path uses {var} placeholders matching chi route params. Use lists
// the persona-level intent (humans + agent share these endpoints).
type EndpointInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Use    string `json:"use"`
}

// EndpointsInfo groups EndpointInfo entries by area so the agent can
// scan the relevant set quickly.
type EndpointsInfo struct {
	Pages         []EndpointInfo `json:"pages"`
	Blocks        []EndpointInfo `json:"blocks"`
	GlobalBlocks  []EndpointInfo `json:"global_blocks"`
	Media         []EndpointInfo `json:"media"`
	Build         []EndpointInfo `json:"build"`
	Preview       []EndpointInfo `json:"preview"`
}

// BlockSchemaInfo describes the keys data_json should carry for one
// block_type. Use this as the source of truth for "where does the
// headline go" when authoring a block.
type BlockSchemaInfo struct {
	BlockType string             `json:"block_type"`
	Use       string             `json:"use"`
	TextKeys  []BlockSchemaField `json:"text_keys"`
	OtherKeys []BlockSchemaField `json:"other_keys"`
}

// BlockSchemaField is one key inside a block's data_json: its name, a
// short label for human display, and whether multiline copy is expected
// (multiline keys belong in textareas; single-line keys are short copy
// like CTAs and titles).
type BlockSchemaField struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Multiline bool   `json:"multiline"`
}

// EditingModeInfo documents one of the two authoring surfaces the agent
// shares with humans on the page editor.
type EditingModeInfo struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// PersonalizationInfo summarises what the agent needs to author conditional
// blocks: whether the feature is enabled, the identity-freshness window,
// the actual metadata keys the CRM has been pushing, and a few example
// conditions to seed pattern recognition.
type PersonalizationInfo struct {
	Enabled             bool     `json:"enabled"`
	IdentityMaxAgeDays  int      `json:"identity_max_age_days"`
	KnownKeys           []string `json:"known_keys"`
	ExampleConditions   []string `json:"example_conditions"`
}

// SetupTask represents one configuration gap the agent should close. The
// agent reads this list, asks the user when needed, and calls the listed
// endpoint to apply the fix.
type SetupTask struct {
	// ID is a stable string the agent can match on (e.g. across two consecutive
	// /api/agent/context calls).
	ID string `json:"id"`
	// Category groups related tasks: branding, profile, seo, analytics,
	// content, media.
	Category string `json:"category"`
	// Title is a one-line description of what is missing.
	Title string `json:"title"`
	// Why explains the consequence of leaving this unfixed (eval impact,
	// compliance, conversion).
	Why string `json:"why"`
	// Action describes what the agent should do.
	Action string `json:"action"`
	// Endpoint is the primary API path the agent should call to resolve.
	Endpoint string `json:"endpoint"`
	// Severity is "required" (blocks an A+ build) or "recommended"
	// (improves quality but the build will still pass without it).
	Severity string `json:"severity"`
}

// DesignReferenceInfo surfaces a fetched GitHub bundle so the AI agent
// has the user's preferred design vocabulary to compose with. The bundle
// is read-only pattern reference, not a code-copy mechanism.
type DesignReferenceInfo struct {
	URL     string         `json:"url"`
	Label   string         `json:"label"`
	RefType string         `json:"ref_type"`
	Bundle  map[string]any `json:"bundle"`
}

type SiteInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Domain   string   `json:"domain"`
	Lang     string   `json:"lang"`
	Status   string   `json:"status"`
	Branding Branding `json:"branding"`
	SEO      SEOInfo  `json:"seo"`
}

type Branding struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	BgColor        string `json:"bg_color"`
	TextColor      string `json:"text_color"`
	// Phase 12.9: extra colour slots so the agent has the same palette
	// the human admin edits in the Branding UI. accent_color = "" means
	// "fall back to primary_color" (preserved by the builder).
	SurfaceColor   string `json:"surface_color"`
	BorderColor    string `json:"border_color"`
	MutedColor     string `json:"muted_color"`
	AccentColor    string `json:"accent_color"`
	OnPrimaryColor string `json:"on_primary_color"`
	FontHeading    string `json:"font_heading"`
	FontBody       string `json:"font_body"`
}

type SEOInfo struct {
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	OgImageID       string `json:"og_image_id"`
	FaviconID       string `json:"favicon_id"`
}

type Structure struct {
	Pages        []PageInfo       `json:"pages"`
	GlobalBlocks []GlobalBlockInfo `json:"global_blocks"`
	Silos        []SiloInfo       `json:"silos"`
}

type PageInfo struct {
	ID               string      `json:"id"`
	Title            string      `json:"title"`
	Slug             string      `json:"slug"`
	Status           string      `json:"status"`
	Layout           string      `json:"layout"`
	SortOrder        int64       `json:"sort_order"`
	ShowInNav        bool        `json:"show_in_nav"`
	NavLabel         string      `json:"nav_label"`
	HideGlobalBlocks bool        `json:"hide_global_blocks"`
	Blocks           []BlockInfo `json:"blocks"`
}

type BlockInfo struct {
	ID        string `json:"id"`
	BlockType string `json:"block_type"`
	SortOrder int64  `json:"sort_order"`
	DataJSON  any    `json:"data"`
	StyleJSON any    `json:"style"`
	IsVisible bool   `json:"is_visible"`
}

type GlobalBlockInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slot      string `json:"slot"`
	BlockType string `json:"block_type"`
	DataJSON  any    `json:"data"`
	StyleJSON any    `json:"style"`
	IsActive  bool   `json:"is_active"`
}

type SiloInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SlugPrefix string `json:"slug_prefix"`
	SortOrder  int64  `json:"sort_order"`
}

type KBEntry struct {
	Category string `json:"category"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

type ComponentInfo struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	PropsSchema any   `json:"props_schema"`
	CSSClasses []string `json:"css_classes"`
	UsageNote  string `json:"usage_note"`
}

type CSSClassInfo struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	CSS       string `json:"css"`
	UsageNote string `json:"usage_note"`
}

type Constraints struct {
	AllowedBlockTypes  []string `json:"allowed_block_types"`
	ForbiddenPatterns  []string `json:"forbidden_patterns"`
	AllowedHosts       []string `json:"allowed_hosts"`
	MaxBlocksPerPage   int      `json:"max_blocks_per_page"`
	MaxURLDepth        int      `json:"max_url_depth"`
	RequiredBlocks     map[string][]string `json:"required_blocks"`
}

type ArchitectureInfo struct {
	StructureType string `json:"structure_type"`
	MaxDepth      int    `json:"max_depth"`
}

// Build assembles the complete context for a site.
func (b *ContextBuilder) Build(ctx context.Context, siteID string) (*SiteContext, error) {
	site, err := b.queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	pages, err := b.queries.ListPagesBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	var pageInfos []PageInfo
	for _, p := range pages {
		blocks, err := b.queries.ListBlocksByPage(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		var blockInfos []BlockInfo
		for _, bl := range blocks {
			blockInfos = append(blockInfos, BlockInfo{
				ID:        bl.ID,
				BlockType: bl.BlockType,
				SortOrder: bl.SortOrder,
				DataJSON:  parseJSONField(bl.DataJson),
				StyleJSON: parseJSONField(bl.StyleJson),
				IsVisible: bl.IsVisible == 1,
			})
		}
		pageInfos = append(pageInfos, PageInfo{
			ID:               p.ID,
			Title:            p.Title,
			Slug:             p.Slug,
			Status:           p.Status,
			Layout:           p.Layout,
			SortOrder:        p.SortOrder,
			ShowInNav:        p.ShowInNav == 1,
			NavLabel:         p.NavLabel,
			HideGlobalBlocks: p.HideGlobalBlocks == 1,
			Blocks:           blockInfos,
		})
	}

	globalBlocks, err := b.queries.ListActiveGlobalBlocksBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	var gbInfos []GlobalBlockInfo
	for _, gb := range globalBlocks {
		gbInfos = append(gbInfos, GlobalBlockInfo{
			ID:        gb.ID,
			Name:      gb.Name,
			Slot:      gb.Slot,
			BlockType: gb.BlockType,
			DataJSON:  parseJSONField(gb.DataJson),
			StyleJSON: parseJSONField(gb.StyleJson),
			IsActive:  gb.IsActive == 1,
		})
	}

	silos, err := b.queries.ListSilosBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	var siloInfos []SiloInfo
	for _, s := range silos {
		siloInfos = append(siloInfos, SiloInfo{
			ID:         s.ID,
			Name:       s.Name,
			SlugPrefix: s.SlugPrefix,
			SortOrder:  s.SortOrder,
		})
	}

	kbEntries, err := b.queries.ListActiveKnowledgebaseBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	var kbInfos []KBEntry
	for _, kb := range kbEntries {
		kbInfos = append(kbInfos, KBEntry{
			Category: kb.Category,
			Title:    kb.Title,
			Content:  kb.Content,
		})
	}

	components, err := b.queries.ListComponentsBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	var compInfos []ComponentInfo
	for _, c := range components {
		var cssClasses []string
		_ = json.Unmarshal([]byte(c.CssClasses), &cssClasses)
		compInfos = append(compInfos, ComponentInfo{
			Name:        c.Name,
			Category:    c.Category,
			PropsSchema: parseJSONField(c.PropsSchema),
			CSSClasses:  cssClasses,
			UsageNote:   c.UsageNote,
		})
	}

	cssClasses, err := b.queries.ListCSSClassesBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	var cssInfos []CSSClassInfo
	for _, cc := range cssClasses {
		cssInfos = append(cssInfos, CSSClassInfo{
			Name:      cc.Name,
			Category:  cc.Category,
			CSS:       cc.Css,
			UsageNote: cc.UsageNote,
		})
	}

	constraints := b.buildConstraints(ctx, siteID)

	// Design references: pull every saved GitHub bundle so the agent can
	// borrow vocabulary (component shapes, naming, tailwind config). The
	// bundle JSON is whatever the design-references handler stored at
	// fetch / refresh time. Failures here are non-fatal: an agent without
	// references is still a useful agent.
	var refInfos []DesignReferenceInfo
	if refRows, err := b.queries.ListDesignReferences(ctx, siteID); err == nil {
		for _, r := range refRows {
			var bundle map[string]any
			_ = json.Unmarshal([]byte(r.FetchedJson), &bundle)
			refInfos = append(refInfos, DesignReferenceInfo{
				URL:     r.Url,
				Label:   r.Label,
				RefType: r.RefType,
				Bundle:  bundle,
			})
		}
	}

	arch, _ := b.queries.GetSiteArchitecture(ctx, siteID)
	archInfo := ArchitectureInfo{
		StructureType: "soft-silo",
		MaxDepth:      3,
	}
	if arch.ID != "" {
		archInfo.StructureType = arch.StructureType
		archInfo.MaxDepth = int(arch.MaxDepth)
	}

	pending := b.computePendingSetup(ctx, siteID, site, pageInfos)
	personalization := b.computePersonalization(ctx, siteID)

	return &SiteContext{
		Site: SiteInfo{
			ID:     site.ID,
			Name:   site.Name,
			Domain: site.Domain,
			Lang:   site.Lang,
			Status: site.Status,
			Branding: Branding{
				PrimaryColor:   site.PrimaryColor,
				SecondaryColor: site.SecondaryColor,
				BgColor:        site.BgColor,
				TextColor:      site.TextColor,
				SurfaceColor:   site.SurfaceColor,
				BorderColor:    site.BorderColor,
				MutedColor:     site.MutedColor,
				AccentColor:    site.AccentColor,
				OnPrimaryColor: site.OnPrimaryColor,
				FontHeading:    site.FontHeading,
				FontBody:       site.FontBody,
			},
			SEO: SEOInfo{
				MetaTitle:       site.MetaTitle,
				MetaDescription: site.MetaDescription,
				OgImageID:       site.OgImageID,
				FaviconID:       site.FaviconID,
			},
		},
		Structure: Structure{
			Pages:        pageInfos,
			GlobalBlocks: gbInfos,
			Silos:        siloInfos,
		},
		Knowledgebase:    kbInfos,
		Components:       compInfos,
		CSSClasses:       cssInfos,
		Constraints:      constraints,
		Architecture:     archInfo,
		DesignReferences: refInfos,
		PendingSetup:     pending,
		Personalization:  personalization,
		Endpoints:        defaultEndpoints(),
		BlockSchemas:     defaultBlockSchemas(),
		EditingModes:     defaultEditingModes(),
	}, nil
}

// defaultEndpoints returns the endpoint catalogue surfaced to AI agents.
// Static for now; if a future endpoint becomes site-conditional it can
// move into the Build closure.
func defaultEndpoints() EndpointsInfo {
	return EndpointsInfo{
		Pages: []EndpointInfo{
			{Method: "GET", Path: "/api/agent/pages", Use: "List published + draft pages."},
			{Method: "POST", Path: "/api/agent/pages", Use: "Create a page. Title + slug required; layout defaults to 'default'."},
			{Method: "PATCH", Path: "/api/agent/pages/{slug}", Use: "Update title, slug, status, meta_title, meta_description, og_image_id, layout, hide_global_blocks, no_index, canonical_url. Slug-addressed."},
		},
		Blocks: []EndpointInfo{
			{Method: "GET", Path: "/api/agent/pages/{slug}/blocks", Use: "List blocks on a page, ordered by sort_order."},
			{Method: "POST", Path: "/api/agent/pages/{slug}/blocks", Use: "Append a block. block_type + data are required; data should match the block_schemas catalogue below."},
			{Method: "PATCH", Path: "/api/agent/blocks/{blockID}", Use: "Patch a block's data, style, sort_order, or is_visible without re-creating it."},
			{Method: "DELETE", Path: "/api/agent/blocks/{blockID}", Use: "Remove a block from a page."},
		},
		GlobalBlocks: []EndpointInfo{
			{Method: "PUT", Path: "/api/agent/global/{slot}", Use: "Upsert the header or footer global block. Slot must be 'header' or 'footer'. Body: name, block_type, data."},
		},
		Media: []EndpointInfo{
			{Method: "GET", Path: "/api/agent/media", Use: "List uploaded media."},
			{Method: "POST", Path: "/api/agent/media", Use: "Multipart upload. Optional folder field puts files under a folder; 'brand' is the system folder for favicon, OG image, logos."},
			{Method: "POST", Path: "/api/agent/media/from-url", Use: "Server-side fetch with SSRF guard. Same folder field as multipart."},
			{Method: "POST", Path: "/api/agent/media/from-base64", Use: "For agents that emit base64 but not multipart. Same folder field."},
		},
		Build: []EndpointInfo{
			{Method: "POST", Path: "/api/agent/build", Use: "Trigger a full build. Returns a build_id for status polling."},
			{Method: "GET", Path: "/api/agent/evaluation/{buildID}", Use: "Read the post-build evaluation results so the agent can self-correct against the 96 Site Inspector parity checks."},
		},
		Preview: []EndpointInfo{
			{Method: "GET", Path: "/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}/preview", Use: "Returns the rendered Astro source for one block. Use this when the user asks 'show me the code' for a specific section. Same source the build pipeline writes to disk."},
			{Method: "GET", Path: "/api/sites/{siteID}/pages/{pageID}/preview", Use: "Returns the assembled .astro source for the whole page. Use to inspect what bun build will see without triggering a build."},
		},
	}
}

// defaultBlockSchemas tells the agent which JSON keys each block type
// recognises. Aligns with internal/builder/pages.go renderDataBlock plus
// the canonical text key set that the Text Mode editor surfaces inline.
func defaultBlockSchemas() []BlockSchemaInfo {
	headingTextKeys := []BlockSchemaField{
		{Key: "eyebrow", Label: "Eyebrow", Multiline: false},
		{Key: "heading", Label: "Heading", Multiline: false},
		{Key: "subheading", Label: "Subheading", Multiline: true},
		{Key: "text", Label: "Text", Multiline: true},
		{Key: "cta_text", Label: "CTA label", Multiline: false},
	}
	heroTextKeys := []BlockSchemaField{
		{Key: "eyebrow", Label: "Eyebrow", Multiline: false},
		{Key: "headline", Label: "Headline", Multiline: false},
		{Key: "subheading", Label: "Subheading", Multiline: true},
		{Key: "cta_text", Label: "Primary CTA label", Multiline: false},
		{Key: "secondary_label", Label: "Secondary CTA label", Multiline: false},
	}
	imageTextKeys := []BlockSchemaField{
		{Key: "alt", Label: "Image alt", Multiline: false},
		{Key: "caption", Label: "Caption", Multiline: false},
	}
	quoteTextKeys := []BlockSchemaField{
		{Key: "quote", Label: "Quote", Multiline: true},
		{Key: "attribution", Label: "Attribution", Multiline: false},
	}
	return []BlockSchemaInfo{
		{
			BlockType: "hero",
			Use:       "Above-the-fold attention-grab. One per page. Drives a primary CTA.",
			TextKeys:  heroTextKeys,
			OtherKeys: []BlockSchemaField{
				{Key: "cta_url", Label: "Primary CTA URL"},
				{Key: "secondary_url", Label: "Secondary CTA URL"},
				{Key: "image_id", Label: "Background image media ID"},
			},
		},
		{
			BlockType: "text",
			Use:       "Rich body copy block. Renders heading + text + optional CTA.",
			TextKeys:  headingTextKeys,
			OtherKeys: []BlockSchemaField{
				{Key: "cta_url", Label: "CTA URL"},
				{Key: "alignment", Label: "Alignment ('left' | 'center')"},
			},
		},
		{
			BlockType: "feature_grid",
			Use:       "3-to-4-up grid of feature cards.",
			TextKeys:  []BlockSchemaField{{Key: "heading", Label: "Heading"}, {Key: "subheading", Label: "Subheading", Multiline: true}},
			OtherKeys: []BlockSchemaField{{Key: "items", Label: "Array of {title, body, icon}"}},
		},
		{
			BlockType: "cta",
			Use:       "Standalone CTA section. Use heading + text to set up the action.",
			TextKeys: []BlockSchemaField{
				{Key: "heading", Label: "Heading"},
				{Key: "text", Label: "Lead-in text", Multiline: true},
				{Key: "cta_text", Label: "CTA label"},
			},
			OtherKeys: []BlockSchemaField{
				{Key: "cta_url", Label: "CTA URL"},
				{Key: "variant", Label: "Variant ('primary' | 'secondary')"},
			},
		},
		{
			BlockType: "image",
			Use:       "Single image. Always include a descriptive alt; never 'image' or 'photo'.",
			TextKeys:  imageTextKeys,
			OtherKeys: []BlockSchemaField{
				{Key: "image_id", Label: "Media ID"},
				{Key: "width", Label: "Pixel width (required, prevents CLS)"},
				{Key: "height", Label: "Pixel height (required, prevents CLS)"},
			},
		},
		{
			BlockType: "quote",
			Use:       "Testimonial / pull quote.",
			TextKeys:  quoteTextKeys,
		},
		{
			BlockType: "raw",
			Use:       "Escape hatch for custom HTML. Pass `html` directly. Last resort, every other block type is preferred.",
			OtherKeys: []BlockSchemaField{{Key: "html", Label: "Raw HTML, no scripts."}},
		},
		{
			BlockType: "component",
			Use:       "Render a custom Astro component declared in the site's components catalogue.",
			OtherKeys: []BlockSchemaField{
				{Key: "component", Label: "Component name (kebab-case)"},
				{Key: "props", Label: "Object of props the component accepts"},
			},
		},
	}
}

// defaultEditingModes documents the two authoring surfaces on the page
// editor. The agent uses these to phrase its responses ("toggle to Text
// mode" / "open the </> code view" / "click View source").
func defaultEditingModes() []EditingModeInfo {
	return []EditingModeInfo{
		{
			Name:        "Blocks",
			URL:         "/sites/{site_id}/pages/{page_id}",
			Description: "Canonical full editor. Drag, add, delete, expand each block to edit every field. Per-block </> toggle reveals the rendered Astro source for that block.",
		},
		{
			Name:        "Text mode",
			URL:         "/sites/{site_id}/pages/{page_id}",
			Description: "Tight inline-editable view of every text field across every visible block. Click any field, edit, blur to autosave. Use this surface when the user asks for copy or SEO edits.",
		},
		{
			Name:        "View source",
			URL:         "/sites/{site_id}/pages/{page_id}",
			Description: "Dialog showing the assembled .astro source for the whole page. Mirrors GET /api/sites/{id}/pages/{id}/preview.",
		},
		{
			Name:        "Global blocks",
			URL:         "/sites/{site_id}/global-blocks",
			Description: "Site-wide header and footer slots. Edit once, applies to every page (unless the page sets hide_global_blocks=true for landing-page suppression).",
		},
	}
}

// computePersonalization builds the personalization summary surfaced to
// the agent. known_keys is the DISTINCT set of metadata keys the CRM has
// pushed for this site (read via a raw json_each query because sqlc cannot
// model SQLite's JSON1 virtual columns). Examples are static so the agent
// has shape recognition even on a fresh site.
func (b *ContextBuilder) computePersonalization(ctx context.Context, siteID string) PersonalizationInfo {
	out := PersonalizationInfo{
		IdentityMaxAgeDays: 30,
		KnownKeys:          []string{},
		ExampleConditions: []string{
			`lead_score >= 60`,
			`lifecycle == "SQL"`,
			`last_topic == "compliance"`,
			`name present`,
			`returning == "true"`,
			`last_topic in compliance,gdpr,nis2`,
		},
	}
	settings, _ := b.queries.ListSettingsBySite(ctx, siteID)
	for _, s := range settings {
		if s.Category != "analytics" {
			continue
		}
		switch s.Key {
		case "personalization_enabled":
			out.Enabled = boolFromSetting(s.Value, false)
		case "identity_max_age_days":
			if n, err := strconv.Atoi(strings.TrimSpace(s.Value)); err == nil && n > 0 {
				out.IdentityMaxAgeDays = n
			}
		}
	}
	out.KnownKeys = b.listVisitorMetadataKeys(ctx, siteID)
	return out
}

// listVisitorMetadataKeys runs a raw query that sqlc cannot model
// (SQLite's json_each returns key/value as virtual columns, which sqlc's
// static analyzer rejects). Best-effort: returns an empty list on error
// rather than aborting the whole context call.
func (b *ContextBuilder) listVisitorMetadataKeys(ctx context.Context, siteID string) []string {
	const q = `SELECT DISTINCT je.key
FROM visit_sessions vs, json_each(vs.metadata_json) je
WHERE vs.site_id = ? AND vs.metadata_json != '{}'
ORDER BY je.key`
	keys := []string{}
	rows, err := b.queries.Raw().QueryContext(ctx, q, siteID)
	if err != nil {
		return keys
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

// computePendingSetup inspects the site's profile, settings, and content to
// produce a list of configuration gaps the agent should resolve. Each task
// names a concrete endpoint so the agent does not have to guess.
//
// Failures here are non-fatal; we return whatever we managed to compute and
// fall back to an empty list if a query errors. Better to ship a partial
// pending list than to crash the entire context call.
func (b *ContextBuilder) computePendingSetup(ctx context.Context, siteID string, _ any, pages []PageInfo) []SetupTask {
	out := []SetupTask{}

	siteRow, err := b.queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return out
	}

	profile, _ := b.queries.GetSiteProfile(ctx, siteID)

	settings, _ := b.queries.ListSettingsBySite(ctx, siteID)
	settingMap := map[string]string{}
	for _, s := range settings {
		settingMap[s.Category+"."+s.Key] = s.Value
	}

	push := func(t SetupTask) {
		out = append(out, t)
	}

	// --- Profile (drives Organization JSON-LD, security.txt, legal pages) ---
	if profile.BusinessName == "" {
		push(SetupTask{
			ID:       "profile.business_name",
			Category: "profile",
			Title:    "Add the business name",
			Why:      "Drives Organization JSON-LD schema, legal pages, and security.txt. Missing it costs 3 SEO/GEO eval checks.",
			Action:   "Ask the user for the legal business name, then PATCH it onto the profile.",
			Endpoint: "PATCH /api/agent/profile",
			Severity: "required",
		})
	}
	if profile.ContactEmail == "" {
		push(SetupTask{
			ID:       "profile.contact_email",
			Category: "profile",
			Title:    "Add a public contact email",
			Why:      "Required for the auto-generated imprint and security.txt. AI search engines look for it as a trust signal.",
			Action:   "Ask the user for a public-facing contact email and PATCH it onto the profile.",
			Endpoint: "PATCH /api/agent/profile",
			Severity: "required",
		})
	}
	if profile.AddressLine1 == "" || profile.City == "" {
		push(SetupTask{
			ID:       "profile.address",
			Category: "profile",
			Title:    "Add a postal address",
			Why:      "Drives Organization JSON-LD (address fields) and the auto-generated imprint page. Required for B2B, B2C-local, and ecommerce kits.",
			Action:   "Ask the user for street + city + postal code, then PATCH the profile.",
			Endpoint: "PATCH /api/agent/profile",
			Severity: "recommended",
		})
	}

	// --- SEO (drives Organization JSON-LD logo + sameAs) ---
	if siteRow.MetaTitle == "" {
		push(SetupTask{
			ID:       "seo.meta_title",
			Category: "seo",
			Title:    "Add a default meta title",
			Why:      "Falls through to every page that does not override it. Empty title fails the on-page SEO eval check.",
			Action:   "Compose a 30-60 char title for the site and PATCH it via /api/agent/branding (meta_title field).",
			Endpoint: "PATCH /api/agent/branding",
			Severity: "required",
		})
	}
	if siteRow.MetaDescription == "" {
		push(SetupTask{
			ID:       "seo.meta_description",
			Category: "seo",
			Title:    "Add a default meta description",
			Why:      "Same fall-through as meta_title. 120-160 chars. Drives SERP and AI-search snippets.",
			Action:   "Compose a 120-160 char description and PATCH it via /api/agent/branding (meta_description field).",
			Endpoint: "PATCH /api/agent/branding",
			Severity: "required",
		})
	}
	if settingMap["seo.logo_url"] == "" {
		push(SetupTask{
			ID:       "seo.logo_url",
			Category: "seo",
			Title:    "Add a logo URL",
			Why:      "Required by Organization JSON-LD (logo field) and the GEO 'Organization Schema Completeness' check.",
			Action:   "Upload or link a logo, then PATCH /api/agent/settings with category=seo, key=logo_url.",
			Endpoint: "PATCH /api/agent/settings",
			Severity: "recommended",
		})
	}
	if settingMap["seo.same_as"] == "" {
		push(SetupTask{
			ID:       "seo.same_as",
			Category: "seo",
			Title:    "Add social profile URLs",
			Why:      "Drives the Organization JSON-LD sameAs array. Strong AI-search trust signal (LinkedIn, GitHub, etc.).",
			Action:   "Ask the user for their social URLs (newline-separated) and PATCH /api/agent/settings (seo.same_as).",
			Endpoint: "PATCH /api/agent/settings",
			Severity: "recommended",
		})
	}

	// --- Branding ---
	if siteRow.PrimaryColor == "" || siteRow.PrimaryColor == "#D4AF37" {
		push(SetupTask{
			ID:       "branding.primary_color",
			Category: "branding",
			Title:    "Pick a brand primary colour",
			Why:      "The wizard default (#D4AF37) is still in place. Real brand colour is the single biggest visual change you can make.",
			Action:   "Ask the user for their brand primary, then PATCH /api/agent/branding (primary_color field).",
			Endpoint: "PATCH /api/agent/branding",
			Severity: "recommended",
		})
	}
	if siteRow.OgImageID == "" {
		push(SetupTask{
			ID:       "seo.og_image",
			Category: "media",
			Title:    "Set an Open Graph image",
			Why:      "Drives social previews and is referenced as a Schema.org logo fallback. Failing this loses 1 SEO eval check.",
			Action:   "Generate or upload a 1200x630 image via /api/agent/media (set folder=brand), then PATCH /api/agent/branding (og_image_id).",
			Endpoint: "PATCH /api/agent/branding",
			Severity: "recommended",
		})
	}
	if siteRow.FaviconID == "" {
		push(SetupTask{
			ID:       "seo.favicon",
			Category: "media",
			Title:    "Set a favicon",
			Why:      "Browsers default to a placeholder; AI agents and crawlers pick up the favicon as a brand signal.",
			Action:   "Generate or upload a square favicon via /api/agent/media (folder=brand), then PATCH /api/agent/branding (favicon_id).",
			Endpoint: "PATCH /api/agent/branding",
			Severity: "recommended",
		})
	}

	// --- Analytics + consent ---
	atomicsiteTracking := boolFromSetting(settingMap["analytics.atomicsite_tracking_enabled"], true)
	cookieproofOn := boolFromSetting(settingMap["analytics.cookieproof_enabled"], false)
	customBanner := strings.TrimSpace(settingMap["analytics.cookie_banner_snippet"]) != ""
	if atomicsiteTracking && !cookieproofOn && !customBanner {
		push(SetupTask{
			ID:       "analytics.consent",
			Category: "analytics",
			Title:    "Add a consent banner",
			Why:      "Tracking is on but no banner is in place. EU GDPR requires consent before non-essential tracking; failing this breaks the privacy eval and risks fines.",
			Action:   "Either flip on CookieProof (PATCH /api/agent/settings analytics.cookieproof_enabled=1 + cookieproof_org_id) or paste a banner snippet (analytics.cookie_banner_snippet) for Cookiebot, OneTrust, Termly, etc.",
			Endpoint: "PATCH /api/agent/settings",
			Severity: "required",
		})
	}
	if settingMap["analytics.crm_webhook_url"] == "" {
		push(SetupTask{
			ID:       "analytics.crm_webhook",
			Category: "analytics",
			Title:    "Wire a CRM webhook (optional)",
			Why:      "Without a CRM webhook, identified visitor events (form submits, opt-ins) are recorded but not forwarded anywhere. Plug in any HTTPS endpoint; we sign payloads with HMAC-SHA256.",
			Action:   "Ask the user for their CRM webhook URL (HubSpot, Salesforce, BrightCRM, n8n) and a shared secret, then PATCH /api/agent/settings (analytics.crm_webhook_url, analytics.crm_webhook_secret).",
			Endpoint: "PATCH /api/agent/settings",
			Severity: "recommended",
		})
	}

	// --- Content ---
	publishedPages := 0
	for _, p := range pages {
		if p.Status == "published" {
			publishedPages++
		}
	}
	if publishedPages == 0 {
		push(SetupTask{
			ID:       "content.publish_home",
			Category: "content",
			Title:    "Publish at least one page",
			Why:      "Every page is still draft. The site has nothing to deploy.",
			Action:   "PATCH /api/agent/pages/{slug} with status='published' once the home page is ready.",
			Endpoint: "PATCH /api/agent/pages/{slug}",
			Severity: "required",
		})
	}
	if len(pages) <= 1 {
		push(SetupTask{
			ID:       "content.add_pages",
			Category: "content",
			Title:    "Add more pages",
			Why:      "A one-page site limits SEO and conversion. Add at least Pricing, About, and Contact unless the user has explicitly chosen a one-pager.",
			Action:   "Ask the user what additional pages they want, then POST /api/agent/pages.",
			Endpoint: "POST /api/agent/pages",
			Severity: "recommended",
		})
	}

	return out
}

// boolFromSetting parses the limited bool encoding the settings table uses
// ("1" / "true" / "yes" / "on" -> true; everything else -> the default).
func boolFromSetting(v string, def bool) bool {
	if v == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

func (b *ContextBuilder) buildConstraints(ctx context.Context, siteID string) Constraints {
	rules, _ := b.queries.ListActiveGuardrailsBySite(ctx, siteID)

	c := Constraints{
		MaxBlocksPerPage: 50,
		MaxURLDepth:      3,
		RequiredBlocks:   make(map[string][]string),
	}

	for _, r := range rules {
		switch r.RuleType {
		case "allow_block_type":
			c.AllowedBlockTypes = append(c.AllowedBlockTypes, r.Value)
		case "forbid_pattern":
			c.ForbiddenPatterns = append(c.ForbiddenPatterns, r.Value)
		case "allowed_host":
			c.AllowedHosts = append(c.AllowedHosts, r.Value)
		case "max_blocks":
			var n int
			if err := json.Unmarshal([]byte(r.Value), &n); err == nil {
				c.MaxBlocksPerPage = n
			}
		case "require_block":
			var blocks []string
			if err := json.Unmarshal([]byte(r.Value), &blocks); err == nil {
				c.RequiredBlocks[r.Target] = blocks
			}
		}
	}

	// Add allowed scripts as allowed hosts
	scripts, _ := b.queries.ListAllowedScriptsBySite(ctx, siteID)
	for _, s := range scripts {
		if s.IsActive == 1 {
			c.AllowedHosts = append(c.AllowedHosts, s.Domain)
		}
	}

	return c
}

func parseJSONField(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	return v
}
