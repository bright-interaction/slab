// Package config loads configuration from environment variables.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port          int
	DataDir       string
	DBPath        string
	MediaDir      string
	FontsDir      string // self-hosted woff2 fonts uploaded per site
	JWTSecret     string
	BaseURL       string
	MaxUploadSize int64
	MediaVariants []int

	// Analytics. The CookieProof widget is now embedded directly into the
	// atomicsite Go binary (see internal/builder/widget_embed.go); the
	// remote-API admin token + base URL fields were removed 2026-04-30.
	TrackPath     string // public tracking endpoint prefix, default "/t"
	AnalyticsSalt string // server secret mixed into visitor fingerprints

	// BrightCRM analytics sync (Phase 10 C2). When either the URL or secret
	// is empty, crmsync becomes a no-op so dev environments don't crash.
	// CRMSyncMinInterval throttles non-identified events per (site, visitor).
	BrightCRMWebhookURL string
	// BrightCRMWebhookSecret is the current shared HMAC value, used both to
	// sign outbound /webhooks/atomicsite calls and to verify inbound /t/inbound
	// pings from BrightCRM. BrightCRMWebhookSecretPrevious is the previous
	// value, accepted by the inbound verifier during a rotation grace window
	// so requests in flight signed with the old key still verify. Outbound
	// signing always uses the current value. See internal/sharedsecret.
	BrightCRMWebhookSecret         string
	BrightCRMWebhookSecretPrevious string
	CRMSyncMinInterval             time.Duration

	// AdminReloadToken gates POST /admin/reload-secrets so Dockyard's rotation
	// engine can push a new pair without a container restart.
	AdminReloadToken string

	// Bidirectional CRM personalization (Phase 18). Built sites live on
	// subdomains of BuiltSiteSuffix (default ".slab.example.com")
	// and need to reach /t/visitor + /t/inbound on the admin host. The CORS
	// middleware (server.go isAllowedOrigin) widens to accept any origin
	// whose hostname ends with this suffix.
	BuiltSiteSuffix string
}

func Load() *Config {
	dataDir := envOr("DATA_DIR", "./data")
	return &Config{
		Port:          envInt("PORT", 8080),
		DataDir:       dataDir,
		DBPath:        filepath.Join(dataDir, "atomicsite.db"),
		MediaDir:      filepath.Join(dataDir, "media"),
		FontsDir:      filepath.Join(dataDir, "fonts"),
		JWTSecret:     envOr("JWT_SECRET", "change-me-in-production"),
		BaseURL:       envOr("BASE_URL", "http://localhost:8080"),
		MaxUploadSize: envInt64("MAX_UPLOAD_SIZE", 20<<20), // 20 MB
		MediaVariants: envIntList("MEDIA_VARIANTS", []int{320, 640, 1280, 1920}),

		TrackPath:     envOr("TRACK_PATH", "/t"),
		AnalyticsSalt: envOr("ANALYTICS_SALT", "atomicsite-default-fingerprint-salt-change-me"),

		BrightCRMWebhookURL:            envOr("BRIGHTCRM_WEBHOOK_URL", ""),
		BrightCRMWebhookSecret:         envOr("BRIGHTCRM_WEBHOOK_SECRET", ""),
		BrightCRMWebhookSecretPrevious: envOr("BRIGHTCRM_WEBHOOK_SECRET_PREVIOUS", ""),
		AdminReloadToken:               envOr("ADMIN_RELOAD_TOKEN", ""),
		CRMSyncMinInterval:     time.Duration(envInt("CRM_SYNC_MIN_INTERVAL_SECONDS", 60)) * time.Second,

		BuiltSiteSuffix: envOr("BUILT_SITE_SUFFIX", ".slab.example.com"),
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
