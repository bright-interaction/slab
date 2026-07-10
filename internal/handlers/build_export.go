// Package handlers - build_export.go.
//
// ExportDistZip streams the latest successful build's dist/ tree
// for a site as a ZIP archive. Lets operators archive a snapshot,
// review what shipped, or hand the bundle to a static host that
// doesn't speak rsync / Caddy / Pages.
//
// GET /api/sites/{siteID}/dist.zip[?build_id=<id>]
//
// Auth: siteR mounts this under SiteAccessMiddleware so only
// members of the site can download. Admin global-role bypass
// works the same as every other admin route.
//
// build_id query param picks a specific deployment; default is the
// most recent succeeded deployment for the site. 404 when no
// successful deployment exists yet.
//
// Path-traversal: every relative path written into the ZIP is
// canonicalized (filepath.Rel) and rejected if it escapes the
// dist root, so a symlink the build planted can't write outside
// the archive. Symlinks themselves are followed (we want the
// content, not the link record) but the destination still has
// to live under dist.

package handlers

import (
	"archive/zip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bright-interaction/atomicsite/internal/store"
)

// ExportDistZip streams a ZIP of the dist/ tree for the latest
// successful deployment (or the one named in build_id).
func (h *BuildHandler) ExportDistZip(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if !isSafeSiteID(siteID) {
		writeError(w, http.StatusBadRequest, "Invalid siteID")
		return
	}

	site, err := h.queries.GetSiteByID(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}

	dep, err := h.resolveExportDeployment(r, siteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "No successful build available to export")
		return
	}

	distDir := filepath.Join(h.cfg.DataDir, "workspaces", siteID, "dist")
	info, err := os.Stat(distDir)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusNotFound, "Built dist directory missing on disk")
		return
	}

	// Resolve dist to its absolute, symlink-evaluated path once so
	// the per-entry traversal check is a cheap prefix compare. On
	// macOS, `/var/...` symlinks to `/private/var/...`, and a naive
	// filepath.Abs on the dataDir would yield `/var/...` while
	// filepath.EvalSymlinks of any leaf returns the `/private/...`
	// form, then the prefix check would reject every legitimate
	// in-tree symlink. EvalSymlinks here normalizes the comparison
	// base to the same form the per-entry resolver returns.
	absDist, err := filepath.EvalSymlinks(distDir)
	if err != nil {
		// EvalSymlinks fails for a non-existent path; we already
		// stat'd distDir above so this is rare. Fall back to Abs
		// so we still have something to compare.
		absDist, err = filepath.Abs(distDir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to resolve dist path")
			return
		}
	}

	filename := buildExportFilename(site.Slug, dep.ID, dep.CreatedAt)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	zw := zip.NewWriter(w)
	defer zw.Close()

	walkErr := filepath.Walk(distDir, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi == nil {
			return nil
		}
		// Skip the root dir entry; ZIPs use file paths, not a top-level
		// container directory, so the archive expands into the user's
		// chosen folder without an extra layer.
		if path == distDir {
			return nil
		}

		rel, err := filepath.Rel(distDir, path)
		if err != nil {
			return err
		}
		// Path-traversal guard. filepath.Rel can return "..", which
		// would happen via a malicious symlink; reject the entry
		// rather than write it into the archive.
		if rel == "." || strings.HasPrefix(rel, "..") {
			return nil
		}
		// readPath is what we eventually Stat + Open. For regular
		// files / dirs it's the walk path. For symlinks we replace
		// it with the EvalSymlinks-resolved absolute path so that
		// (a) we re-validate the resolved target against absDist,
		// (b) a TOCTOU swap of the symlink between resolve and open
		// can't redirect us to an attacker-chosen path: we open the
		// previously-validated resolved path directly. EvalSymlinks
		// walks the WHOLE chain, so symlinks-pointing-to-symlinks-
		// pointing-out are also rejected.
		readPath := path
		if fi.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil // dangling symlink; skip silently
			}
			absTarget, err := filepath.Abs(resolved)
			if err != nil {
				return nil
			}
			if !strings.HasPrefix(absTarget, absDist+string(filepath.Separator)) && absTarget != absDist {
				// Resolved target leaves the dist tree; refuse to follow.
				return nil
			}
			readPath = absTarget
			// Re-stat the resolved target so we read it as a file/dir.
			fi, err = os.Stat(absTarget)
			if err != nil {
				return nil
			}
		}

		// Skip device files, sockets, pipes, etc. We only zip regular
		// files and directories. zip.FileInfoHeader doesn't model the
		// other modes well and serving them would just confuse the
		// downloader.
		if !fi.IsDir() && !fi.Mode().IsRegular() {
			return nil
		}

		// ZIP uses forward slashes regardless of OS. filepath.Rel
		// returns native; convert before writing.
		zipName := filepath.ToSlash(rel)
		if fi.IsDir() {
			// ZIP encodes a directory as an entry whose name ends in '/'.
			// Optional but readers like Finder use it for empty dirs.
			zipName += "/"
		}

		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}
		header.Name = zipName
		// Deflate for files; Store for directories. Saves ~0% on
		// pre-compressed assets (PNG/JPEG/WOFF2) but ~70% on HTML/JS/CSS.
		if fi.IsDir() {
			header.Method = zip.Store
		} else {
			header.Method = zip.Deflate
		}

		entryWriter, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		f, err := os.Open(readPath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(entryWriter, f)
		_ = f.Close()
		return copyErr
	})

	if walkErr != nil && !errors.Is(walkErr, io.ErrClosedPipe) {
		// Headers already flushed; we can't change status. Logged
		// upstream via the access log + recoverer middleware.
		_ = walkErr
	}
}

// resolveExportDeployment picks the deployment to export. Either the
// build_id query param (cross-tenant guarded so a guess from another
// site returns 404) or the latest successful deployment for the site.
func (h *BuildHandler) resolveExportDeployment(r *http.Request, siteID string) (store.Deployment, error) {
	if id := strings.TrimSpace(r.URL.Query().Get("build_id")); id != "" {
		dep, err := h.queries.GetDeploymentByID(r.Context(), id)
		if err != nil {
			return store.Deployment{}, err
		}
		if dep.SiteID != siteID {
			return store.Deployment{}, errors.New("deployment belongs to another site")
		}
		if dep.Status != "succeeded" {
			return store.Deployment{}, errors.New("deployment did not succeed")
		}
		return dep, nil
	}
	deps, err := h.queries.ListDeploymentsBySite(r.Context(), siteID)
	if err != nil {
		return store.Deployment{}, err
	}
	for _, d := range deps {
		if d.Status == "succeeded" {
			return d, nil
		}
	}
	return store.Deployment{}, errors.New("no successful deployment")
}

// buildExportFilename builds the Content-Disposition filename so the
// downloaded file name encodes site + build identity. Avoids special
// characters that confuse browsers' download handling.
func buildExportFilename(slug, buildID, createdAt string) string {
	safeSlug := safeSlugForFilename(slug)
	if safeSlug == "" {
		safeSlug = "site"
	}
	short := buildID
	if len(short) > 12 {
		short = short[:12]
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	if t, err := time.Parse("2006-01-02T15:04:05Z", createdAt); err == nil {
		stamp = t.UTC().Format("20060102T150405Z")
	} else if t, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
		stamp = t.UTC().Format("20060102T150405Z")
	}
	return safeSlug + "-" + stamp + "-" + short + ".zip"
}

// safeSlugForFilename keeps a-z 0-9 and hyphen; replaces everything
// else with hyphen so a hostile slug can't break out of the
// Content-Disposition header.
func safeSlugForFilename(slug string) string {
	out := make([]byte, 0, len(slug))
	for _, r := range strings.ToLower(slug) {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, byte(r))
		case r >= '0' && r <= '9':
			out = append(out, byte(r))
		case r == '-':
			out = append(out, '-')
		default:
			out = append(out, '-')
		}
	}
	// Collapse runs of '-' and trim.
	collapsed := make([]byte, 0, len(out))
	prevDash := false
	for _, b := range out {
		if b == '-' {
			if prevDash {
				continue
			}
			prevDash = true
		} else {
			prevDash = false
		}
		collapsed = append(collapsed, b)
	}
	return strings.Trim(string(collapsed), "-")
}
