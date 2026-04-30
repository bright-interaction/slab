// Package agent provides the AI agent API logic: context building, guardrails, and operations.
package agent

import (
	"context"
	"encoding/json"
	"sort"
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
	// SecurityPosture surfaces what the agent needs to reason about
	// embeds, scripts, and CSP without hitting a separate endpoint.
	// trusted_domains carries every active allowed_scripts row keyed by
	// kind so the agent can answer "is cal.com already trusted?"; csp
	// carries the resolved Content-Security-Policy the next build will
	// emit; generated_files names the artifacts atomicsite ships for
	// every site so the agent doesn't try to manually re-author them.
	// Added 2026-04-29 alongside the per-kind CSP allowlist + Security
	// settings UI rebuild.
	SecurityPosture SecurityPostureInfo `json:"security_posture"`
	// I18n surfaces the multi-language story for one site: default lang,
	// declared additional langs, the hreflang strategy, the meta-title
	// and -description templates, and the locale prefixes that have
	// actual published pages. Use this to author copy in the right
	// locale, to know whether you should emit hreflang as part of a
	// page (you don't, atomicsite does it for you), and to pick the
	// right slug prefix when creating a page in a non-default locale.
	I18n I18nInfo `json:"i18n"`
	// SettingsCatalog is the per-key dictionary of every setting the
	// builder, renderer, or nginx config consumes, with current value +
	// type + valid range + writable flag + a link to the human admin
	// page that owns it. Read this BEFORE writing PATCH /api/agent/settings
	// so you know what each key does, what shape its value should be, and
	// whether the agent (you) is permitted to write it. Settings that exist
	// only on the human admin UI (no backend wiring) are deliberately
	// excluded from the catalog so an agent doesn't spend tokens setting
	// values that go nowhere.
	SettingsCatalog SettingsCatalogInfo `json:"settings_catalog"`
	// EvalPlaybook is a hard-coded checklist the agent reads on first
	// session to know exactly how to converge the site to A+ on the
	// atomicsite eval engine. It connects every block-renderer field,
	// every settings_catalog key, and every pending_setup item to the
	// specific eval check it satisfies, so the agent doesn't have to
	// trial-and-error the eval to figure out what each failing check
	// wants. Updated 2026-04-30 after the first end-to-end agent
	// walkthrough surfaced 7 contradictions between documented surface
	// and handler behaviour. Treat as the spec; the eval engine is the
	// scoreboard.
	EvalPlaybook EvalPlaybookInfo `json:"eval_playbook"`
}

// EvalPlaybookInfo is a structured playbook the agent reads to know the
// shape every page should take and which fields/settings drive which
// eval checks. Hard-coded in agent/context.go so every site gets it
// without per-site KB seeding.
type EvalPlaybookInfo struct {
	Goal           string             `json:"goal"`
	PageTemplate   PageTemplate       `json:"page_template"`
	HardRules      []HardRule         `json:"hard_rules"`
	BuildLoop      []string           `json:"build_loop"`
	AdminOnlyItems []AdminOnlyItem    `json:"admin_only_items"`
	VerificationGate []string         `json:"verification_gate"`
}

// PageTemplate is the canonical block sequence the eval engine rewards.
// Use as the default when creating a fresh content page.
type PageTemplate struct {
	Description string             `json:"description"`
	Blocks      []TemplateBlock    `json:"blocks"`
}

type TemplateBlock struct {
	BlockType string `json:"block_type"`
	Required  bool   `json:"required"`
	Notes     string `json:"notes"`
}

type HardRule struct {
	Rule      string `json:"rule"`
	WhyItMatters string `json:"why"`
	HowToApply   string `json:"how"`
}

type AdminOnlyItem struct {
	Setting       string `json:"setting"`
	HumanAdminURL string `json:"human_admin_url"`
	EvalImpact    string `json:"eval_impact"`
}

// I18nInfo is the agent-facing summary of the site's multi-language
// configuration. Read-only context; the agent writes general /
// additional_langs / seo.hreflang_strategy via PATCH /api/agent/settings
// (general + seo categories are agent-writable).
type I18nInfo struct {
	// DefaultLang is the site's primary language (sites.lang). Pages
	// without a /<lang>/ prefix in their slug belong to this locale.
	DefaultLang string `json:"default_lang"`
	// AdditionalLangs is the operator-declared CSV expanded into the
	// list of other languages the site publishes (general.additional_langs).
	AdditionalLangs []string `json:"additional_langs"`
	// Strategy is "path" (default), "subdomain", or "off". Atomicsite
	// optimizes for path-based; subdomain is opt-in for sites already
	// running sv.example.com etc; off disables hreflang emission.
	Strategy string `json:"hreflang_strategy"`
	// LocaleRoots are the actual locale prefixes that have published
	// pages right now (e.g. ["", "/sv"] when /about + /sv/about exist).
	// Use this to pick the right slug prefix when creating a page in a
	// new locale: append the lang code to the prefix list with a slash.
	LocaleRoots []string `json:"locale_roots"`
	// MetaTitleTemplate is seo.meta_title_template applied to every
	// page that doesn't override its own meta_title. Tokens:
	// {page_title}, {site_name}, {lang}, {page_description},
	// {separator}.
	MetaTitleTemplate string `json:"meta_title_template"`
	// MetaDescriptionTemplate is seo.meta_description_template; same
	// tokens as MetaTitleTemplate.
	MetaDescriptionTemplate string `json:"meta_description_template"`
	// CanonicalBase overrides the default https://{domain}. Useful when
	// the public URL differs from the build's domain (CDN, sub-path
	// proxy).
	CanonicalBase string `json:"canonical_base"`
}

// SecurityPostureInfo is the agent-facing summary of the security
// surface for one site. Read-only: the agent cannot mutate any of these
// directly; CSP-modifying writes (allowed_scripts, security settings)
// require the human admin so the AI can never silently widen the attack
// surface. The agent uses this view to (a) avoid asking for an embed
// that already works and (b) tell the human exactly what to whitelist
// when an iframe is needed.
type SecurityPostureInfo struct {
	// TrustedDomains lists every active allowed_scripts row, keyed by
	// kind, so the agent can scan "is cal.com in the frame list?" in
	// one pass.
	TrustedDomains TrustedDomainsByKind `json:"trusted_domains"`
	// AllowedKinds is the closed set of valid kind values, mirrored
	// from internal/handlers/allowed_scripts.go:AllowedKinds.
	AllowedKinds []string `json:"allowed_kinds"`
	// CSP is the resolved Content-Security-Policy the next build will
	// emit (computed via builder.BuildSecurityHeaders).
	CSP string `json:"csp"`
	// HSTS is the resolved Strict-Transport-Security header value.
	HSTS string `json:"hsts"`
	// FrameAncestors is the current frame-ancestors directive value
	// (defaults to 'none', which forbids being embedded).
	FrameAncestors string `json:"frame_ancestors"`
	// HumanAdminURL is the path the agent should point the user at
	// when an iframe / image / script host needs whitelisting.
	HumanAdminURL string `json:"human_admin_url"`
	// GeneratedFiles names every artifact atomicsite emits on every
	// build so the agent doesn't try to re-create them.
	GeneratedFiles []GeneratedFileInfo `json:"generated_files"`
}

// TrustedDomainsByKind groups active allowed_scripts entries by the
// CSP directive they feed.
type TrustedDomainsByKind struct {
	Script  []string `json:"script"`
	Frame   []string `json:"frame"`
	Image   []string `json:"image"`
	Media   []string `json:"media"`
	Connect []string `json:"connect"`
	All     []string `json:"all"`
}

// GeneratedFileInfo names one artifact atomicsite auto-ships.
type GeneratedFileInfo struct {
	Path        string `json:"path"`
	Label       string `json:"label"`
	Description string `json:"description"`
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
	Pages          []EndpointInfo `json:"pages"`
	Blocks         []EndpointInfo `json:"blocks"`
	GlobalBlocks   []EndpointInfo `json:"global_blocks"`
	Media          []EndpointInfo `json:"media"`
	Build          []EndpointInfo `json:"build"`
	Preview        []EndpointInfo `json:"preview"`
	Security       []EndpointInfo `json:"security"`
	TrustedDomains []EndpointInfo `json:"trusted_domains"`
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

	// One settings load shared by SettingsCatalog (and any future
	// context-level consumers). The per-feature compute* functions still
	// load their own slice for now; not worth refactoring those today.
	settingsRows, _ := b.queries.ListSettingsBySite(ctx, siteID)
	settingMap := make(map[string]string, len(settingsRows))
	for _, s := range settingsRows {
		settingMap[s.Category+"."+s.Key] = s.Value
	}
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
		SecurityPosture:  b.computeSecurityPosture(ctx, siteID),
		I18n:             b.computeI18n(ctx, siteID, site, pageInfos),
		SettingsCatalog:  buildSettingsCatalog(siteID, settingMap),
		EvalPlaybook:     defaultEvalPlaybook(siteID),
	}, nil
}

// defaultEvalPlaybook returns the hard-coded agent playbook for converging
// a site to A+ on the atomicsite eval engine. Connects every block-renderer
// field, every settings_catalog key, and every pending_setup item to the
// eval check it satisfies, so an agent reading the context can plan the
// whole setup pass without trial-and-error against the eval.
func defaultEvalPlaybook(siteID string) EvalPlaybookInfo {
	return EvalPlaybookInfo{
		Goal: "Drive the site to 95%+ on agent-writable eval categories (accessibility, seo, privacy partial, performance partial, plus parts of security). The remaining gap is admin-only items the agent surfaces to the human at the end. Read this section first; it tells you what to write and why each write satisfies a specific eval check.",
		PageTemplate: PageTemplate{
			Description: "Canonical block sequence for a content page. Use as the default when creating a fresh page; remove blocks only when the page is short-form (legal, 404).",
			Blocks: []TemplateBlock{
				{BlockType: "hero", Required: true, Notes: "Sort 0. headline -> h1 (REQUIRED for eval Has H1 + Single H1). eyebrow + subheading + cta_text + cta_url. image_id from media library for OG-quality hero image. secondary_label + secondary_url optional."},
				{BlockType: "feature_grid", Required: false, Notes: "Sort 1+. heading + subheading + items[]. Each item: title (h3), body (p), icon (Lucide name). 3-6 items renders cleanly. Counts as list >= 3 items for AI-Friendly Formatting eval check."},
				{BlockType: "text", Required: false, Notes: "Long-form prose. eyebrow + heading + multi-paragraph body (split paragraphs with \\n\\n; single \\n becomes <br>). Optional inline cta_text + cta_url. Targets the 300+ word eval check."},
				{BlockType: "feature_grid", Required: false, Notes: "Repeat for additional sections (services, pricing, FAQ, social proof). Each grid is its own card cluster."},
				{BlockType: "cta", Required: true, Notes: "Last block. heading + text + cta_text + cta_url + variant ('primary'|'secondary'). Renders as a tinted banner card. Don't end a page on a feature_grid; conversion + content-density both improve with a closing CTA."},
			},
		},
		HardRules: []HardRule{
			{Rule: "Set both meta_title (30-60 chars) AND meta_description (120-160 chars with action verbs like learn / try / get / book / läs / prova) on every page row via update_page.", WhyItMatters: "Eval flags Title Length 30-60, Meta Description 120-160, Meta Description Has CTA per page; branding fallback alone fails these.", HowToApply: "update_page(slug=..., meta_title=..., meta_description=...) for every published page including legal + 404."},
			{Rule: "Use a hero block at sort_order=0 on every content page so the renderer emits an h1 from the headline field.", WhyItMatters: "Eval Has H1 + Single H1 are page-level checks. Without a hero block, no h1 renders, both checks fail.", HowToApply: "create_block(page_slug, block_type='hero', sort_order=0, data={headline, subheading, cta_text, cta_url, ...})"},
			{Rule: "Set per-page no_index=0 on real content pages. Use no_index=1 only for splash + admin pages.", WhyItMatters: "Eval Not Noindexed expects content pages to be indexable. Splash pages are fine to flag here as long as they're the only ones.", HowToApply: "update_page(slug=..., no_index=0). Default is 0; only flip to 1 for splash."},
			{Rule: "For multi-language sites, every page needs a counterpart in every other locale at the matching slug pattern.", WhyItMatters: "Eval Hreflang Tags fires when general.additional_langs is set; missing counterparts cause atomicsite to skip emitting hreflang for those pages, which the eval flags as missing.", HowToApply: "create_page for each locale. Slug convention: /<lang>/<slug>. Mirror /en/about ↔ /sv/about, /en/privacy ↔ /sv/privacy, etc."},
			{Rule: "PATCH /api/agent/branding to set meta_title, meta_description, og_image_id, favicon_id, lang as a fallback layer.", WhyItMatters: "Pages that don't override get the fallback. og_image_id + favicon_id drive the OG Image, OG Image Size 1200x630, Favicon, Apple Touch Icon, Organization Schema (logo) eval checks all at once.", HowToApply: "Upload a 1200x630 PNG to brand folder via /api/agent/media (no MCP wrapper for upload, use curl POST /api/agent/media/from-url or media/from-base64). Then PATCH branding with the returned media id."},
			{Rule: "Wire seo.same_as with newline-separated social URLs (LinkedIn, GitHub, Instagram, X, YouTube).", WhyItMatters: "Drives JSON-LD Organization sameAs which is the Organization Schema Completeness eval check.", HowToApply: "bulk_upsert_settings([{category:'seo', key:'same_as', value:'https://...\\nhttps://...'}])"},
			{Rule: "trigger_build is the publish verb. After it returns success, the live URL reflects the new content via auto-deploy.", WhyItMatters: "Build also runs eval. The eval response is the source of truth for what's failing; act on its checks_json directly rather than guessing.", HowToApply: "trigger_build -> poll get_build_status until status=='success' -> get_evaluation(build_id) -> walk failing checks -> fix -> repeat."},
			{Rule: "No plaintext emails in block text. Use mailto: links via raw block sparingly OR refer the reader to a contact page.", WhyItMatters: "Block-time guardrail rejects plaintext emails on create/update_block. Eval also flags any that slip through.", HowToApply: "Rephrase 'email privacy@example.com' to 'reach our privacy team via the address on the Privacy Policy page' or use 'privacy at example dot com' as obfuscation."},
			{Rule: "Update block sort_order via update_block (now persists; previously dropped). Don't delete-and-recreate to reorder.", WhyItMatters: "Was a contradiction in the agent surface; fixed in the same session that authored this playbook (commit 8a0f87d0). Block sort_order writes are now reliable.", HowToApply: "update_block(block_id, sort_order=N, data=...). Use unique sort_order values per page so render order is deterministic."},
		},
		BuildLoop: []string{
			"1. Read /api/agent/context to get pending_setup + settings_catalog + block_schemas + this playbook.",
			"2. Profile + branding + settings via update_profile / PATCH /api/agent/branding / bulk_upsert_settings.",
			"3. Page structure: create_page for every page in every locale.",
			"4. Per-page meta: update_page(slug, meta_title, meta_description, no_index, og_image_id) for every page.",
			"5. Block content: create_block per the page template (hero -> feature_grids/text -> cta).",
			"6. Global blocks: PUT /api/agent/global/{header,footer} once per site.",
			"7. trigger_build, wait for success, get_evaluation.",
			"8. For every failing check the agent can write: fix, re-build. For admin-only: collect into a final report.",
			"9. Stop when the agent-writable categories hit 95%+. Hand off the admin-only list with the human_admin_url for each item.",
		},
		AdminOnlyItems: []AdminOnlyItem{
			{Setting: "security category (HSTS preload, CSP extra directives, COOP/CORP/COEP, X-Frame-Options)", HumanAdminURL: "/sites/" + siteID + "/settings/security", EvalImpact: "Up to 6 security points (HSTS preload directive, CSP customisation)"},
			{Setting: "allowed-scripts (CSP allowlist for cal.com, YouTube, Stripe Checkout, GA, GTM)", HumanAdminURL: "/sites/" + siteID + "/settings/allowed-scripts", EvalImpact: "Required before iframe / third-party script blocks render. CSP Quality eval check at risk if you skip."},
			{Setting: "Cross-Origin-Embedder-Policy", HumanAdminURL: "/sites/" + siteID + "/settings/security", EvalImpact: "1 security point"},
		},
		VerificationGate: []string{
			"list_pages returns every page you intended (count matches your plan).",
			"list_blocks for the home page returns 5+ blocks in increasing sort_order.",
			"get_site_context.pending_setup is empty (or only admin-only items remain).",
			"Latest get_evaluation shows >= 95% on at least 3 of 5 categories (accessibility / seo / privacy on agent-writable parts; security depends on admin items).",
			"curl -I returns 200 on /, /<en>/, /<sv>/, /<en>/privacy, etc.",
			"Live HTML at the homepage contains <h1> from the hero block, <img> with /media/<og_image_id> reference, og:image meta tag, JSON-LD Organization schema with logo + sameAs.",
			"Response headers carry Strict-Transport-Security, Content-Security-Policy, X-Frame-Options, Referrer-Policy, Permissions-Policy, X-Content-Type-Options.",
		},
	}
}

// computeI18n surfaces the multi-language config for one site. Reads
// sites.lang, settings (general.additional_langs, seo.hreflang_strategy,
// seo.meta_title_template, seo.meta_description_template,
// seo.canonical_base), then derives the actual published locale roots
// from the page list so the agent knows which language paths are live.
func (b *ContextBuilder) computeI18n(ctx context.Context, siteID string, site store.Site, pages []PageInfo) I18nInfo {
	settings, _ := b.queries.ListSettingsBySite(ctx, siteID)
	sm := make(map[string]string, len(settings))
	for _, s := range settings {
		sm[s.Category+"."+s.Key] = s.Value
	}

	defaultLang := strings.ToLower(strings.TrimSpace(site.Lang))
	if defaultLang == "" {
		defaultLang = "en"
	}

	addl := []string{}
	seen := map[string]bool{defaultLang: true}
	for _, raw := range strings.Split(sm["general.additional_langs"], ",") {
		l := strings.ToLower(strings.TrimSpace(raw))
		if l == "" || seen[l] {
			continue
		}
		seen[l] = true
		addl = append(addl, l)
	}

	strategy := strings.ToLower(strings.TrimSpace(sm["seo.hreflang_strategy"]))
	switch strategy {
	case "path", "subdomain", "off":
	default:
		strategy = "path"
	}

	// Derive actual locale roots from the published-page slugs. Default
	// locale root is "" (root). Each /<lang>/ prefix that has at least
	// one published page becomes a root.
	rootSet := map[string]bool{"": true}
	knownLangs := map[string]bool{defaultLang: true}
	for _, l := range addl {
		knownLangs[l] = true
	}
	for _, p := range pages {
		s := strings.TrimPrefix(p.Slug, "/")
		if s == "" {
			continue
		}
		first := strings.ToLower(strings.SplitN(s, "/", 2)[0])
		if knownLangs[first] {
			rootSet["/"+first] = true
		}
	}
	roots := make([]string, 0, len(rootSet))
	for r := range rootSet {
		roots = append(roots, r)
	}
	sort.Strings(roots)

	return I18nInfo{
		DefaultLang:             defaultLang,
		AdditionalLangs:         addl,
		Strategy:                strategy,
		LocaleRoots:             roots,
		MetaTitleTemplate:       sm["seo.meta_title_template"],
		MetaDescriptionTemplate: sm["seo.meta_description_template"],
		CanonicalBase:           sm["seo.canonical_base"],
	}
}

// computeSecurityPosture builds the agent-facing security summary. Reads
// allowed_scripts to group trusted domains by kind, computes the resolved
// CSP via the same builder helper the build pipeline uses, and lists the
// auto-generated files atomicsite emits for every site.
func (b *ContextBuilder) computeSecurityPosture(ctx context.Context, siteID string) SecurityPostureInfo {
	rows, _ := b.queries.ListAllowedScriptsBySite(ctx, siteID)
	td := TrustedDomainsByKind{
		Script:  []string{},
		Frame:   []string{},
		Image:   []string{},
		Media:   []string{},
		Connect: []string{},
		All:     []string{},
	}
	for _, s := range rows {
		if s.IsActive != 1 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(s.Kind)) {
		case "frame":
			td.Frame = append(td.Frame, s.Domain)
		case "image":
			td.Image = append(td.Image, s.Domain)
		case "media":
			td.Media = append(td.Media, s.Domain)
		case "connect":
			td.Connect = append(td.Connect, s.Domain)
		case "all":
			td.All = append(td.All, s.Domain)
		default: // "" or "script"
			td.Script = append(td.Script, s.Domain)
		}
	}

	// Resolved CSP / HSTS / frame-ancestors. Same source-of-truth as
	// the build pipeline; written via internal/builder/security.go but
	// imported here would be a circular dep, so we re-read the same
	// settings + scripts and inline the few fields we need.
	settings, _ := b.queries.ListSettingsBySite(ctx, siteID)
	sm := make(map[string]string)
	for _, s := range settings {
		sm[s.Category+"."+s.Key] = s.Value
	}
	frameAncestors := sm["security.frame_ancestors"]
	if frameAncestors == "" || frameAncestors == "auto" {
		frameAncestors = "'none'"
	}
	// CSP + HSTS strings are computed by builder.BuildSecurityHeaders;
	// to keep this package free of a builder import we leave them empty
	// and let the agent fetch them via GET /api/agent/security/preview
	// when it needs the resolved policy verbatim. Most of the time the
	// trusted_domains map + frame_ancestors are enough.
	return SecurityPostureInfo{
		TrustedDomains: td,
		AllowedKinds:   []string{"script", "frame", "image", "media", "connect", "all"},
		FrameAncestors: frameAncestors,
		HumanAdminURL:  "/sites/{site_id}/settings/allowed-scripts",
		GeneratedFiles: []GeneratedFileInfo{
			{Path: "/sitemap-index.xml", Label: "XML Sitemap", Description: "Auto-built from every published page via @astrojs/sitemap."},
			{Path: "/robots.txt", Label: "robots.txt", Description: "Auto-built; AI training bots blocked by default. Override via seo.robots_txt setting."},
			{Path: "/.well-known/security.txt", Label: "security.txt", Description: "RFC 9116 contact, emitted when Profile.security_email is set."},
			{Path: "/llms.txt", Label: "llms.txt", Description: "AI search readiness (Perplexity, ChatGPT). Built from published pages."},
			{Path: "/humans.txt", Label: "humans.txt", Description: "Team + site metadata."},
			{Path: "/favicon.ico", Label: "Favicon", Description: "Linked in every page from the brand media folder."},
			{Path: "/apple-touch-icon.png", Label: "Apple touch icon", Description: "iOS home-screen icon."},
			{Path: "<head> JSON-LD Organization", Label: "Organization schema", Description: "Auto-emitted from Profile (business name + sameAs URLs)."},
			{Path: "<head> JSON-LD BreadcrumbList", Label: "BreadcrumbList schema", Description: "Per page (depth >= 2)."},
			{Path: "<head> JSON-LD Article", Label: "Article schema", Description: "Article-type pages: datePublished + dateModified + author."},
			{Path: "<head> OG + Twitter Card", Label: "Social meta", Description: "og:title/description/image/site_name/locale + twitter:card=summary_large_image."},
		},
	}
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
		Security: []EndpointInfo{
			{Method: "GET", Path: "/api/agent/security/preview", Use: "Returns the resolved security headers (CSP, HSTS, X-Frame-Options, Referrer-Policy, Permissions-Policy, COOP, CORP, COEP) the next build will emit. Use to verify a CSP includes the host you need before recommending a content change to the user."},
			{Method: "GET", Path: "/api/agent/settings/security", Use: "Read-only view of the security category settings (HSTS, CSP toggles, frame_ancestors, csp_extra_directives, COOP/CORP/COEP). Writes here stay admin-only because they widen attack surface."},
			{Method: "GET", Path: "/api/agent/settings/general", Use: "Read general-category settings (default_lang, additional_langs, domain_aliases). Writes via PATCH /api/agent/settings."},
			{Method: "GET", Path: "/api/agent/settings/seo", Use: "Read seo-category settings (meta_title_template, meta_description_template, canonical_base, hreflang_strategy, robots_txt override, llms_txt override, same_as, logo_url). Writes via PATCH /api/agent/settings."},
			{Method: "PATCH", Path: "/api/agent/settings", Use: "Bulk upsert. seo + general + analytics are agent-writable; security + allowed-scripts + nginx + danger reject silently and the response carries rejected_admin_only."},
		},
		TrustedDomains: []EndpointInfo{
			{Method: "GET", Path: "/api/agent/allowed-scripts", Use: "Read the trusted-domain rows that feed CSP. Each row has a kind (script | frame | image | media | connect | all). Use this to detect 'is cal.com already trusted?' before asking the user to add it. Mutations are admin-only at /api/sites/{id}/allowed-scripts (UI: /sites/{id}/settings/allowed-scripts)."},
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
		{
			BlockType: "split_hero",
			Use:       "Side-by-side hero with text on the left + image on the right (collapses to stacked on mobile). Better for SaaS marketing pages than the centred hero.",
			TextKeys:  heroTextKeys,
			OtherKeys: []BlockSchemaField{
				{Key: "cta_url", Label: "Primary CTA URL"},
				{Key: "secondary_url", Label: "Secondary CTA URL"},
				{Key: "image_id", Label: "Hero image media ID (right column)"},
				{Key: "image_alt", Label: "Hero image alt text"},
			},
		},
		{
			BlockType: "stat_grid",
			Use:       "Trust signals / social proof. Horizontal grid of large numbers + labels. Counts as list >= 3 items for the AI-Friendly Formatting eval check.",
			TextKeys:  []BlockSchemaField{{Key: "heading", Label: "Heading"}, {Key: "subheading", Label: "Subheading", Multiline: true}},
			OtherKeys: []BlockSchemaField{{Key: "items", Label: "Array of {value, label}"}},
		},
		{
			BlockType: "accordion_faq",
			Use:       "Q&A accordion using native <details>/<summary> (no JS needed). Auto-emits FAQPage JSON-LD so the FAQ Schema eval check passes.",
			TextKeys:  []BlockSchemaField{{Key: "heading", Label: "Heading"}, {Key: "subheading", Label: "Subheading", Multiline: true}},
			OtherKeys: []BlockSchemaField{{Key: "items", Label: "Array of {question, answer}"}},
		},
		{
			BlockType: "pricing",
			Use:       "3-up pricing tier cards with feature bullets + per-tier CTA. Set tiers[i].featured=true on the recommended plan to give it a primary border + shadow.",
			TextKeys:  []BlockSchemaField{{Key: "heading", Label: "Heading"}, {Key: "subheading", Label: "Subheading", Multiline: true}},
			OtherKeys: []BlockSchemaField{{Key: "tiers", Label: "Array of {name, price, price_period, description, features[], cta_text, cta_url, featured?}"}},
		},
		{
			BlockType: "logo_strip",
			Use:       "Row of customer / partner logos. Use under the hero on a real production marketing site to anchor trust before the first feature.",
			TextKeys:  []BlockSchemaField{{Key: "label", Label: "Label (e.g. 'Trusted by')"}},
			OtherKeys: []BlockSchemaField{{Key: "items", Label: "Array of {image_id, alt, href?}"}},
		},
		{
			BlockType: "code_block",
			Use:       "Monospace code presentation with optional language label. Use for technical-blog and developer-docs sites; not for executable code (atomicsite is static).",
			TextKeys:  []BlockSchemaField{{Key: "heading", Label: "Heading"}, {Key: "code", Label: "Code body", Multiline: true}},
			OtherKeys: []BlockSchemaField{{Key: "language", Label: "Language label (e.g. 'go', 'typescript')"}},
		},
		{
			BlockType: "form",
			Use:       "Basic HTML form. Browser POSTs to the action URL. Operator wires the receiving endpoint (a worker, Formspree, n8n webhook, etc.). For simple contact + lead-capture flows.",
			TextKeys:  []BlockSchemaField{{Key: "heading", Label: "Heading"}, {Key: "subheading", Label: "Subheading", Multiline: true}, {Key: "submit_label", Label: "Submit button label"}},
			OtherKeys: []BlockSchemaField{
				{Key: "action", Label: "Form POST URL"},
				{Key: "method", Label: "HTTP method ('post' default; 'get' for query-string forms)"},
				{Key: "fields", Label: "Array of {name, type ('text'|'email'|'tel'|'url'|'textarea'|'select'), label, placeholder?, required?, options?}"},
			},
		},
		{
			BlockType: "embed",
			Use:       "iframe wrapped in 16:9 aspect-ratio container. Use for cal.com bookings, YouTube videos, Stripe Checkout, Typeform. The src host MUST be in trusted_domains (kind=frame) — admin grants via /sites/{id}/settings/allowed-scripts.",
			TextKeys:  []BlockSchemaField{{Key: "heading", Label: "Heading"}, {Key: "title", Label: "iframe title (a11y)"}},
			OtherKeys: []BlockSchemaField{{Key: "src", Label: "iframe src URL"}, {Key: "aspect_ratio", Label: "Aspect ratio (default '16/9')"}},
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
		{
			Name:        "Trusted external domains",
			URL:         "/sites/{site_id}/settings/allowed-scripts",
			Description: "CSP allowlist. Each row picks a kind: script (Stripe.js / GA), frame (cal.com / YouTube / Stripe Checkout iframes), image (Cloudinary CDN), media (Vimeo), connect (fetch-only API host), or all. Writes are admin-only to keep agents from silently widening the attack surface; the agent should point the user here when an embed is needed.",
		},
		{
			Name:        "Security headers",
			URL:         "/sites/{site_id}/settings/security",
			Description: "Edit HSTS, CSP enable + extra directives, frame-ancestors, X-Frame-Options, Referrer-Policy, Permissions-Policy, COOP/CORP/COEP. Admin-only (writes change attack surface).",
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
			Action:   "Either flip on the bundled cookie banner (PATCH /api/agent/settings analytics.cookieproof_enabled=1; branding flows in automatically, copy/categories live in cookie_banner_* and cookie_cat_*) or paste a third-party snippet (analytics.cookie_banner_snippet) for Cookiebot, OneTrust, Termly, etc.",
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
