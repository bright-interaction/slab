// Package config loads configuration from environment variables.
package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port      int
	DataDir   string
	DBPath    string
	JWTSecret string
	BaseURL   string
}

func Load() *Config {
	dataDir := envOr("DATA_DIR", "./data")
	return &Config{
		Port:      envInt("PORT", 8080),
		DataDir:   dataDir,
		DBPath:    filepath.Join(dataDir, "atomicsite.db"),
		JWTSecret: envOr("JWT_SECRET", "change-me-in-production"),
		BaseURL:   envOr("BASE_URL", "http://localhost:8080"),
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
