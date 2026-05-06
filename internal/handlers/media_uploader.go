package handlers

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/bright-interaction/slab/internal/migration"
)

// productionMediaUploader is the real migration.MediaUploader used at boot.
// It hooks the migration porter into the same imaging + storage pipeline
// that drives /api/sites/{id}/media/upload-from-url, so a migrated image
// lands as a fully-processed atomicsite media row (variants, blurhash,
// content-addressed storage) instead of staying as a hot-link to the
// source.
//
// Sprint 1 of the post-Layer-3d plan (2026-05-06). Replaces the boot
// default `NoopMediaUploader` so migrations actually self-host their
// assets. Without this, the migration system's "your new site is
// independent of the old domain" promise is unfulfilled.
//
// HTTP fetch goes through migration.SafeFetch (the SSRF-guarded fetcher
// already shared by every importer in internal/migration/), not through
// handlers.fetchWithSSRFGuard, because the migration variant already
// supports test-friendly AllowPrivate so end-to-end tests can exercise
// the full upload pipeline against an httptest.Server bound to loopback.
// Production behaviour is identical: AllowPrivate stays false at the
// boot wiring in server.go.
type productionMediaUploader struct {
	h         *MediaHandler
	fetchOpts migration.FetchOptions
}

// NewProductionMediaUploader wires the real uploader. Pass the MediaHandler
// the rest of the server uses; reusing processAndSave keeps the migration
// path on the exact same security guarantees as the agent
// /upload-from-url endpoint (private-IP block via SafeFetch, MIME sniff,
// dangerous-extension block, quota enforcement).
func NewProductionMediaUploader(h *MediaHandler) migration.MediaUploader {
	return &productionMediaUploader{
		h: h,
		fetchOpts: migration.FetchOptions{
			MaxBytes: defaultMaxUploadBytes(h),
		},
	}
}

// NewProductionMediaUploaderWithFetchOpts is the test seam. Production code
// always uses NewProductionMediaUploader; tests pass FetchOptions{AllowPrivate:
// true} so httptest.Server bound to 127.0.0.1 actually gets fetched. Never
// used outside _test.go files.
func NewProductionMediaUploaderWithFetchOpts(h *MediaHandler, opts migration.FetchOptions) migration.MediaUploader {
	if opts.MaxBytes == 0 {
		opts.MaxBytes = defaultMaxUploadBytes(h)
	}
	return &productionMediaUploader{h: h, fetchOpts: opts}
}

func defaultMaxUploadBytes(h *MediaHandler) int64 {
	if h == nil || h.cfg == nil || h.cfg.MaxUploadSize <= 0 {
		return 25 * 1024 * 1024 // safe ceiling matching the typical handler default
	}
	return h.cfg.MaxUploadSize
}

// UploadFromURL fetches sourceURL through the SSRF guard, runs it through
// the imaging pipeline (variants + blurhash + content-addressed storage),
// inserts a media row, and returns the public URL the rendered atomicsite
// will serve. The porter uses (mediaID, publicURL) to rewrite <img src>
// references in imported HTML blobs.
//
// caption is currently dropped because the media schema has no caption
// column; alt + filename carry the editorial context. If the schema gains
// a caption field later, wire it through here.
func (p *productionMediaUploader) UploadFromURL(ctx context.Context, siteID, sourceURL, alt, _ string) (string, string, error) {
	if p.h == nil {
		return "", "", fmt.Errorf("media uploader: handler not configured")
	}
	resp, err := migration.SafeFetch(ctx, sourceURL, p.fetchOpts)
	if err != nil {
		return "", "", fmt.Errorf("fetch %s: %w", sourceURL, err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return "", "", fmt.Errorf("fetch %s: status %d", sourceURL, resp.Status)
	}
	if len(resp.Body) == 0 {
		return "", "", fmt.Errorf("fetch %s: empty body", sourceURL)
	}

	filename := filenameFromURL(sourceURL)
	media, status, msg := p.h.processAndSave(ctx, siteID, filename, alt, "", resp.Body)
	if status >= 400 {
		return "", "", fmt.Errorf("process %s: %s (status %d)", sourceURL, msg, status)
	}

	// Public URL the deployed atomicsite serves. The build pipeline
	// (internal/builder/media.go) rewrites these to the per-tenant
	// /media/{mediaID}/{variant}.{ext} convention; we use original.{ext}
	// because that's the canonical full-resolution variant the imaging
	// pipeline always produces. Variants for responsive sizes are derived
	// from the same mediaID so a future block-level enhancement can pick
	// a smaller variant; the porter's HTML rewrite stays correct either
	// way because it only string-substitutes the source URL.
	ext := strings.TrimPrefix(filepath.Ext(media.OriginalPath), ".")
	if ext == "" {
		ext = "jpg"
	}
	publicURL := fmt.Sprintf("/media/%s/original.%s", media.ID, ext)
	return media.ID, publicURL, nil
}

// filenameFromURL pulls a sane filename from the source URL's path. WP
// usually serves /wp-content/uploads/2024/03/foo.jpg; the basename is
// what we want. Requires an absolute URL with a scheme (http/https) so
// stray strings like "not a url" or filesystem paths can't slip through
// and pollute the filename column. Anything pathological collapses to
// "imported-image" so the imaging pipeline still has a non-empty name
// to sanitise.
func filenameFromURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" || u.Path == "" {
		return "imported-image"
	}
	base := filepath.Base(u.Path)
	if base == "" || base == "/" || base == "." {
		return "imported-image"
	}
	return base
}
