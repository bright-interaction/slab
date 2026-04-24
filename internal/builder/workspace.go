// Package builder generates complete Astro projects from database content.
package builder

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:skeleton
var skeletonFS embed.FS

const skeletonRoot = "skeleton"

// InitWorkspace copies the embedded Astro skeleton to the workspace directory.
// If the workspace already exists, it is removed and recreated.
func InitWorkspace(wsDir string) error {
	// Clean slate
	if err := os.RemoveAll(wsDir); err != nil {
		return err
	}
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		return err
	}

	return fs.WalkDir(skeletonFS, skeletonRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path from skeleton root
		relPath, err := filepath.Rel(skeletonRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		target := filepath.Join(wsDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := skeletonFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// EnsureDir creates a directory and all parents if they don't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// WriteFile writes content to a file, creating parent directories.
func WriteFile(path string, content string) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
