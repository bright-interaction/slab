package cookieproof

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestEnsureDomain_HappyPath verifies both calls (PUT /api/config + POST
// /api/settings/domains) fire with the expected bodies and headers.
func TestEnsureDomain_HappyPath(t *testing.T) {
	var configCalled, originCalled atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		configCalled.Store(true)
		if r.Method != http.MethodPut {
			t.Errorf("config: expected PUT, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("config: bad auth header %q", got)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("config: parse body: %v", err)
		}
		if body["domain"] != "example.com" {
			t.Errorf("config: expected domain=example.com, got %v", body["domain"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"domain":"example.com"}`))
	})
	mux.HandleFunc("/api/settings/domains", func(w http.ResponseWriter, r *http.Request) {
		originCalled.Store(true)
		if r.Method != http.MethodPost {
			t.Errorf("origins: expected POST, got %s", r.Method)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("origins: parse body: %v", err)
		}
		if !strings.HasPrefix(body["origin"].(string), "https://") {
			t.Errorf("origins: expected https origin, got %v", body["origin"])
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"x","origin":"https://example.com"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL, "test-token")
	if err := c.EnsureDomain(context.Background(), "example.com"); err != nil {
		t.Fatalf("EnsureDomain: %v", err)
	}
	if !configCalled.Load() {
		t.Error("expected PUT /api/config to be called")
	}
	if !originCalled.Load() {
		t.Error("expected POST /api/settings/domains to be called")
	}
}

// TestEnsureDomain_DuplicateOriginIsOK confirms 409 from /api/settings/domains
// is treated as success (idempotent re-runs).
func TestEnsureDomain_DuplicateOriginIsOK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/settings/domains", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"Origin already exists"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL, "tok")
	if err := c.EnsureDomain(context.Background(), "example.com"); err != nil {
		t.Fatalf("EnsureDomain should ignore 409, got: %v", err)
	}
}

// TestEnsureDomain_SkipsWithoutCreds returns nil instead of erroring when the
// admin token isn't configured. Build pipelines must keep working in dev.
func TestEnsureDomain_SkipsWithoutCreds(t *testing.T) {
	c := New("https://nope.invalid", "")
	if err := c.EnsureDomain(context.Background(), "example.com"); err != nil {
		t.Errorf("expected nil error when authToken is empty, got %v", err)
	}
	c2 := New("", "tok")
	if err := c2.EnsureDomain(context.Background(), "example.com"); err != nil {
		t.Errorf("expected nil error when baseURL is empty, got %v", err)
	}
}

// TestEnsureDomain_ConfigErrorBubbles wraps non-2xx, non-409 responses as
// errors so the build can log + skip.
func TestEnsureDomain_ConfigErrorBubbles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"This domain is owned by another organization"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL, "tok")
	err := c.EnsureDomain(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected status code in error, got %v", err)
	}
}
