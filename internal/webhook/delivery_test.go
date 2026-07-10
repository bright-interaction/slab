package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	dbpkg "github.com/bright-interaction/atomicsite/internal/db"
	"github.com/bright-interaction/atomicsite/internal/store"
)

func newWebhookTestStack(t *testing.T) (*sql.DB, *store.Queries, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wh.db")
	sqlDB, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(string(dbpkg.Schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	q := store.New(sqlDB)
	siteID := "abcdef0123456789abcdef77"
	if err := q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: siteID, Name: "WH", Slug: "wh", Domain: "", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	}); err != nil {
		t.Fatal(err)
	}
	return sqlDB, q, siteID
}

func seedSubscription(t *testing.T, q *store.Queries, siteID, targetURL, secret string, eventTypes []string) string {
	t.Helper()
	id := newID()
	typesJSON, _ := json.Marshal(eventTypes)
	if err := q.CreateWebhookSubscription(context.Background(), store.CreateWebhookSubscriptionParams{
		ID: id, SiteID: siteID, TargetUrl: targetURL, Secret: secret,
		EventTypesJson: string(typesJSON), Active: 1,
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSignBody_DeterministicAndConstantTimeSafe(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	a := signBody("secret", body)
	b := signBody("secret", body)
	if a != b {
		t.Errorf("HMAC must be deterministic, got %s vs %s", a, b)
	}
	c := signBody("different", body)
	if a == c {
		t.Error("HMAC must change with secret")
	}
	if len(a) != 64 {
		t.Errorf("hex-encoded SHA256 must be 64 chars, got %d", len(a))
	}
}

func TestSubscriptionMatches_FilterSemantics(t *testing.T) {
	cases := []struct {
		filter string
		event  string
		want   bool
	}{
		{"", "page.created", true},   // empty = all
		{"[]", "page.created", true}, // explicit empty = all
		{`["page.created"]`, "page.created", true},
		{`["page.created"]`, "page.deleted", false},
		{`["page.created","page.deleted"]`, "page.deleted", true},
		{`["*"]`, "anything.goes", true},
		{`{not-json}`, "page.created", false}, // malformed = fail closed
	}
	for _, tc := range cases {
		got := subscriptionMatches(tc.filter, tc.event)
		if got != tc.want {
			t.Errorf("filter=%q event=%q -> %v, want %v", tc.filter, tc.event, got, tc.want)
		}
	}
}

func TestEmitter_CreatesDeliveriesForActiveMatchingSubscriptions(t *testing.T) {
	sqlDB, q, siteID := newWebhookTestStack(t)
	_ = sqlDB
	subA := seedSubscription(t, q, siteID, "https://a.example.com", "secA", []string{"page.created"})
	subB := seedSubscription(t, q, siteID, "https://b.example.com", "secB", []string{"page.deleted"})
	subC := seedSubscription(t, q, siteID, "https://c.example.com", "secC", []string{}) // wildcard
	_ = subB

	em := NewEmitter(q)
	em.Emit(context.Background(), siteID, EventPageCreated, map[string]any{"page_id": "p1"})

	// Only subA + subC should have deliveries; subB filtered out.
	rows, err := q.ListWebhookDeliveriesBySite(context.Background(), store.ListWebhookDeliveriesBySiteParams{
		SiteID: siteID, Limit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	deliveredFor := map[string]bool{}
	for _, r := range rows {
		deliveredFor[r.SubscriptionID] = true
		if r.EventType != EventPageCreated {
			t.Errorf("unexpected event type %q on delivery", r.EventType)
		}
	}
	if !deliveredFor[subA] {
		t.Error("subA should have a delivery (matched event)")
	}
	if !deliveredFor[subC] {
		t.Error("subC should have a delivery (wildcard / empty filter)")
	}
	if deliveredFor[subB] {
		t.Error("subB should NOT have a delivery (filter excluded the event)")
	}
}

func TestEmitter_NoOpForInactiveSubscription(t *testing.T) {
	sqlDB, q, siteID := newWebhookTestStack(t)
	_ = sqlDB
	id := seedSubscription(t, q, siteID, "https://x.example.com", "s", []string{})
	// Flip to inactive.
	if err := q.UpdateWebhookSubscription(context.Background(), store.UpdateWebhookSubscriptionParams{
		ID: id, TargetUrl: "https://x.example.com", EventTypesJson: "[]", Active: 0,
	}); err != nil {
		t.Fatal(err)
	}

	NewEmitter(q).Emit(context.Background(), siteID, EventPageCreated, nil)
	rows, _ := q.ListWebhookDeliveriesBySite(context.Background(), store.ListWebhookDeliveriesBySiteParams{
		SiteID: siteID, Limit: 100,
	})
	if len(rows) != 0 {
		t.Errorf("inactive subscription should not receive deliveries; got %d", len(rows))
	}
}

func TestDelivery_HappyPathPostsAndMarksSucceeded(t *testing.T) {
	sqlDB, q, siteID := newWebhookTestStack(t)
	_ = sqlDB

	var (
		mu          sync.Mutex
		gotEvent    string
		gotSig      string
		gotDelivery string
		gotBody     []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotEvent = r.Header.Get("X-Atomicsite-Event")
		gotSig = r.Header.Get("X-Atomicsite-Signature")
		gotDelivery = r.Header.Get("X-Atomicsite-Delivery")
		gotBody = body
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	subID := seedSubscription(t, q, siteID, srv.URL, "shh", []string{})
	NewEmitter(q).Emit(context.Background(), siteID, EventPageCreated, map[string]any{"id": "p1"})

	mgr := NewDeliveryManager(q)
	mgr.RunOnce(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if gotEvent != EventPageCreated {
		t.Errorf("X-Atomicsite-Event = %q, want %q", gotEvent, EventPageCreated)
	}
	wantSig := "sha256=" + signBody("shh", gotBody)
	if gotSig != wantSig {
		t.Errorf("signature mismatch:\n got: %s\nwant: %s", gotSig, wantSig)
	}
	if gotDelivery == "" {
		t.Error("X-Atomicsite-Delivery must be present")
	}
	if !strings.Contains(string(gotBody), `"id":"p1"`) {
		t.Errorf("body did not contain payload; got %s", gotBody)
	}

	// DB state: the row should be 'success' with attempts=1.
	rows, _ := q.ListWebhookDeliveriesBySite(context.Background(), store.ListWebhookDeliveriesBySiteParams{
		SiteID: siteID, Limit: 10,
	})
	if len(rows) != 1 {
		t.Fatalf("expected 1 delivery row, got %d", len(rows))
	}
	if rows[0].Status != "success" {
		t.Errorf("status = %q, want success", rows[0].Status)
	}
	if rows[0].Attempts != 1 {
		t.Errorf("attempts = %d, want 1", rows[0].Attempts)
	}
	_ = subID
}

func TestDelivery_5xxSchedulesRetry(t *testing.T) {
	sqlDB, q, siteID := newWebhookTestStack(t)
	_ = sqlDB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	seedSubscription(t, q, siteID, srv.URL, "s", []string{})
	NewEmitter(q).Emit(context.Background(), siteID, EventPageCreated, nil)

	mgr := NewDeliveryManager(q)
	mgr.RunOnce(context.Background())

	rows, _ := q.ListWebhookDeliveriesBySite(context.Background(), store.ListWebhookDeliveriesBySiteParams{
		SiteID: siteID, Limit: 10,
	})
	if len(rows) != 1 {
		t.Fatalf("expected 1 delivery row, got %d", len(rows))
	}
	if rows[0].Status != "retrying" {
		t.Errorf("status = %q, want retrying", rows[0].Status)
	}
	if rows[0].Attempts != 1 {
		t.Errorf("attempts = %d, want 1", rows[0].Attempts)
	}
	if rows[0].LastResponseStatus != 503 {
		t.Errorf("last_response_status = %d, want 503", rows[0].LastResponseStatus)
	}
}

func TestDelivery_DropsAfterMaxAttempts(t *testing.T) {
	sqlDB, q, siteID := newWebhookTestStack(t)
	_ = sqlDB
	var attempts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	seedSubscription(t, q, siteID, srv.URL, "s", []string{})
	NewEmitter(q).Emit(context.Background(), siteID, EventPageCreated, nil)

	mgr := NewDeliveryManager(q)
	for i := 0; i < MaxAttempts+2; i++ {
		// Fast-forward each row's next_attempt_at so the worker picks
		// it up on every loop iteration.
		past := time.Now().UTC().Add(-time.Hour).Format("2006-01-02 15:04:05")
		if _, err := sqlDB.Exec(`UPDATE webhook_deliveries SET next_attempt_at = ?`, past); err != nil {
			t.Fatal(err)
		}
		mgr.RunOnce(context.Background())
	}

	rows, _ := q.ListWebhookDeliveriesBySite(context.Background(), store.ListWebhookDeliveriesBySiteParams{
		SiteID: siteID, Limit: 10,
	})
	if rows[0].Status != "dropped" {
		t.Errorf("status = %q, want dropped after MaxAttempts", rows[0].Status)
	}
	if rows[0].Attempts != int64(MaxAttempts) {
		t.Errorf("attempts = %d, want %d", rows[0].Attempts, MaxAttempts)
	}
}

func TestDelivery_DeletedSubscriptionMarksDropped(t *testing.T) {
	// If the subscription is removed between Emit and the worker
	// pickup, the delivery row may still exist (FK cascade behavior
	// depends on whether PRAGMA foreign_keys is on). Either way,
	// the worker must NOT loop forever: when GetWebhookSubscription
	// returns ErrNoRows on pickup, the delivery is marked 'dropped'.
	sqlDB, q, siteID := newWebhookTestStack(t)
	_ = sqlDB
	id := seedSubscription(t, q, siteID, "https://x.example.com", "s", []string{})
	NewEmitter(q).Emit(context.Background(), siteID, EventPageCreated, nil)
	if err := q.DeleteWebhookSubscription(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	// If FK cascade fired, the delivery is gone and that's fine
	// (nothing to deliver). Otherwise the worker should mark it
	// dropped on pickup.
	mgr := NewDeliveryManager(q)
	mgr.RunOnce(context.Background())

	rows, _ := q.ListWebhookDeliveriesBySite(context.Background(), store.ListWebhookDeliveriesBySiteParams{
		SiteID: siteID, Limit: 10,
	})
	for _, row := range rows {
		if row.Status != "dropped" {
			t.Errorf("orphan delivery %s status = %q, want dropped", row.ID, row.Status)
		}
	}
}

func TestDelivery_OnlyPullsDueRows(t *testing.T) {
	// A delivery scheduled for the future MUST NOT be picked up on
	// this tick. Otherwise backoff means nothing.
	sqlDB, q, siteID := newWebhookTestStack(t)
	_ = sqlDB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	seedSubscription(t, q, siteID, srv.URL, "s", []string{})
	NewEmitter(q).Emit(context.Background(), siteID, EventPageCreated, nil)

	// Push next_attempt_at into the future.
	future := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := sqlDB.Exec(`UPDATE webhook_deliveries SET next_attempt_at = ?`, future); err != nil {
		t.Fatal(err)
	}

	mgr := NewDeliveryManager(q)
	mgr.RunOnce(context.Background())

	rows, _ := q.ListWebhookDeliveriesBySite(context.Background(), store.ListWebhookDeliveriesBySiteParams{
		SiteID: siteID, Limit: 10,
	})
	if rows[0].Status != "pending" {
		t.Errorf("future-scheduled row picked up; status = %q (want pending)", rows[0].Status)
	}
}
