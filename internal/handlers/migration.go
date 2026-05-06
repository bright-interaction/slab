// Package handlers - migration.go (Layer 3d, 2026-05-06).
//
// Routes:
//
//	POST   /api/sites/{siteID}/migrations           start a crawl, store manifest
//	GET    /api/sites/{siteID}/migrations           list migrations for the site
//	GET    /api/sites/{siteID}/migrations/{id}      manifest detail
//	POST   /api/sites/{siteID}/migrations/{id}/plan generate URL plan, return for review
//	POST   /api/sites/{siteID}/migrations/{id}/apply commit pages/items/redirects/media
//	DELETE /api/sites/{siteID}/migrations/{id}      drop a stored manifest
//
// The handler keeps the manifest + plan in the migrations row's manifest_json
// column. Apply reads the same row, runs the porter, then sets status to
// "applied" with the porter's audit-friendly result counts.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/migration"
	"github.com/bright-interaction/slab/internal/store"
)

// MigrationHandler exposes the migration lifecycle. Crawl invocation and
// porter wiring happen here; the migration package itself stays HTTP-free.
type MigrationHandler struct {
	cfg     *config.Config
	queries *store.Queries
	media   migration.MediaUploader
}

// NewMigrationHandler returns a handler. uploader is the production media
// uploader (or a fake in tests); the porter is constructed per-Apply call
// so unit tests can swap it via dependency injection at the package level.
func NewMigrationHandler(cfg *config.Config, queries *store.Queries, uploader migration.MediaUploader) *MigrationHandler {
	return &MigrationHandler{cfg: cfg, queries: queries, media: uploader}
}

// migrationStartRequest is the input for POST /migrations.
type migrationStartRequest struct {
	SourceType string `json:"source_type"` // sitemap | wordpress | webflow | ghost
	SourceURL  string `json:"source_url"`  // sitemap URL, WP root, Webflow site (legacy), Ghost root
	// Auth shape varies per importer. Only the field for the chosen
	// source_type is read.
	WordPressAuthHeader string `json:"wordpress_auth_header,omitempty"`
	WebflowSiteID       string `json:"webflow_site_id,omitempty"`
	WebflowAuthToken    string `json:"webflow_auth_token,omitempty"`
	GhostContentKey     string `json:"ghost_content_key,omitempty"`
}

// Start kicks off a crawl synchronously. For small sites this returns the
// stored manifest in seconds; for large sites the operator should expect
// to wait. Async/queued crawls land in a follow-up.
func (h *MigrationHandler) Start(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	var req migrationStartRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.SourceURL == "" {
		writeError(w, http.StatusBadRequest, "source_url is required")
		return
	}
	manifest, err := h.runImporter(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Importer failed: "+err.Error())
		return
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Encode manifest")
		return
	}
	id := newID()
	if err := h.queries.CreateMigration(r.Context(), store.CreateMigrationParams{
		ID: id, SiteID: siteID, SourceUrl: req.SourceURL, SourceType: req.SourceType,
		ManifestJson: string(manifestJSON), Status: "ready", Error: "",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Persist migration")
		return
	}
	AuditLog(r.Context(), h.queries, r, siteID, AuditActionCreate, "migration", id, map[string]any{
		"source_type": req.SourceType, "source_url": req.SourceURL,
		"pages": manifest.Stats.PagesFound, "media": manifest.Stats.MediaFound,
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       id,
		"status":   "ready",
		"manifest": manifest,
	})
}

// runImporter dispatches to the right migration package client based on
// source_type. Each importer is independent and inherits the SSRF guard
// from internal/migration/safefetch.go.
func (h *MigrationHandler) runImporter(ctx context.Context, req migrationStartRequest) (*migration.MigrationManifest, error) {
	switch strings.ToLower(req.SourceType) {
	case "sitemap", "":
		return migration.CrawlSitemap(ctx, req.SourceURL, migration.CrawlOptions{})
	case "wordpress":
		c := migration.NewWordPressClient(req.SourceURL, migration.WordPressOptions{
			AuthHeader: req.WordPressAuthHeader,
		})
		return c.Crawl(ctx, migration.WordPressOptions{})
	case "webflow":
		c, err := migration.NewWebflowClient(migration.WebflowOptions{
			SiteID:    req.WebflowSiteID,
			AuthToken: req.WebflowAuthToken,
		})
		if err != nil {
			return nil, err
		}
		return c.Crawl(ctx, migration.WebflowOptions{})
	case "ghost":
		c, err := migration.NewGhostClient(migration.GhostOptions{
			BaseURL:    req.SourceURL,
			ContentKey: req.GhostContentKey,
		})
		if err != nil {
			return nil, err
		}
		return c.Crawl(ctx, migration.GhostOptions{})
	default:
		return nil, errors.New("unknown source_type; want sitemap, wordpress, webflow, or ghost")
	}
}

// List returns every migration row for the site, newest first.
func (h *MigrationHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	rows, err := h.queries.ListMigrationsBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "List migrations")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, m := range rows {
		out = append(out, migrationToSummary(m))
	}
	writeJSON(w, http.StatusOK, out)
}

// Get returns the full manifest body (potentially big - 10s of MB on large
// crawls). Guarded by site-id ownership check.
func (h *MigrationHandler) Get(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "migrationID")
	row, err := h.queries.GetMigrationByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Migration not found")
		return
	}
	var manifest migration.MigrationManifest
	_ = json.Unmarshal([]byte(row.ManifestJson), &manifest)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          row.ID,
		"site_id":     row.SiteID,
		"source_url":  row.SourceUrl,
		"source_type": row.SourceType,
		"status":      row.Status,
		"error":       row.Error,
		"created_at":  row.CreatedAt,
		"updated_at":  row.UpdatedAt,
		"manifest":    manifest,
	})
}

// migrationPlanRequest carries the operator's overrides to the URL planner.
// Empty body means "use defaults".
type migrationPlanRequest struct {
	SkipPaths            []string `json:"skip_paths,omitempty"`
	CollapseDateArchives *bool    `json:"collapse_date_archives,omitempty"`
	PreserveLocalePrefix *bool    `json:"preserve_locale_prefix,omitempty"`
}

// Plan re-reads the stored manifest, runs PlanURLs against the live page
// set (so collisions reflect the current atomicsite state), and returns
// the URLPlan + conflict count for the agent / operator to review.
func (h *MigrationHandler) Plan(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "migrationID")
	row, err := h.queries.GetMigrationByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Migration not found")
		return
	}
	var req migrationPlanRequest
	_ = parseJSON(r, &req) // empty body OK
	var manifest migration.MigrationManifest
	if err := json.Unmarshal([]byte(row.ManifestJson), &manifest); err != nil {
		writeError(w, http.StatusInternalServerError, "Decode manifest")
		return
	}

	pages, _ := h.queries.ListPagesBySite(r.Context(), siteID)
	existing := make(map[string]bool, len(pages))
	for _, p := range pages {
		existing[strings.TrimPrefix(p.Slug, "/")] = true
	}

	opts := migration.DefaultPlanOptions()
	if req.CollapseDateArchives != nil {
		opts.CollapseDateArchives = *req.CollapseDateArchives
	}
	if req.PreserveLocalePrefix != nil {
		opts.PreserveLocalePrefix = *req.PreserveLocalePrefix
	}
	for _, p := range req.SkipPaths {
		opts.SkipPaths[p] = true
	}

	plan := migration.PlanURLs(&manifest, existing, opts)
	writeJSON(w, http.StatusOK, plan)
}

// migrationApplyRequest mirrors migrationPlanRequest plus a confirmation
// flag the operator must flip when conflicts are non-zero (currently the
// porter refuses to apply a plan with conflicts at all; this flag is
// reserved for a future "force-apply" override).
type migrationApplyRequest struct {
	migrationPlanRequest
	ConfirmConflicts bool `json:"confirm_conflicts,omitempty"`
}

// Apply runs the porter against the current plan. On success the migration
// row's status flips to "applied"; on failure it flips to "failed" with
// the error in the row's error column.
func (h *MigrationHandler) Apply(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "migrationID")
	row, err := h.queries.GetMigrationByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Migration not found")
		return
	}
	if row.Status == "applied" {
		writeError(w, http.StatusConflict, "Migration already applied")
		return
	}

	var req migrationApplyRequest
	_ = parseJSON(r, &req)

	var manifest migration.MigrationManifest
	if err := json.Unmarshal([]byte(row.ManifestJson), &manifest); err != nil {
		writeError(w, http.StatusInternalServerError, "Decode manifest")
		return
	}

	pages, _ := h.queries.ListPagesBySite(r.Context(), siteID)
	existing := make(map[string]bool, len(pages))
	for _, p := range pages {
		existing[strings.TrimPrefix(p.Slug, "/")] = true
	}
	opts := migration.DefaultPlanOptions()
	if req.CollapseDateArchives != nil {
		opts.CollapseDateArchives = *req.CollapseDateArchives
	}
	if req.PreserveLocalePrefix != nil {
		opts.PreserveLocalePrefix = *req.PreserveLocalePrefix
	}
	for _, p := range req.SkipPaths {
		opts.SkipPaths[p] = true
	}
	plan := migration.PlanURLs(&manifest, existing, opts)

	porter := migration.NewPorter(h.queries, h.media)
	result, err := porter.Apply(r.Context(), siteID, &manifest, plan)
	if err != nil {
		_ = h.queries.UpdateMigrationStatus(r.Context(), store.UpdateMigrationStatusParams{
			ID: id, Status: "failed", Error: err.Error(),
		})
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	_ = h.queries.UpdateMigrationStatus(r.Context(), store.UpdateMigrationStatusParams{
		ID: id, Status: "applied", Error: "",
	})
	AuditLog(r.Context(), h.queries, r, siteID, AuditActionUpdate, "migration_apply", id, map[string]any{
		"pages": result.PagesCreated, "items": result.ItemsCreated,
		"redirects": result.RedirectsCreated, "media_uploaded": result.MediaUploaded,
		"media_failed": result.MediaFailed,
	})
	writeJSON(w, http.StatusOK, result)
}

// Delete drops the manifest row. The pages/items/redirects already created
// from a previously-applied migration are NOT touched (the operator deletes
// those manually if they want a clean slate).
func (h *MigrationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "migrationID")
	row, err := h.queries.GetMigrationByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Migration not found")
		return
	}
	if err := h.queries.DeleteMigration(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Delete migration")
		return
	}
	AuditLog(r.Context(), h.queries, r, siteID, AuditActionDelete, "migration", id, map[string]any{
		"source_url": row.SourceUrl,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// migrationToSummary is the small shape used in List - drops the heavy
// manifest_json blob so the listing endpoint stays under a few KB.
func migrationToSummary(m store.Migration) map[string]any {
	stats := map[string]any{}
	var manifest migration.MigrationManifest
	if json.Unmarshal([]byte(m.ManifestJson), &manifest) == nil {
		stats = map[string]any{
			"pages":       manifest.Stats.PagesFound,
			"collections": manifest.Stats.CollectionsFound,
			"media":       manifest.Stats.MediaFound,
		}
	}
	return map[string]any{
		"id":          m.ID,
		"source_url":  m.SourceUrl,
		"source_type": m.SourceType,
		"status":      m.Status,
		"error":       m.Error,
		"created_at":  m.CreatedAt,
		"updated_at":  m.UpdatedAt,
		"stats":       stats,
	}
}

// noopMediaUploader is the default uploader when the server boots without
// real media handling wired. Returns an error every time so the porter
// records a clean MediaFailed warning. Production wires the real uploader
// in cmd/server/main.go.
type noopMediaUploader struct{}

// NoopMediaUploader returns a MediaUploader that fails every upload with
// a clear "not configured" message. Use during boot when the media wiring
// has not been done yet so the migration handler can still serve crawls
// (which don't need media re-upload to be valuable).
func NoopMediaUploader() migration.MediaUploader {
	return &noopMediaUploader{}
}

func (noopMediaUploader) UploadFromURL(ctx context.Context, siteID, sourceURL, alt, caption string) (string, string, error) {
	return "", "", fmt.Errorf("media uploader not configured; source URL preserved")
}
