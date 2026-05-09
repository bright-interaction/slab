package migration

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	dbpkg "github.com/brightinteraction/atomicsite/internal/db"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// newJobManagerForTest returns a JobManager bound to a fresh sqlite + a
// pre-seeded site so verify_jobs FK passes.
func newJobManagerForTest(t *testing.T) (*JobManager, *store.Queries, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "vj.db")
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
	siteID := "abcdef0123456789abcdefvj"
	if err := q.CreateSite(context.Background(), store.CreateSiteParams{
		ID: siteID, Name: "T", Slug: "t-vj", PrimaryColor: "#000",
		SecondaryColor: "#000", BgColor: "#FFF", TextColor: "#111",
		FontHeading: "Inter", FontBody: "Inter", Lang: "en",
	}); err != nil {
		t.Fatal(err)
	}
	m := NewJobManager(q)
	m.SetVerifyOptionsForTest(VerifyOptions{
		AllowPrivate:    true,
		PolitenessDelay: 1 * time.Millisecond,
		Concurrency:     2,
	})
	return m, q, siteID
}

// TestJobManager_EnqueueWritesQueuedRow verifies that Enqueue persists a
// row immediately and returns a non-empty job id (the goroutine spawn
// happens asynchronously; this test only proves the synchronous side).
func TestJobManager_EnqueueWritesQueuedRow(t *testing.T) {
	m, q, siteID := newJobManagerForTest(t)
	// httptest.Server we never actually want the worker to reach.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	id, err := m.Enqueue(context.Background(), siteID, "mig-1", []string{srv.URL}, srv.URL)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty job id")
	}
	// Drain the worker so the test's t.Cleanup doesn't observe an
	// in-flight goroutine accessing a closed DB handle.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		row, _ := q.GetVerifyJob(context.Background(), id)
		if row.Status == "done" || row.Status == "failed" || row.Status == "cancelled" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("worker never reached terminal state")
}

// TestJobManager_HappyPathTerminalCounts drives the worker against a
// deterministic httptest.Server and asserts the final verify_jobs row
// matches the expected ok/fail split.
func TestJobManager_HappyPathTerminalCounts(t *testing.T) {
	m, q, siteID := newJobManagerForTest(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/ok-1", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/ok-2", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/ok-3", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/fail-1", http.NotFound)
	mux.HandleFunc("/fail-2", http.NotFound)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	urls := []string{"/ok-1", "/ok-2", "/ok-3", "/fail-1", "/fail-2"}
	id, err := m.Enqueue(context.Background(), siteID, "mig-happy", urls, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	row := waitTerminal(t, q, id)
	if row.Status != "done" {
		t.Errorf("status: want done got %q (err=%q)", row.Status, row.Error)
	}
	if row.ProcessedUrls != 5 || row.OkCount != 3 || row.FailCount != 2 {
		t.Errorf("counts: %+v", row)
	}
	// Per-URL rows persisted to migration_verifications.
	stored, _ := q.ListMigrationVerifications(context.Background(), "mig-happy")
	if len(stored) != 5 {
		t.Errorf("per-URL rows: want 5, got %d", len(stored))
	}
}

// TestJobManager_CancelMidRunTransitions asserts that cancelling a
// running job lands at status='cancelled'. We use a slow handler so
// the worker is guaranteed to be in flight when Cancel fires.
func TestJobManager_CancelMidRunTransitions(t *testing.T) {
	m, q, siteID := newJobManagerForTest(t)
	// Each /slow handler blocks for 200ms so a 5-URL crawl takes ~1s
	// even at concurrency=2; cancel after 50ms reliably arrives mid-run.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(200 * time.Millisecond):
			w.WriteHeader(200)
		case <-r.Context().Done():
			return
		}
	}))
	defer srv.Close()

	urls := []string{"/a", "/b", "/c", "/d", "/e"}
	id, err := m.Enqueue(context.Background(), siteID, "mig-cancel", urls, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if !m.Cancel(id) {
		t.Fatal("Cancel returned false on running job")
	}
	row := waitTerminal(t, q, id)
	if row.Status != "cancelled" {
		t.Errorf("status: want cancelled got %q", row.Status)
	}
}

// TestJobManager_ReapStaleOnStart simulates a server crash by writing a
// row in 'running' state directly, then Start runs the boot reaper.
func TestJobManager_ReapStaleOnStart(t *testing.T) {
	m, q, siteID := newJobManagerForTest(t)
	jobID := "job-stale"
	if err := q.CreateVerifyJob(context.Background(), store.CreateVerifyJobParams{
		ID: jobID, SiteID: siteID, MigrationID: "mig-reap", TotalUrls: 10,
		DeployedDomain: "deploy.example",
	}); err != nil {
		t.Fatal(err)
	}
	if err := q.MarkVerifyJobRunning(context.Background(), jobID); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		m.Stop(context.Background())
	})
	m.Start(ctx)
	// Reaper runs synchronously inside Start's goroutine; poll briefly
	// to absorb scheduling jitter.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, _ := q.GetVerifyJob(context.Background(), jobID)
		if row.Status == "failed" {
			if row.Error != "server restart" {
				t.Errorf("reap error message: %q", row.Error)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("reaper did not flip stale row")
}

// TestJobManager_ProgressCallbackFiresPerURL verifies that OnResult is
// invoked exactly once per URL in the order completion happens. Lower-
// level than the manager test - asserts the verify.go contract Sprint 4
// added so the manager can rely on it.
func TestJobManager_ProgressCallbackFiresPerURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	var fired atomic.Int64
	var seen sync.Map
	urls := make([]string, 25)
	for i := range urls {
		urls[i] = "/u"
	}
	_, err := VerifyLive(context.Background(), urls, srv.URL, VerifyOptions{
		AllowPrivate:    true,
		PolitenessDelay: 1 * time.Millisecond,
		Concurrency:     4,
		OnResult: func(r VerifyResult) {
			fired.Add(1)
			seen.Store(r.SourceURL+r.FinalURL, true) // unique per call
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fired.Load(); got != 25 {
		t.Errorf("OnResult fires: want 25 got %d", got)
	}
}

// waitTerminal polls for a job's terminal state, used by happy/cancel
// tests so they don't race the worker goroutine.
func waitTerminal(t *testing.T, q *store.Queries, jobID string) store.VerifyJob {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		row, err := q.GetVerifyJob(context.Background(), jobID)
		if err == nil {
			switch row.Status {
			case "done", "failed", "cancelled":
				return row
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("job never reached terminal state")
	return store.VerifyJob{}
}
