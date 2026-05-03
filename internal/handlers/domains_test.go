package handlers

import (
	"strings"
	"testing"
)

func TestValidateHostname(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		wantErr  bool
		wantHost string
	}{
		{"example.com", false, "example.com"},
		{"  EXAMPLE.com  ", false, "example.com"},
		{"sub.example.com", false, "sub.example.com"},
		{"example.com.", false, "example.com"},
		{"a-b.c-d.example.com", false, "a-b.c-d.example.com"},
		{"xn--n3h.com", false, "xn--n3h.com"},

		{"", true, ""},
		{"   ", true, ""},
		{"example", true, ""},
		{".example.com", true, ""},
		{"-example.com", true, ""},
		{"example-.com", true, ""},
		{"example..com", true, ""},
		{"example.com:8080", true, ""},
		{"http://example.com", true, ""},
		{"example.com/path", true, ""},
		{"example.com?q=1", true, ""},
		{"example.com\nfoo: bar", true, ""},
		{"example.localhost", true, ""},
		{"example.local", true, ""},
		{"localhost", true, ""},
		// 64-char label exceeds RFC 1035 single-label cap.
		{strings.Repeat("a", 64) + ".com", true, ""},
		// Total length over 253.
		{strings.Repeat("a", 250) + ".co", true, ""},
	}
	for _, tc := range cases {
		got, err := validateHostname(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("validateHostname(%q) expected error, got %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("validateHostname(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.wantHost {
			t.Errorf("validateHostname(%q) = %q, want %q", tc.in, got, tc.wantHost)
		}
	}
}

func TestGenerateVerifyToken(t *testing.T) {
	t.Parallel()
	a, err := generateVerifyToken()
	if err != nil {
		t.Fatalf("generateVerifyToken: %v", err)
	}
	b, err := generateVerifyToken()
	if err != nil {
		t.Fatalf("generateVerifyToken: %v", err)
	}
	if a == b {
		t.Errorf("two consecutive tokens collided: %q", a)
	}
	if len(a) != 48 {
		t.Errorf("token length = %d, want 48 hex chars", len(a))
	}
	for _, r := range a {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Errorf("non-hex char in token: %q", a)
			break
		}
	}
}
