// Package config loads configuration from environment variables.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port          int
	DataDir       string
	DBPath        string
	MediaDir      string
	JWTSecret     string
	BaseURL       string
	MaxUploadSize int64
	MediaVariants []int
}

func Load() *Config {
	dataDir := envOr("DATA_DIR", "./data")
	return &Config{
		Port:          envInt("PORT", 8080),
		DataDir:       dataDir,
		DBPath:        filepath.Join(dataDir, "atomicsite.db"),
		MediaDir:      filepath.Join(dataDir, "media"),
		JWTSecret:     envOr("JWT_SECRET", "change-me-in-production"),
		BaseURL:       envOr("BASE_URL", "http://localhost:8080"),
		MaxUploadSize: envInt64("MAX_UPLOAD_SIZE", 20<<20), // 20 MB
		MediaVariants: envIntList("MEDIA_VARIANTS", []int{320, 640, 1280, 1920}),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func envIntList(key string, fallback []int) []int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
