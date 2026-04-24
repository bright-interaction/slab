package builder

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// CopyMedia hardlinks (or copies) the site's processed media into the Astro
// workspace's public/media/ so that Astro bundles them into dist/. URLs stored
// in blocks as /media/{mediaID}/... resolve both during dev-serve and in the
// built static site.
//
// Disk layout transformation:
//   src: {dataDir}/media/{siteID}/{mediaID}/{variant}.{ext}
//   dst: {wsDir}/public/media/{mediaID}/{variant}.{ext}
//
// The siteID segment is dropped because the workspace is always single-site.
func CopyMedia(ctx context.Context, queries *store.Queries, siteID, dataDir, wsDir string) error {
	srcDir := filepath.Join(dataDir, "media", siteID)
	dstDir := filepath.Join(wsDir, "public", "media")

	// If the site has no media, that's fine.
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return nil
	}

	// Ensure dst exists (parent of each mediaID dir)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dstDir, err)
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return atomicLinkOrCopy(path, target)
	})
}

// atomicLinkOrCopy writes src to dst via a temp name + rename, avoiding the
// TOCTOU window between Remove and Link. Tries hardlink first (same filesystem),
// falls back to a byte copy across filesystems.
func atomicLinkOrCopy(src, dst string) error {
	tmp := dst + ".tmp-" + randomSuffix()
	if err := os.Link(src, tmp); err != nil {
		// Hardlink failed (likely cross-filesystem). Fall back to copy.
		if err := copyFile(src, tmp); err != nil {
			os.Remove(tmp)
			return err
		}
	}
	// rename is atomic on POSIX; overwrites target if it exists.
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func randomSuffix() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
