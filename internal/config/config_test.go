package config

import (
	"strings"
	"testing"
)

// TestValidate_LocalDevAllowsDefaults confirms the guard does not block
// `go run ./cmd/server` with no env vars set. Local dev is the only
// branch where Validate returns nil with sentinel secrets in place.
func TestValidate_LocalDevAllowsDefaults(t *testing.T) {
	cfg := &Config{
		BaseURL:       "http://localhost:8080",
		JWTSecret:     DefaultJWTSecret,
		AnalyticsSalt: DefaultAnalyticsSalt,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should pass for localhost dev with defaults; got %v", err)
	}
}

// TestValidate_ProductionRejectsDefaults asserts the audit's C3 guard.
// Any single sentinel value in production-like config is a refuse-to-start.
func TestValidate_ProductionRejectsDefaults(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
		want string
	}{
		{
			name: "default jwt secret",
			mut:  func(c *Config) { c.JWTSecret = DefaultJWTSecret },
			want: "JWT_SECRET",
		},
		{
			name: "empty jwt secret",
			mut:  func(c *Config) { c.JWTSecret = "" },
			want: "JWT_SECRET",
		},
		{
			name: "short jwt secret",
			mut:  func(c *Config) { c.JWTSecret = "tooshort" },
			want: "32 for HS256",
		},
		{
			name: "default analytics salt",
			mut:  func(c *Config) { c.AnalyticsSalt = DefaultAnalyticsSalt },
			want: "ANALYTICS_SALT",
		},
		{
			name: "empty analytics salt",
			mut:  func(c *Config) { c.AnalyticsSalt = "" },
			want: "ANALYTICS_SALT",
		},
		{
			name: "empty shield key",
			mut:  func(c *Config) { c.ShieldKey = "" },
			want: "SLAB_SHIELD_KEY",
		},
		{
			name: "short shield key",
			mut:  func(c *Config) { c.ShieldKey = "tooshort" },
			want: "SLAB_SHIELD_KEY",
		},
		{
			name: "non-https base url",
			mut:  func(c *Config) { c.BaseURL = "http://prod.example.com" },
			want: "not HTTPS",
		},
		{
			name: "no primary domain or built-site suffix",
			mut: func(c *Config) {
				c.PrimaryDomain = ""
				c.BuiltSiteSuffix = ""
			},
			want: "SLAB_PRIMARY_DOMAIN",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := goodConfig()
			tc.mut(cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

// TestValidate_ProductionAcceptsStrongConfig confirms a fully-set config
// passes; guards against accidental over-strictness in Validate.
func TestValidate_ProductionAcceptsStrongConfig(t *testing.T) {
	if err := goodConfig().Validate(); err != nil {
		t.Errorf("goodConfig should validate; got %v", err)
	}
}

// TestValidate_BuiltSiteSuffixOnlyOK confirms the legacy multi-tenant
// shape (BUILT_SITE_SUFFIX set, PRIMARY_DOMAIN unset) still satisfies
// the validator. Either knob is enough; both unset is the failure mode.
func TestValidate_BuiltSiteSuffixOnlyOK(t *testing.T) {
	cfg := goodConfig()
	cfg.PrimaryDomain = ""
	cfg.BuiltSiteSuffix = ".tenants.example.com"
	if err := cfg.Validate(); err != nil {
		t.Errorf("multi-tenant config should validate; got %v", err)
	}
}

// TestValidateAdminPassword_RejectsDefaultInProd asserts the seed-time
// guard. Empty or default password in production is a hard fail.
func TestValidateAdminPassword_RejectsDefaultInProd(t *testing.T) {
	cfg := goodConfig()
	cases := map[string]string{
		"empty":   "",
		"default": DefaultAdminPassword,
		"short":   "abc12345",
	}
	for name, pw := range cases {
		t.Run(name, func(t *testing.T) {
			if err := cfg.ValidateAdminPassword(pw); err == nil {
				t.Errorf("ValidateAdminPassword(%q) should fail in prod", pw)
			}
		})
	}
}

// TestValidateAdminPassword_LocalDevPermissive confirms the seed-time
// guard skips local dev so contributors can run the server with no env.
func TestValidateAdminPassword_LocalDevPermissive(t *testing.T) {
	cfg := &Config{BaseURL: "http://localhost:8080"}
	for _, pw := range []string{"", DefaultAdminPassword, "short"} {
		if err := cfg.ValidateAdminPassword(pw); err != nil {
			t.Errorf("local dev should accept %q; got %v", pw, err)
		}
	}
}

// TestValidate_DeploymentMode_Common covers the build-tag-independent
// cases: empty defaults to "single", "single" is always accepted, and
// unknown values are always rejected. Cloud-mode behaviour is split
// into two tag-gated tests below because it depends on ee.IsAvailable().
func TestValidate_DeploymentMode_Common(t *testing.T) {
	cases := []struct {
		name    string
		mode    string
		baseURL string
		wantErr string
	}{
		{name: "empty defaults to single (localhost)", mode: "", baseURL: "http://localhost:8080"},
		{name: "single explicit (localhost)", mode: "single", baseURL: "http://localhost:8080"},
		{name: "single explicit (prod)", mode: "single"},
		{name: "unknown value rejected (localhost)", mode: "enterprise", baseURL: "http://localhost:8080", wantErr: "is not a recognised value"},
		{name: "unknown value rejected (prod)", mode: "enterprise", wantErr: "is not a recognised value"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := goodConfig()
			cfg.DeploymentMode = tc.mode
			if tc.baseURL != "" {
				cfg.BaseURL = tc.baseURL
			}
			err := cfg.Validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("expected success, got %v", err)
			case tc.wantErr != "" && err == nil:
				t.Errorf("expected error containing %q, got nil", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestIsLocalDev_RecognisesLoopbackVariants: loopback can come in three
// shapes; all three should bypass the validation guards.
func TestIsLocalDev_RecognisesLoopbackVariants(t *testing.T) {
	cases := map[string]bool{
		"http://localhost:8080":  true,
		"http://127.0.0.1:8080":  true,
		"http://0.0.0.0:8080":    true,
		"https://prod.example":   false,
		"http://prod.example":    false,
		"http://example.com:443": false,
	}
	for url, want := range cases {
		got := (&Config{BaseURL: url}).IsLocalDev()
		if got != want {
			t.Errorf("IsLocalDev(%q) = %v, want %v", url, got, want)
		}
	}
}

func goodConfig() *Config {
	return &Config{
		BaseURL:        "https://slab.example.com",
		JWTSecret:      "this-is-a-strong-32-byte-secret-value-here",
		AnalyticsSalt:  "this-is-a-random-32-byte-fingerprint-salt-value",
		ShieldKey:      "0123456789abcdef0123456789abcdef",
		PrimaryDomain:  "example.com",
		DeploymentMode: DeploymentModeSingle,
	}
}

// TestEnv_LegacyAtomicsiteFallback proves the SLAB_ env prefix falls back to
// the pre-rename ATOMICSITE_ name, so an env-file from before the Slab rename
// keeps working. SLAB_ wins when both are set.
func TestEnv_LegacyAtomicsiteFallback(t *testing.T) {
	t.Setenv("SLAB_PRIMARY_DOMAIN", "")
	t.Setenv("ATOMICSITE_PRIMARY_DOMAIN", "legacy.example.com")
	if got := Env("SLAB_PRIMARY_DOMAIN"); got != "legacy.example.com" {
		t.Fatalf("legacy fallback: Env(SLAB_PRIMARY_DOMAIN) = %q, want legacy.example.com", got)
	}

	t.Setenv("SLAB_PRIMARY_DOMAIN", "current.example.com")
	if got := Env("SLAB_PRIMARY_DOMAIN"); got != "current.example.com" {
		t.Fatalf("SLAB_ must win over legacy: got %q, want current.example.com", got)
	}

	// A non-SLAB_ key has no legacy fallback.
	t.Setenv("PORT", "9090")
	if got := Env("PORT"); got != "9090" {
		t.Fatalf("Env(PORT) = %q, want 9090", got)
	}
}
