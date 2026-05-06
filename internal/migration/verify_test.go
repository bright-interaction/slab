package migration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ----- rewriteHostToDeployed -----

func TestRewriteHostToDeployed_FullURL(t *testing.T) {
	got, err := rewriteHostToDeployed("https://old-site.example/about?ref=x", "new.example")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "https://new.example/about?ref=x" {
		t.Errorf("rewrite: %q", got)
	}
}

func TestRewriteHostToDeployed_BarePath(t *testing.T) {
	got, err := rewriteHostToDeployed("/contact", "new.example")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "https://new.example/contact" {
		t.Errorf("rewrite: %q", got)
	}
}

func TestRewriteHostToDeployed_Empty(t *testing.T) {
	if _, err := rewriteHostToDeployed("", "new.example"); err == nil {
		t.Errorf("empty source must error")
	}
}

func TestRewriteHostToDeployed_PreservesFragment(t *testing.T) {
	got, _ := rewriteHostToDeployed("https://x/p?q=1#sec", "new.example")
	if !strings.Contains(got, "#sec") || !strings.Contains(got, "?q=1") {
		t.Errorf("fragment/query lost: %q", got)
	}
}

// ----- VerifyLive: terminal 200 -----

func TestVerifyLive_HappyPath_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	results, err := VerifyLive(context.Background(),
		[]string{"/about", "/contact"},
		srv.URL,
		VerifyOptions{AllowPrivate: true, PolitenessDelay: 1 * time.Millisecond},
	)
	if err != nil {
		t.Fatalf("VerifyLive: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.OK || r.StatusCode != 200 {
			t.Errorf("expected OK 200, got %+v", r)
		}
		if r.Hops != 0 {
			t.Errorf("0 hops expected for direct 200, got %d", r.Hops)
		}
	}
}

// ----- VerifyLive: 301 chain to 200 -----

func TestVerifyLive_RedirectChainOK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/new", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	results, err := VerifyLive(context.Background(),
		[]string{"/old"}, srv.URL,
		VerifyOptions{AllowPrivate: true, PolitenessDelay: 1 * time.Millisecond},
	)
	if err != nil {
		t.Fatalf("VerifyLive: %v", err)
	}
	r := results[0]
	if !r.OK {
		t.Errorf("redirect chain to 200 should be OK, got %+v", r)
	}
	if r.StatusCode != 200 {
		t.Errorf("final status: %d", r.StatusCode)
	}
	if r.Hops != 1 {
		t.Errorf("hops: want 1, got %d", r.Hops)
	}
	if !strings.HasSuffix(r.FinalURL, "/new") {
		t.Errorf("final URL: %q", r.FinalURL)
	}
}

// ----- VerifyLive: terminal 404 -----

func TestVerifyLive_TerminalFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	results, _ := VerifyLive(context.Background(), []string{"/missing"}, srv.URL,
		VerifyOptions{AllowPrivate: true, PolitenessDelay: 1 * time.Millisecond})
	r := results[0]
	if r.OK {
		t.Errorf("404 must not be OK: %+v", r)
	}
	if r.StatusCode != 404 {
		t.Errorf("status: %d", r.StatusCode)
	}
	if !strings.Contains(r.Error, "404") {
		t.Errorf("error should mention status code, got %q", r.Error)
	}
}

// ----- VerifyLive: redirect loop -----

func TestVerifyLive_RedirectLoopHopBudget(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Redirect(w, r, "/b", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Redirect(w, r, "/a", http.StatusMovedPermanently)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	results, _ := VerifyLive(context.Background(), []string{"/a"}, srv.URL,
		VerifyOptions{AllowPrivate: true, PolitenessDelay: 1 * time.Millisecond, MaxHops: 3})
	r := results[0]
	if r.OK {
		t.Errorf("loop must not be OK: %+v", r)
	}
	if !strings.Contains(r.Error, "exceeded") {
		t.Errorf("error should mention hop budget, got %q", r.Error)
	}
	if hits == 0 {
		t.Errorf("server should have received at least one hit")
	}
}

// ----- VerifyLive: empty Location header -----

func TestVerifyLive_RedirectEmptyLocation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently) // no Location
	}))
	defer srv.Close()
	results, _ := VerifyLive(context.Background(), []string{"/x"}, srv.URL,
		VerifyOptions{AllowPrivate: true, PolitenessDelay: 1 * time.Millisecond})
	r := results[0]
	if r.OK {
		t.Errorf("empty Location must not be OK")
	}
	if !strings.Contains(r.Error, "empty Location") {
		t.Errorf("error should mention empty Location: %q", r.Error)
	}
}

// ----- VerifyLive: SSRF guard inherited -----

func TestVerifyLive_SSRFGuardBlocksLoopback(t *testing.T) {
	results, _ := VerifyLive(context.Background(),
		[]string{"https://old.example/about"},
		"127.0.0.1:1",
		VerifyOptions{PolitenessDelay: 1 * time.Millisecond}, // AllowPrivate left false
	)
	r := results[0]
	if r.OK {
		t.Errorf("loopback target must be blocked")
	}
	if !strings.Contains(strings.ToLower(r.Error), "blocked") &&
		!strings.Contains(strings.ToLower(r.Error), "private") {
		t.Errorf("error should mention SSRF block: %q", r.Error)
	}
}

// ----- Per-host serialization -----

func TestVerifyLive_PolitenessSerializesPerHost(t *testing.T) {
	var inflight, peak int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := atomic.AddInt32(&inflight, 1)
		for {
			cur := atomic.LoadInt32(&peak)
			if now <= cur || atomic.CompareAndSwapInt32(&peak, cur, now) {
				break
			}
		}
		time.Sleep(15 * time.Millisecond)
		atomic.AddInt32(&inflight, -1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	urls := []string{"/a", "/b", "/c", "/d"}
	_, _ = VerifyLive(context.Background(), urls, srv.URL,
		VerifyOptions{AllowPrivate: true, Concurrency: 8, PolitenessDelay: 5 * time.Millisecond},
	)
	if atomic.LoadInt32(&peak) > 1 {
		t.Errorf("politeness should serialise per host; peak in-flight = %d", peak)
	}
}

// ----- Empty input rejected -----

func TestVerifyLive_RequiresInputs(t *testing.T) {
	if _, err := VerifyLive(context.Background(), nil, "x", VerifyOptions{}); err == nil {
		t.Errorf("empty source list must error")
	}
	if _, err := VerifyLive(context.Background(), []string{"/x"}, "", VerifyOptions{}); err == nil {
		t.Errorf("empty deployed domain must error")
	}
}
