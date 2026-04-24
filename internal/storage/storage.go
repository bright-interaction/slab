// Package storage provides a pluggable file storage interface.
// MVP: local disk. Extensible to S3-compatible backends via the Store interface.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Store is the interface for file storage backends.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader, contentType string) (url string, err error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	DeleteDir(ctx context.Context, prefix string) error
	URL(ctx context.Context, key string, expires time.Duration) (string, error)
	LocalPath(key string) (string, error)
	Backend() string
}

// LocalStore stores files on the local filesystem.
type LocalStore struct {
	baseDir string
}

// NewLocalStore creates a local filesystem store rooted at baseDir.
func NewLocalStore(baseDir string) (*LocalStore, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("create storage dir %s: %w", baseDir, err)
	}
	return &LocalStore{baseDir: baseDir}, nil
}

// safePath resolves key under baseDir, blocking path traversal.
func (s *LocalStore) safePath(key string) (string, error) {
	// Reject keys containing ".." segments outright
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("path traversal blocked")
	}
	fullPath := filepath.Join(s.baseDir, key)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	absBase, _ := filepath.Abs(s.baseDir)
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("path traversal blocked")
	}
	return absPath, nil
}

func (s *LocalStore) Put(_ context.Context, key string, r io.Reader, _ string) (string, error) {
	absPath, err := s.safePath(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o750); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}
	f, err := os.Create(absPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		os.Remove(absPath)
		return "", fmt.Errorf("write file: %w", err)
	}
	return "/media/" + key, nil
}

func (s *LocalStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	absPath, err := s.safePath(key)
	if err != nil {
		return nil, err
	}
	return os.Open(absPath)
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	absPath, err := s.safePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *LocalStore) DeleteDir(_ context.Context, prefix string) error {
	absPath, err := s.safePath(prefix)
	if err != nil {
		return err
	}
	return os.RemoveAll(absPath)
}

func (s *LocalStore) URL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "/media/" + key, nil
}

// LocalPath returns the absolute filesystem path for a key. Used by the build
// pipeline to hardlink media into workspaces without going through Get/Put.
func (s *LocalStore) LocalPath(key string) (string, error) {
	return s.safePath(key)
}

func (s *LocalStore) Backend() string { return "local" }

// BaseDir returns the absolute root dir. Used for static file serving and
// walkdir operations.
func (s *LocalStore) BaseDir() string { return s.baseDir }
