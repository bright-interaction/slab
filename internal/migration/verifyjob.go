// Package migration - verifyjob.go (Sprint 4, 2026-05-06).
//
// JobManager owns the async lifecycle for verify-live runs. The migration
// handler hands off a URL set + deployed domain via Enqueue, gets back a
// job id, and the worker goroutine runs the same migration.VerifyLive
// crawl that the synchronous handler used to drive directly. Per-URL
// rows still land in migration_verifications; the new verify_jobs row
// aggregates total + processed + ok/fail counts so the UI can render a
// progress bar and a cancel button while the crawl runs for minutes.
//
// Lifecycle mirrors retention.Manager: Start spawns a single reaper
// goroutine that on boot marks any 'queued' / 'running' rows as
// 'failed' with error="server restart" (a server crash mid-crawl can't
// leave a row stuck running forever). Stop cancels the manager ctx and
// every in-flight worker, then waits bounded for them to exit.
//
// Cancellation: Cancel(jobID) loads the per-job context.CancelFunc from
// the cancels map and invokes it. The worker observes via the ctx
// passed into migration.VerifyLive, which propagates through the
// errgroup into each per-URL probe. Once cancelled the worker writes
// status='cancelled' and clears the cancels entry.

package migration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bright-interaction/slab/internal/store"
)

// progressFlushEveryN flushes the verify_jobs counters every N processed
// URLs. 50 is a balance between DB write pressure on a 25k-URL crawl
// (one write every ~10s at 4-concurrency * 200ms politeness) and the
// frontend's perceived cadence (the polling store reads every 2s).
const progressFlushEveryN = 50

// progressFlushEveryDur ensures we flush at least once per 5s even if
// the crawl is slow enough that 50 URLs hasn't completed yet. This is
// what keeps the UI moving when, say, a host is timing out repeatedly
// at 15s/probe.
const progressFlushEveryDur = 5 * time.Second

// JobManager runs verify-live crawls in the background. One process-wide
// instance is constructed at boot in cmd/server/main.go and injected
// into MigrationHandler via SetVerifyJobManager.
type JobManager struct {
	queries    *store.Queries
	verifyOpts VerifyOptions

	cancels sync.Map // jobID -> context.CancelFunc

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	wg      sync.WaitGroup // tracks in-flight worker goroutines
	started bool
}

// NewJobManager wires the manager. verifyOpts is the same options shape
// the synchronous VerifyLive used; production sets AllowPrivate=false
// so the SSRF guard blocks loopback. Tests inject AllowPrivate=true via
// SetVerifyOptionsForTest so httptest.Server stand-ins work.
func NewJobManager(queries *store.Queries) *JobManager {
	return &JobManager{queries: queries}
}

// SetVerifyOptionsForTest is the test seam. Mirrors the existing
// MigrationHandler.SetVerifyOptionsForTest method - production never
// calls it.
func (m *JobManager) SetVerifyOptionsForTest(opts VerifyOptions) {
	m.verifyOpts = opts
}

// Start runs the boot reaper and blocks until ctx is cancelled. The
// reaper marks any rows left in 'queued' or 'running' state on boot as
// 'failed' with reason="server restart" so a crashed run can't leave a
// job stuck. After the reap, Start does nothing else; in-flight jobs
// are tracked via wg.
func (m *JobManager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})
	m.mu.Unlock()

	go func() {
		defer close(m.done)
		// Boot reap: any row still 'queued' / 'running' is from a
		// previous process that crashed. Mark as failed so the UI
		// shows a clean terminal state and the operator can retry.
		if err := m.queries.ReapStaleVerifyJobs(ctx); err != nil {
			slog.Warn("verify_jobs boot reap", "err", err)
		}
		<-ctx.Done()
	}()
}

// Stop cancels the manager ctx (every in-flight worker observes the
// cancellation through its derived ctx) and waits up to ctx's deadline
// for the workers to exit. Safe to call when Start was never invoked.
func (m *JobManager) Stop(ctx context.Context) {
	m.mu.Lock()
	cancel := m.cancel
	done := m.done
	m.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	// Cancel every in-flight worker.
	m.cancels.Range(func(_, v any) bool {
		if cf, ok := v.(context.CancelFunc); ok {
			cf()
		}
		return true
	})
	// Wait for the reaper goroutine + workers, bounded by ctx.
	workersDone := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(workersDone)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
	select {
	case <-workersDone:
	case <-ctx.Done():
	}
}

// Enqueue registers a new verify-live job, writes the verify_jobs row
// with status='queued', spawns a worker goroutine, and returns the job
// id. The worker uses a fresh context.Background() derivation so the
// crawl outlives the HTTP request that triggered it. Cancellation is
// orchestrated via the cancels map: jobID -> context.CancelFunc.
func (m *JobManager) Enqueue(ctx context.Context, siteID, migrationID string, urls []string, deployedDomain string) (string, error) {
	if siteID == "" || migrationID == "" {
		return "", errors.New("siteID and migrationID are required")
	}
	if len(urls) == 0 {
		return "", errors.New("urls is required")
	}
	if deployedDomain == "" {
		return "", errors.New("deployedDomain is required")
	}
	jobID := newVerifyJobID()
	if err := m.queries.CreateVerifyJob(ctx, store.CreateVerifyJobParams{
		ID:             jobID,
		SiteID:         siteID,
		MigrationID:    migrationID,
		TotalUrls:      int64(len(urls)),
		DeployedDomain: deployedDomain,
	}); err != nil {
		return "", err
	}

	workerCtx, cancel := context.WithCancel(context.Background())
	m.cancels.Store(jobID, cancel)
	m.wg.Add(1)

	urlsCopy := make([]string, len(urls))
	copy(urlsCopy, urls)

	go m.runWorker(workerCtx, cancel, jobID, siteID, migrationID, urlsCopy, deployedDomain)
	return jobID, nil
}

// Cancel signals the worker to stop. Returns false if the job is unknown
// or already terminal (the cancels entry is removed when the worker
// finishes, so a missing entry means the job already settled).
func (m *JobManager) Cancel(jobID string) bool {
	v, ok := m.cancels.Load(jobID)
	if !ok {
		return false
	}
	if cf, ok := v.(context.CancelFunc); ok {
		cf()
		return true
	}
	return false
}

// runWorker is the background goroutine that owns one verify-live run.
// It marks the row 'running', drives migration.VerifyLive with an
// OnResult callback that writes per-URL rows + bumps the in-memory
// counters, flushes counters to the verify_jobs row periodically, and
// finally writes a terminal status (done | cancelled | failed).
//
// Context discipline: in-flight DB writes use the worker ctx so that a
// user-initiated Cancel halts further writes once observed. The
// terminal FinishVerifyJob write uses a separate context.Background()-
// derived ctx with a short timeout so cancellation can't prevent the
// final state row from landing (otherwise a cancelled job would stay
// 'running' until the next boot's reaper marks it stale).
func (m *JobManager) runWorker(ctx context.Context, cancel context.CancelFunc, jobID, siteID, migrationID string, urls []string, deployedDomain string) {
	defer m.wg.Done()
	defer m.cancels.Delete(jobID)
	defer cancel()

	if err := m.queries.MarkVerifyJobRunning(ctx, jobID); err != nil {
		slog.Warn("verify_job mark running", "job", jobID, "err", err)
	}

	var processed, okCount, failCount atomic.Int64
	lastFlush := time.Now()
	var flushMu sync.Mutex

	flush := func(force bool) {
		flushMu.Lock()
		defer flushMu.Unlock()
		now := time.Now()
		p := processed.Load()
		if !force && p%progressFlushEveryN != 0 && now.Sub(lastFlush) < progressFlushEveryDur {
			return
		}
		lastFlush = now
		_ = m.queries.UpdateVerifyJobProgress(ctx, store.UpdateVerifyJobProgressParams{
			ID:            jobID,
			ProcessedUrls: p,
			OkCount:       okCount.Load(),
			FailCount:     failCount.Load(),
		})
	}

	opts := m.verifyOpts
	opts.OnResult = func(r VerifyResult) {
		ok := int64(0)
		if r.OK {
			ok = 1
			okCount.Add(1)
		} else {
			failCount.Add(1)
		}
		_ = m.queries.CreateMigrationVerification(ctx, store.CreateMigrationVerificationParams{
			ID:          newVerifyJobID(),
			SiteID:      siteID,
			MigrationID: migrationID,
			SourceUrl:   r.SourceURL,
			StatusCode:  int64(r.StatusCode),
			FinalUrl:    r.FinalURL,
			Hops:        int64(r.Hops),
			Ok:          ok,
			Error:       r.Error,
		})
		processed.Add(1)
		flush(false)
	}

	_, runErr := VerifyLive(ctx, urls, deployedDomain, opts)

	// Resolve terminal status. ctx.Err() takes precedence: a cancelled
	// crawl may have non-nil runErr because errgroup propagates the
	// context cancellation to each probe.
	status := "done"
	errStr := ""
	if ctx.Err() != nil {
		status = "cancelled"
	} else if runErr != nil {
		status = "failed"
		errStr = runErr.Error()
	}

	finishCtx, finishCancel := context.WithTimeout(context.Background(), terminalWriteTimeout)
	defer finishCancel()
	if err := m.queries.FinishVerifyJob(finishCtx, store.FinishVerifyJobParams{
		ID:            jobID,
		Status:        status,
		ProcessedUrls: processed.Load(),
		OkCount:       okCount.Load(),
		FailCount:     failCount.Load(),
		Error:         errStr,
	}); err != nil {
		slog.Warn("verify_job finish", "job", jobID, "err", err)
	}
	slog.Info("verify_job complete",
		"job", jobID, "status", status,
		"processed", processed.Load(),
		"ok", okCount.Load(), "fail", failCount.Load())
}

// terminalWriteTimeout bounds the FinishVerifyJob write so a hung DB
// can't keep a worker goroutine alive forever after cancellation.
const terminalWriteTimeout = 10 * time.Second

// newVerifyJobID returns a 24-char hex id, same shape as newPorterID
// (kept local so verifyjob doesn't depend on the porter).
func newVerifyJobID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
