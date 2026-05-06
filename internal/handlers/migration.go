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
	"regexp"
	"strings"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/migration"
	"github.com/bright-interaction/slab/internal/store"
)

// MigrationHandler exposes the migration lifecycle. Crawl invocation and
// porter wiring happen here; the migration package itself stays HTTP-free.
type MigrationHandler struct {
	cfg        *config.Config
	queries    *store.Queries
	media      migration.MediaUploader
	verifyOpts migration.VerifyOptions
}

// NewMigrationHandler returns a handler. uploader is the production media
// uploader (or a fake in tests); the porter is constructed per-Apply call
// so unit tests can swap it via dependency injection at the package level.
//
// VerifyOptions defaults to zero, which means the production SSRF guard
// blocks loopback during VerifyLive. Tests use SetVerifyOptionsForTest to
// flip AllowPrivate so httptest-based integration tests can exercise the
// real crawl path.
func NewMigrationHandler(cfg *config.Config, queries *store.Queries, uploader migration.MediaUploader) *MigrationHandler {
	return &MigrationHandler{cfg: cfg, queries: queries, media: uploader}
}

// SetVerifyOptionsForTest is the test seam for VerifyLive. Production
// callers never invoke it; the boot wiring in server.go leaves the zero
// value (AllowPrivate=false) so loopback / RFC1918 / link-local addresses
// stay blocked. Setting AllowPrivate=true is only safe in unit tests
// against httptest.Server.
func (h *MigrationHandler) SetVerifyOptionsForTest(opts migration.VerifyOptions) {
	h.verifyOpts = opts
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

// ----- Layer 5 / Sprint 2: pre-launch verify + post-launch crawl -----

// verifyCoverageRequest is the input for the dry-run pre-launch check.
// The operator pastes URLs from a Search Console "Top URLs" export (or
// a sitemap export from the source site) and we report which buckets
// each URL will fall into post-launch. No DB writes.
type verifyCoverageRequest struct {
	URLs []string `json:"urls"`
}

// VerifyCoverage runs the same logic as RedirectHandler.Verify but scoped
// to a specific migration's manifest so the agent can reason about
// "after this migration applies, what survives?" without flipping DNS.
// Same response shape as the existing redirects/verify endpoint so the UI
// can reuse the bucket renderer.
func (h *MigrationHandler) VerifyCoverage(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "migrationID")
	row, err := h.queries.GetMigrationByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Migration not found")
		return
	}
	var req verifyCoverageRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if len(req.URLs) == 0 {
		writeError(w, http.StatusBadRequest, "urls array is required")
		return
	}
	if len(req.URLs) > 100_000 {
		writeError(w, http.StatusRequestEntityTooLarge, "verify-coverage capped at 100,000 URLs")
		return
	}

	// Reuse the live page + redirect set so the diff reflects current
	// atomicsite state plus what THIS migration would add. We don't
	// actually apply the manifest; we just merge planned slugs into
	// the existing-page map for the diff.
	pages, _ := h.queries.ListPagesBySite(r.Context(), siteID)
	pageSlugs := make(map[string]bool, len(pages))
	for _, p := range pages {
		pageSlugs[normalizeVerifyPath("/"+strings.TrimPrefix(p.Slug, "/"))] = true
	}
	var manifest migration.MigrationManifest
	_ = json.Unmarshal([]byte(row.ManifestJson), &manifest)
	plan := migration.PlanURLs(&manifest, pageSlugsToTrimmed(pageSlugs), migration.DefaultPlanOptions())
	for _, pp := range plan.Pages {
		if !pp.Skipped {
			pageSlugs[normalizeVerifyPath(pp.NewSlug)] = true
		}
	}
	redirects, _ := h.queries.ListRedirectsBySite(r.Context(), siteID)
	exact := make(map[string]store.Redirect, len(redirects))
	regexes := make([]compiledRegexRedirect, 0, 8)
	for _, rr := range redirects {
		if strings.HasPrefix(rr.FromPath, "~") {
			pat := strings.TrimSpace(strings.TrimPrefix(rr.FromPath, "~"))
			if re, err := regexp.Compile(pat); err == nil {
				regexes = append(regexes, compiledRegexRedirect{re: re, target: rr})
			}
			continue
		}
		exact[normalizeVerifyPath(rr.FromPath)] = rr
	}
	// Merge planned redirects into the diff so the operator sees the
	// shape AFTER apply, not just what's live today.
	for _, rp := range plan.Redirects {
		if strings.HasPrefix(rp.FromPath, "~") {
			pat := strings.TrimSpace(strings.TrimPrefix(rp.FromPath, "~"))
			if re, err := regexp.Compile(pat); err == nil {
				regexes = append(regexes, compiledRegexRedirect{
					re:     re,
					target: store.Redirect{FromPath: rp.FromPath, ToPath: rp.ToPath, StatusCode: int64(rp.StatusCode)},
				})
			}
			continue
		}
		exact[normalizeVerifyPath(rp.FromPath)] = store.Redirect{
			FromPath: rp.FromPath, ToPath: rp.ToPath, StatusCode: int64(rp.StatusCode),
		}
	}

	resp := VerifyResponse{
		Will200:   []string{},
		Will301:   []string{},
		Will410:   []string{},
		Will404:   []string{},
		Malformed: []string{},
	}
	for _, raw := range req.URLs {
		path, ok := extractURLPath(raw)
		if !ok {
			resp.Malformed = append(resp.Malformed, raw)
			continue
		}
		key := normalizeVerifyPath(path)
		if pageSlugs[key] {
			resp.Will200 = append(resp.Will200, raw)
			continue
		}
		if rr, ok := exact[key]; ok {
			if rr.StatusCode == 410 {
				resp.Will410 = append(resp.Will410, raw)
			} else {
				resp.Will301 = append(resp.Will301, raw)
			}
			continue
		}
		matched := false
		for _, rx := range regexes {
			if rx.re.MatchString(path) {
				if rx.target.StatusCode == 410 {
					resp.Will410 = append(resp.Will410, raw)
				} else {
					resp.Will301 = append(resp.Will301, raw)
				}
				matched = true
				break
			}
		}
		if !matched {
			resp.Will404 = append(resp.Will404, raw)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// pageSlugsToTrimmed flips a "/about" map to an "about" map for PlanURLs.
// Tiny adapter; PlanURLs takes existingSlugs without leading slash to
// match the pages.slug column convention.
func pageSlugsToTrimmed(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k := range in {
		out[strings.TrimPrefix(k, "/")] = true
	}
	return out
}

// verifyLiveRequest is the input for the post-DNS-flip crawl.
type verifyLiveRequest struct {
	// URLs is the source-site URL list (typically the same set used for
	// VerifyCoverage). Each gets a fresh GET with redirect-following.
	URLs []string `json:"urls"`
	// DeployedDomain is what the operator's DNS now points at, e.g.
	// "www.acme.com". Used to swap the host into each source URL before
	// fetching so we exercise the new edge.
	DeployedDomain string `json:"deployed_domain"`
}

// VerifyLive runs migration.VerifyLive against the deployed site and
// stores per-URL results. Returns summary counts; the full per-row
// detail is available via ListVerifications. Capped at 1,000 URLs per
// run because each URL can take seconds (network + politeness delay)
// and the request runs synchronously in this layer; async + queue is
// Sprint 4.
func (h *MigrationHandler) VerifyLive(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "migrationID")
	row, err := h.queries.GetMigrationByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Migration not found")
		return
	}
	var req verifyLiveRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if strings.TrimSpace(req.DeployedDomain) == "" {
		writeError(w, http.StatusBadRequest, "deployed_domain is required")
		return
	}
	if len(req.URLs) == 0 {
		writeError(w, http.StatusBadRequest, "urls is required")
		return
	}
	if len(req.URLs) > 1000 {
		writeError(w, http.StatusRequestEntityTooLarge, "verify-live capped at 1,000 URLs per run")
		return
	}

	results, err := migration.VerifyLive(r.Context(), req.URLs, req.DeployedDomain, h.verifyOpts)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	okCount := 0
	failCount := 0
	for _, vr := range results {
		ok := int64(0)
		if vr.OK {
			ok = 1
			okCount++
		} else {
			failCount++
		}
		_ = h.queries.CreateMigrationVerification(r.Context(), store.CreateMigrationVerificationParams{
			ID:         newID(),
			SiteID:     siteID,
			MigrationID: id,
			SourceUrl:  vr.SourceURL,
			StatusCode: int64(vr.StatusCode),
			FinalUrl:   vr.FinalURL,
			Hops:       int64(vr.Hops),
			Ok:         ok,
			Error:      vr.Error,
		})
	}
	AuditLog(r.Context(), h.queries, r, siteID, AuditActionUpdate, "migration_verify_live", id, map[string]any{
		"deployed_domain": req.DeployedDomain,
		"checked":         len(results),
		"ok":              okCount,
		"failed":          failCount,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"checked": len(results),
		"ok":      okCount,
		"failed":  failCount,
		"results": results,
	})
}

// ListVerifications returns the stored per-URL verification history for a
// migration, newest first. Used by the migration UI to render the
// post-launch verification panel.
func (h *MigrationHandler) ListVerifications(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "migrationID")
	row, err := h.queries.GetMigrationByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Migration not found")
		return
	}
	rows, err := h.queries.ListMigrationVerifications(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "List verifications")
		return
	}
	counts, _ := h.queries.CountMigrationVerificationsByOK(r.Context(), id)
	out := make([]map[string]any, 0, len(rows))
	for _, v := range rows {
		out = append(out, map[string]any{
			"id":          v.ID,
			"source_url":  v.SourceUrl,
			"status_code": v.StatusCode,
			"final_url":   v.FinalUrl,
			"hops":        v.Hops,
			"ok":          v.Ok == 1,
			"error":       v.Error,
			"checked_at":  v.CheckedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  out,
		"ok":     counts.OkCount,
		"failed": counts.FailCount,
	})
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
