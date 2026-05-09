package builder

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// BuildResult holds the complete outcome of a site build.
type BuildResult struct {
	DistDir    string
	BuildLog   string
	PagesBuilt int
	DurationMs int64
	Success    bool
	Error      string
}

// Build orchestrates the full build pipeline for a site:
// 1. Init workspace
// 2. Generate CSS
// 3. Generate components
// 4. Generate layouts
// 5. Generate pages
// 6. Generate config (astro.config, robots.txt, security.txt, redirects)
// 7. Compile (bun install + bun run build)
func Build(ctx context.Context, queries *store.Queries, siteID string, dataDir string) *BuildResult {
	wsDir := filepath.Join(dataDir, "workspaces", siteID)

	slog.Info("build: starting", "site_id", siteID, "workspace", wsDir)

	// 1. Init workspace
	if err := InitWorkspace(wsDir); err != nil {
		return fail("init workspace: " + err.Error())
	}

	// 2. CSS
	if err := RenderCSS(ctx, queries, siteID, wsDir); err != nil {
		return fail("render css: " + err.Error())
	}

	// 3. Components
	if err := RenderComponents(ctx, queries, siteID, wsDir); err != nil {
		return fail("render components: " + err.Error())
	}

	// 4. Layouts
	if err := RenderLayouts(ctx, queries, siteID, wsDir); err != nil {
		return fail("render layouts: " + err.Error())
	}

	// 5. Pages
	pageCount, err := RenderPages(ctx, queries, siteID, wsDir)
	if err != nil {
		return fail("render pages: " + err.Error())
	}
	if pageCount == 0 {
		return fail("no published pages found. Publish at least one page before building.")
	}

	// 5a. Custom Collections (Sprint 4, 2026-05-04). Emits index +
	// per-item .astro pages for every Collection with
	// settings.render_as_pages = true. Inherits Base.astro layout,
	// hreflang, lang, A+ headers from the standard renderer chain.
	collectionPageCount, err := RenderCollectionPages(ctx, queries, siteID, wsDir)
	if err != nil {
		return fail("render collections: " + err.Error())
	}
	pageCount += collectionPageCount

	// 5b. Per-collection Atom feeds. One feed.xml per opted-in
	// collection at public/{collection.slug}/feed.xml. Auto-emits
	// when schema_org_type is article-like (BlogPosting / Article /
	// NewsArticle); explicit settings.feed_enabled overrides.
	feedCount, err := RenderCollectionFeeds(ctx, queries, siteID, wsDir)
	if err != nil {
		return fail("render collection feeds: " + err.Error())
	}
	if feedCount > 0 {
		slog.Info("build: collection feeds emitted", "site_id", siteID, "feeds", feedCount)
	}

	// 6. Config
	if err := RenderConfig(ctx, queries, siteID, wsDir); err != nil {
		return fail("render config: " + err.Error())
	}

	// 6a. Security & settings (_headers, humans.txt, llms.txt, nginx.conf)
	if err := RenderSecurityHeaders(ctx, queries, siteID, wsDir); err != nil {
		return fail("render security headers: " + err.Error())
	}
	if err := RenderHumansTxt(ctx, queries, siteID, wsDir); err != nil {
		return fail("render humans.txt: " + err.Error())
	}
	if err := RenderLLMsTxt(ctx, queries, siteID, wsDir); err != nil {
		return fail("render llms.txt: " + err.Error())
	}
	if err := RenderNginxConfig(ctx, queries, siteID, wsDir); err != nil {
		return fail("render nginx.conf: " + err.Error())
	}
	// Phase 30.3: Caddy fragment for multi-tenant edge deployments.
	// Lives alongside _headers and nginx.conf so the operator can
	// pick whichever serving layer fits their deployment shape.
	if err := RenderCaddyFragment(ctx, queries, siteID, wsDir); err != nil {
		return fail("render Caddyfile fragment: " + err.Error())
	}

	// 6b. Media
	if err := CopyMedia(ctx, queries, siteID, dataDir, wsDir); err != nil {
		return fail("copy media: " + err.Error())
	}

	slog.Info("build: generated files", "pages", pageCount, "workspace", wsDir)

	// 7. Compile
	// Pass the build context through so cancellation (admin click,
	// graceful shutdown, BuildTimeout) reaches the bun child processes.
	result := Compile(ctx, wsDir)

	return &BuildResult{
		DistDir:    result.DistDir,
		BuildLog:   result.BuildLog,
		PagesBuilt: pageCount,
		DurationMs: result.DurationMs,
		Success:    result.Success,
		Error:      result.Error,
	}
}

func fail(msg string) *BuildResult {
	return &BuildResult{
		Error:   fmt.Sprintf("build failed: %s", msg),
		Success: false,
	}
}
