package handlers

import (
	"strings"
	"testing"
)

func TestValidateScreenshotURL(t *testing.T) {
	// Reset before / after each subtest, since the allow-list is
	// package-level state that survives across tests.
	t.Cleanup(func() { ConfigureScreenshotAllowList() })

	t.Run("loopback always passes regardless of config", func(t *testing.T) {
		ConfigureScreenshotAllowList()
		for _, raw := range []string{
			"http://localhost:8080/",
			"http://127.0.0.1:5173/preview",
			"https://[::1]/",
		} {
			if err := validateScreenshotURL(raw); err != nil {
				t.Errorf("loopback %q rejected: %v", raw, err)
			}
		}
	})

	t.Run("empty allow-list refuses public hosts", func(t *testing.T) {
		ConfigureScreenshotAllowList()
		err := validateScreenshotURL("https://example.com/")
		if err == nil {
			t.Fatal("empty allow-list accepted public host")
		}
		if !strings.Contains(err.Error(), "screenshot disabled") {
			t.Errorf("expected disabled-screenshot error, got %q", err.Error())
		}
	})

	t.Run("bare domain matches apex and subdomains, not siblings", func(t *testing.T) {
		ConfigureScreenshotAllowList("example.com")
		for _, ok := range []string{
			"https://example.com/",
			"https://www.example.com/",
			"https://foo.bar.example.com/path",
		} {
			if err := validateScreenshotURL(ok); err != nil {
				t.Errorf("expected %q to pass, got %v", ok, err)
			}
		}
		for _, bad := range []string{
			"https://sister-example.com/",
			"https://attacker.com/",
			"https://example.com.attacker.com/",
		} {
			if err := validateScreenshotURL(bad); err == nil {
				t.Errorf("expected %q to fail", bad)
			}
		}
	})

	t.Run("dot-prefixed entry is suffix-only (no apex match)", func(t *testing.T) {
		ConfigureScreenshotAllowList(".tenants.example.com")
		if err := validateScreenshotURL("https://acme.tenants.example.com/"); err != nil {
			t.Errorf("expected subdomain to pass: %v", err)
		}
		// Apex of the suffix itself is not on the allow-list when the
		// entry starts with a dot, since the operator typed a wildcard
		// shape on purpose.
		if err := validateScreenshotURL("https://tenants.example.com/"); err == nil {
			t.Error("expected suffix apex to fail under dot-prefixed entry")
		}
	})

	t.Run("multiple entries OR together", func(t *testing.T) {
		ConfigureScreenshotAllowList("example.com", ".tenants.other.com")
		for _, ok := range []string{
			"https://example.com/",
			"https://acme.tenants.other.com/",
		} {
			if err := validateScreenshotURL(ok); err != nil {
				t.Errorf("expected %q to pass: %v", ok, err)
			}
		}
		if err := validateScreenshotURL("https://other.com/"); err == nil {
			t.Error("expected non-match to fail")
		}
	})

	t.Run("scheme + parse rejects", func(t *testing.T) {
		ConfigureScreenshotAllowList("example.com")
		for _, bad := range []string{
			"",
			"file:///etc/passwd",
			"javascript:alert(1)",
			"https://",
		} {
			if err := validateScreenshotURL(bad); err == nil {
				t.Errorf("expected %q to fail", bad)
			}
		}
	})

	t.Run("ScreenshotAllowedSuffixes returns a copy", func(t *testing.T) {
		ConfigureScreenshotAllowList("a.com", "b.com")
		got := ScreenshotAllowedSuffixes()
		if len(got) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(got))
		}
		got[0] = "mutated"
		// Mutating the returned slice must not affect future lookups.
		again := ScreenshotAllowedSuffixes()
		if again[0] == "mutated" {
			t.Error("returned slice must be a copy, not a shared reference")
		}
	})
}
