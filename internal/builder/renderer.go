package builder

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/bright-interaction/slab/internal/store"
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

	// 6. Config
	if err := RenderConfig(ctx, queries, siteID, wsDir); err != nil {
		return fail("render config: " + err.Error())
	}

	slog.Info("build: generated files", "pages", pageCount, "workspace", wsDir)

	// 7. Compile
	result := Compile(wsDir)

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
