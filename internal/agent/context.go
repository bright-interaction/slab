// Package agent provides the AI agent API logic: context building, guardrails, and operations.
package agent

import (
	"context"
	"encoding/json"

	"github.com/brightinteraction/atomicsite/internal/store"
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
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Slug      string      `json:"slug"`
	Status    string      `json:"status"`
	Layout    string      `json:"layout"`
	SortOrder int64       `json:"sort_order"`
	ShowInNav bool        `json:"show_in_nav"`
	NavLabel  string      `json:"nav_label"`
	Blocks    []BlockInfo `json:"blocks"`
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
			ID:        p.ID,
			Title:     p.Title,
			Slug:      p.Slug,
			Status:    p.Status,
			Layout:    p.Layout,
			SortOrder: p.SortOrder,
			ShowInNav: p.ShowInNav == 1,
			NavLabel:  p.NavLabel,
			Blocks:    blockInfos,
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
	}, nil
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
