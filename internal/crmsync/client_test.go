package crmsync

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testSecret = "test-secret-32-bytes-x-x-x-x-x-x"

// TestClientSend_SignsAndPosts confirms the signature is computed over the
// raw JSON body (in hex) and shipped in X-Atomicsite-Signature, and that
// the server-side payload matches what we sent.
func TestClientSend_SignsAndPosts(t *testing.T) {
	var (
		gotSig  string
		gotBody []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get(SignatureHeader)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, testSecret)

	event := Event{
		Event:      EventIdentified,
		SiteID:     "site-1",
		VisitorID:  "v_aaa",
		Email:      "alice@acme.com",
		OccurredAt: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		Page:       Page{URL: "https://example.com/", Path: "/", Title: "Home"},
	}

	if err := client.Send(context.Background(), event); err != nil {
		t.Fatalf("Send: %v", err)
	}

	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write(gotBody)
	wantSig := hex.EncodeToString(mac.Sum(nil))
	if gotSig != wantSig {
		t.Fatalf("signature mismatch: got %q, want %q", gotSig, wantSig)
	}

	var roundtrip Event
	if err := json.Unmarshal(gotBody, &roundtrip); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if roundtrip.Email != event.Email || roundtrip.VisitorID != event.VisitorID || roundtrip.Event != event.Event {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", roundtrip, event)
	}
}

func TestClientSend_AcceptsNoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	client := NewClient(srv.URL, testSecret)
	if err := client.Send(context.Background(), Event{Event: EventVisit, VisitorID: "v"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestClientSend_NonSuccessStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	client := NewClient(srv.URL, testSecret)
	err := client.Send(context.Background(), Event{Event: EventVisit, VisitorID: "v"})
	if err == nil {
		t.Fatal("Send should have returned an error on 400")
	}
}

func TestClientSend_DisabledIsNoOp(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cases := []struct {
		name   string
		url    string
		secret string
	}{
		{"empty url", "", testSecret},
		{"empty secret", srv.URL, ""},
		{"both empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := NewClient(c.url, c.secret)
			if client.Enabled() {
				t.Fatal("client should be disabled")
			}
			if err := client.Send(context.Background(), Event{Event: EventVisit, VisitorID: "v"}); err != nil {
				t.Fatalf("disabled Send returned err: %v", err)
			}
		})
	}
	if calls != 0 {
		t.Fatalf("server received %d calls, want 0", calls)
	}
}

func TestThrottler_DropsWithinWindow(t *testing.T) {
	t0 := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	now := t0
	tr := NewThrottler(60 * time.Second)
	tr.clock = func() time.Time { return now }

	ev := Event{Event: EventVisit, SiteID: "s1", VisitorID: "v1"}

	if !tr.Allow(ev) {
		t.Fatal("first event must pass")
	}
	now = t0.Add(30 * time.Second)
	if tr.Allow(ev) {
		t.Fatal("second event within 60s must be dropped")
	}
	now = t0.Add(61 * time.Second)
	if !tr.Allow(ev) {
		t.Fatal("event after 61s must pass again")
	}
}

func TestThrottler_IdentifiedAlwaysPasses(t *testing.T) {
	tr := NewThrottler(60 * time.Second)
	now := time.Now()
	tr.clock = func() time.Time { return now }

	visit := Event{Event: EventVisit, SiteID: "s1", VisitorID: "v1"}
	if !tr.Allow(visit) {
		t.Fatal("first visit must pass")
	}
	ident := Event{Event: EventIdentified, SiteID: "s1", VisitorID: "v1"}
	if !tr.Allow(ident) {
		t.Fatal("identified must always pass")
	}
	if !tr.Allow(ident) {
		t.Fatal("a second identified must also always pass")
	}
}

func TestThrottler_ZeroIntervalDisabled(t *testing.T) {
	tr := NewThrottler(0)
	ev := Event{Event: EventVisit, SiteID: "s1", VisitorID: "v1"}
	for i := 0; i < 10; i++ {
		if !tr.Allow(ev) {
			t.Fatalf("event %d dropped despite zero interval", i)
		}
	}
}

func TestThrottler_PerVisitorIsolation(t *testing.T) {
	tr := NewThrottler(60 * time.Second)
	now := time.Now()
	tr.clock = func() time.Time { return now }

	a := Event{Event: EventVisit, SiteID: "s1", VisitorID: "v1"}
	b := Event{Event: EventVisit, SiteID: "s1", VisitorID: "v2"}

	if !tr.Allow(a) {
		t.Fatal("a first must pass")
	}
	if !tr.Allow(b) {
		t.Fatal("b first must pass; per-visitor key must isolate from a")
	}
	if tr.Allow(a) {
		t.Fatal("a within window must drop")
	}
	if tr.Allow(b) {
		t.Fatal("b within window must drop")
	}
}

func TestNilThrottlerAllowsAll(t *testing.T) {
	var tr *Throttler
	ev := Event{Event: EventVisit, SiteID: "s", VisitorID: "v"}
	if !tr.Allow(ev) {
		t.Fatal("nil throttler must always allow")
	}
}
