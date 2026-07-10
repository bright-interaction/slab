package handlers

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/bright-interaction/atomicsite/internal/atrest"
	"github.com/bright-interaction/atomicsite/internal/config"
	"github.com/bright-interaction/atomicsite/internal/store"
)

const testShieldKey = "01234567890123456789012345678901"

// TestApps_CredentialsEncryptedAtRest drives the real Install handler with
// a shield key configured and asserts the stored credentials_json is
// ciphertext (no plaintext secret at rest) while both read paths round
// trip. C3 regression guard for app-install credentials.
func TestApps_CredentialsEncryptedAtRest(t *testing.T) {
	_, q := setupDeployTestDB(t)
	ctx := context.Background()
	siteID := "asite0000000000aaaa0099"
	seedSite(t, q, siteID)
	h := NewAppsHandler(&config.Config{ShieldKey: testShieldKey}, q)
	if err := h.SeedCuratedApps(ctx); err != nil {
		t.Fatalf("seed apps: %v", err)
	}
	stripe, err := q.GetAppBySlug(ctx, "stripe")
	if err != nil {
		t.Fatalf("get stripe: %v", err)
	}

	router := chi.NewRouter()
	router.Post("/api/sites/{siteID}/apps/{appID}/install", h.Install)
	code, _ := postJSON(t, router, "/api/sites/"+siteID+"/apps/"+stripe.ID+"/install", map[string]any{
		"credentials": map[string]string{"secret_key": "sk_live_SECRET123"},
	})
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("install status=%d", code)
	}

	row, err := q.GetSiteAppInstall(ctx, store.GetSiteAppInstallParams{SiteID: siteID, AppID: stripe.ID})
	if err != nil {
		t.Fatalf("get install: %v", err)
	}
	if !atrest.IsEncrypted(row.CredentialsJson) {
		t.Fatalf("credentials_json not encrypted at rest: %q", row.CredentialsJson)
	}
	if strings.Contains(row.CredentialsJson, "sk_live_SECRET123") {
		t.Fatalf("plaintext secret leaked into stored row: %q", row.CredentialsJson)
	}
	if got := h.decryptCreds(row.CredentialsJson); !strings.Contains(got, "sk_live_SECRET123") {
		t.Fatalf("decryptCreds round-trip failed: %q", got)
	}
	if ks := credentialsKeySet(h.decryptCreds(row.CredentialsJson)); len(ks) != 1 || ks[0] != "secret_key" {
		t.Fatalf("credentialsKeySet=%v; want [secret_key]", ks)
	}
}

// TestTOTP_SecretEncryptedAtRest asserts the staged TOTP seed is stored
// ciphertext under a shield key and decrypts back so login/verify still
// work (no 2FA lockout). C3 regression guard for the TOTP seed.
func TestTOTP_SecretEncryptedAtRest(t *testing.T) {
	db, q := setupDeployTestDB(t)
	ctx := context.Background()
	if _, err := db.Exec(
		`INSERT INTO users (id, email, password_hash, name, role) VALUES ('u1','u1@e.com','','','')`,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := &AuthHandler{cfg: &config.Config{ShieldKey: testShieldKey}, queries: q}

	const seed = "JBSWY3DPEHPK3PXP"
	enc, err := h.secretCipher().Encrypt(seed)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if err := q.SetUserTOTPSecret(ctx, store.SetUserTOTPSecretParams{TotpSecret: enc, ID: "u1"}); err != nil {
		t.Fatalf("set secret: %v", err)
	}
	row, err := q.GetUserByID(ctx, "u1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if !atrest.IsEncrypted(row.TotpSecret) {
		t.Fatalf("totp_secret not encrypted at rest: %q", row.TotpSecret)
	}
	if strings.Contains(row.TotpSecret, seed) {
		t.Fatalf("plaintext TOTP seed at rest: %q", row.TotpSecret)
	}
	got, err := h.secretCipher().Decrypt(row.TotpSecret)
	if err != nil || got != seed {
		t.Fatalf("decrypt round-trip = %q err=%v; want %q", got, err, seed)
	}
}
