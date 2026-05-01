// Package config loads configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/brightinteraction/atomicsite/ee"
)

// Default sentinel values. The Validate() guard refuses to start in
// production-like deployments when any one of these is still in place.
const (
	DefaultJWTSecret     = "change-me-in-production"
	DefaultAnalyticsSalt = "atomicsite-default-fingerprint-salt-change-me"
	DefaultAdminPassword = "changeme123"
)

// DeploymentMode values. The OSS distribution defaults to "single";
// "cloud" requires a binary built with -tags ee.
const (
	DeploymentModeSingle = "single"
	DeploymentModeCloud  = "cloud"
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

	// PrimaryDomain is the apex domain this Atomic Site instance serves
	// (e.g. "example.com"). Empty for local dev or for deployments that
	// host many unrelated tenants. When set, it surfaces in the agent
	// context as the canonical "what domain am I building for?" answer
	// and seeds defaults for fresh sites (cookieproof_domain, default
	// canonical base, CORS allow-list). Read from the
	// ATOMICSITE_PRIMARY_DOMAIN environment variable.
	PrimaryDomain string

	// BuiltSiteSuffix is set ONLY for multi-tenant deployments where
	// each Atomic Site tenant gets a wildcard subdomain (e.g. every
	// tenant lives at <slug>.tenants.example.com). The CORS middleware
	// (server.go isAllowedOriginForPath) widens to accept any origin
	// whose hostname ends with this suffix on public visitor paths.
	//
	// Empty by default. The typical single-deployment-per-customer
	// shape leaves it unset; PrimaryDomain takes its place for any
	// deployment that serves a single root domain. Read from the
	// BUILT_SITE_SUFFIX environment variable.
	BuiltSiteSuffix string

	// DeploymentMode selects the operating shape: "single" (the OSS
	// default, one Atomic Site instance for one root domain) or
	// "cloud" (multi-tenant edge orchestration, requires a binary
	// built with -tags ee). Read from ATOMICSITE_DEPLOYMENT_MODE.
	// Validate() rejects unknown values and refuses to start in
	// "cloud" mode when ee.IsAvailable() returns false.
	DeploymentMode string
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

		PrimaryDomain:   envOr("ATOMICSITE_PRIMARY_DOMAIN", ""),
		BuiltSiteSuffix: envOr("BUILT_SITE_SUFFIX", ""),
		DeploymentMode:  envOr("ATOMICSITE_DEPLOYMENT_MODE", DeploymentModeSingle),
	}
}

// Validate returns nil when the loaded config is safe for the deployment
// shape implied by BaseURL. Production-like deployments (BaseURL != localhost)
// must override every secret and operational toggle that has a known default;
// any one still at its sentinel value is treated as a misconfiguration and
// the server refuses to start.
//
// Why a refuse-to-start guard rather than a warning: every previous
// "warning-only" approach in the codebase has been ignored at least once
// in production. The boot-time hard fail is the only mechanism that
// guarantees the operator notices.
//
// Local development (BaseURL starts with http://localhost) keeps the
// permissive behaviour so contributors don't have to set five env vars
// to run `go test` or `go run`.
func (c *Config) Validate() error {
	// Deployment-mode validation runs even on localhost: a misspelt mode
	// or "cloud" on an OSS binary should fail loudly regardless of where
	// the operator is running it. Localhost still bypasses the secret /
	// HTTPS / domain guards below.
	mode := strings.TrimSpace(c.DeploymentMode)
	if mode == "" {
		mode = DeploymentModeSingle
	}
	switch mode {
	case DeploymentModeSingle:
		// always fine
	case DeploymentModeCloud:
		if !ee.IsAvailable() {
			return fmt.Errorf("ATOMICSITE_DEPLOYMENT_MODE=%s requires a binary built with -tags ee; this is the OSS build", mode)
		}
	default:
		return fmt.Errorf("ATOMICSITE_DEPLOYMENT_MODE=%q is not a recognised value; use %q or %q", mode, DeploymentModeSingle, DeploymentModeCloud)
	}

	if c.IsLocalDev() {
		return nil
	}

	var problems []string

	if c.JWTSecret == "" || c.JWTSecret == DefaultJWTSecret {
		problems = append(problems, "JWT_SECRET is unset or equal to the documented default; set a 32+ byte random value")
	} else if len(c.JWTSecret) < 32 {
		problems = append(problems, fmt.Sprintf("JWT_SECRET is %d bytes; require at least 32 for HS256 collision resistance", len(c.JWTSecret)))
	}

	if c.AnalyticsSalt == "" || c.AnalyticsSalt == DefaultAnalyticsSalt {
		problems = append(problems, "ANALYTICS_SALT is unset or equal to the documented default; set a random value (visitor fingerprints become predictable otherwise)")
	}

	// HTTPS scheme guard. The auth cookie's Secure attribute is gated on
	// "not localhost"; without HTTPS in production, the cookie still gets
	// set Secure=true and never travels, which manifests as login that
	// "doesn't stick". Fail fast with the right error message.
	if !strings.HasPrefix(c.BaseURL, "https://") {
		problems = append(problems, fmt.Sprintf("BASE_URL=%q is not HTTPS; production deployments must serve over TLS or sessions will not persist", c.BaseURL))
	}

	// Either ATOMICSITE_PRIMARY_DOMAIN (single-root-domain shape, the
	// typical SaaS deploy) or BUILT_SITE_SUFFIX (legacy multi-tenant
	// subdomain shape) must be set in production. Both unset means the
	// operator hasn't told Atomic Site which domain it's responsible
	// for, and any defaults we'd seed for fresh sites would be wrong.
	// Allowing both is fine: PrimaryDomain seeds the canonical site,
	// BuiltSiteSuffix widens CORS for legacy subdomain tenancy.
	if strings.TrimSpace(c.PrimaryDomain) == "" && strings.TrimSpace(c.BuiltSiteSuffix) == "" {
		problems = append(problems, "ATOMICSITE_PRIMARY_DOMAIN is unset; set it to the apex domain this Atomic Site instance serves (e.g. ATOMICSITE_PRIMARY_DOMAIN=example.com). Multi-tenant deployments that use wildcard subdomains may set BUILT_SITE_SUFFIX instead.")
	}

	if len(problems) > 0 {
		return errors.New("config validation failed:\n  - " + strings.Join(problems, "\n  - "))
	}
	return nil
}

// ValidateAdminPassword is called at admin-seed time only. It enforces the
// same "no defaults in production" guard as Validate but reports the password
// problem separately so the caller can fail with a useful message even when
// the rest of the config is fine. Empty password in production is a hard
// fail; the seed path must abort rather than silently use the documented
// default.
func (c *Config) ValidateAdminPassword(adminPassword string) error {
	if c.IsLocalDev() {
		return nil
	}
	if adminPassword == "" || adminPassword == DefaultAdminPassword {
		return errors.New("ADMIN_PASSWORD is unset or equal to the documented default; production deployments must set ADMIN_PASSWORD to a strong unique value before first boot")
	}
	if len(adminPassword) < 12 {
		return fmt.Errorf("ADMIN_PASSWORD is only %d characters; require at least 12 (use a passphrase generator)", len(adminPassword))
	}
	return nil
}

// IsLocalDev reports whether the BaseURL points at localhost. Local dev
// keeps the lenient defaults (no secrets, no HTTPS) so contributors can
// run the server with a single command. The Validate guards bypass on
// this branch.
func (c *Config) IsLocalDev() bool {
	return strings.HasPrefix(c.BaseURL, "http://localhost") ||
		strings.HasPrefix(c.BaseURL, "http://127.0.0.1") ||
		strings.HasPrefix(c.BaseURL, "http://0.0.0.0")
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
