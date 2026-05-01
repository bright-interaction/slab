// Package retention runs a daily background sweep that purges old analytics
// and consent rows per site. Without this the SQLite DB grows unbounded:
// at moderate traffic (10 sites, 1k visitors/day, 5 pageviews/visit) the
// visit_events table alone gains ~7 GB/year. The sweep keeps the DB lean
// without needing an external cron, by running inside the atomicsite Go
// binary on a 24h ticker.
//
// Per-site retention is configurable via three settings (general category):
//
//	general.analytics_retention_days   default 180  range 30..3650
//	general.consent_retention_days     default 730  range 30..3650
//	general.engagement_retention_days  default  90  range  7..3650
//
// Why these defaults:
//   - Analytics 180d: half a year of raw event detail is enough for trend
//     reports + ad-hoc forensics. Aggregates (the planned cookie-analytics
//     rollup tables) survive the purge.
//   - Consent 730d: GDPR audit horizon. EU regulators expect proof-of-consent
//     to be retrievable for the full validity window of the consent itself
//     (which CookieProof's widget also defaults to 365 days), plus a buffer.
//   - Engagement 90d: high row volume, low forensic value past a quarter.
//
// Identified visit_sessions (those with a non-empty identified_at) are NOT
// purged regardless of last_seen_at, since they represent CRM-confirmed
// contacts the dashboard's identified-visitor view depends on.
package retention

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bright-interaction/slab/internal/store"
)

// DefaultAnalyticsDays is the per-site fallback when general.analytics_retention_days
// isn't set. 180 days = two quarters of raw events, enough for ad-hoc
// forensics; aggregates persist beyond this window.
const DefaultAnalyticsDays = 180

// DefaultConsentDays is the per-site fallback when general.consent_retention_days
// isn't set. 730 days = 2 years, the standard GDPR proof-of-consent horizon.
const DefaultConsentDays = 730

// DefaultEngagementDays is the per-site fallback when general.engagement_retention_days
// isn't set. 90 days = one quarter of high-volume engagement metrics.
const DefaultEngagementDays = 90

// MinRetentionDays floor. Anything below this is treated as the default;
// retention shorter than a week breaks any meaningful weekly reporting and
// is almost certainly a misconfiguration.
const MinRetentionDays = 7

// MaxRetentionDays ceiling: 10 years. Going past this is rare and likely
// a typo (someone wrote 36500 thinking days, meaning years).
const MaxRetentionDays = 3650

// SaltRetentionDays is the fixed window for consent_salts. Salts older than
// this can be deleted because no live consent_records still hashed against
// them remain after the consent purge runs (which itself uses
// DefaultConsentDays >= 30).
const SaltRetentionDays = 30

// PauseAfterStartup gives the rest of the process (HTTP server, schema
// migrations, analytics manager) breathing room before the first sweep.
// SQLite WAL checkpointing also benefits from not being slammed during boot.
const PauseAfterStartup = 60 * time.Second

// SweepInterval is how often the manager wakes up to purge. Daily matches
// the consent_salts daily-rotation cadence and limits the worst-case
// staleness of "purged within the last X" to 24 hours.
const SweepInterval = 24 * time.Hour

// Manager runs the periodic purge loop. One process owns one Manager.
type Manager struct {
	queries *store.Queries
	db      *sql.DB

	mu         sync.Mutex
	lastSweep  time.Time
	lastResult SweepResult
	// dataDir is the workspace root used by the orphan-workspace sweep
	// (audit H8). Empty means the sweep is disabled (e.g. in tests). Set
	// via SetDataDir before Start.
	dataDir string

	cancel context.CancelFunc
	done   chan struct{}
}

// SweepResult is the per-run summary the slog line + admin debug endpoint
// expose. Counts are aggregate across all sites; PerSite holds the breakdown.
type SweepResult struct {
	StartedAt    time.Time            `json:"started_at"`
	FinishedAt   time.Time            `json:"finished_at"`
	DurationMs   int64                `json:"duration_ms"`
	SitesScanned int                  `json:"sites_scanned"`
	VisitEvents  int64                `json:"visit_events_deleted"`
	Engagement   int64                `json:"engagement_deleted"`
	Sessions     int64                `json:"sessions_deleted"`
	Consent      int64                `json:"consent_deleted"`
	Salts        int64                `json:"salts_deleted"`
	PerSite      []PerSiteSweepResult `json:"per_site,omitempty"`
	Errors       []string             `json:"errors,omitempty"`
}

// PerSiteSweepResult is the per-tenant breakdown inside a SweepResult. Useful
// for a "your site dropped N rows last night" admin dashboard line.
type PerSiteSweepResult struct {
	SiteID                 string `json:"site_id"`
	AnalyticsDays          int    `json:"analytics_days"`
	ConsentDays            int    `json:"consent_days"`
	EngagementDays         int    `json:"engagement_days"`
	VisitEventsDeleted     int64  `json:"visit_events_deleted"`
	EngagementDeleted      int64  `json:"engagement_deleted"`
	SessionsDeleted        int64  `json:"sessions_deleted"`
	ConsentDeleted         int64  `json:"consent_deleted"`
}

// NewManager builds a Manager. Pass the same *sql.DB and *store.Queries the
// HTTP server uses; the manager runs scoped DELETEs against the shared
// connection (SQLite WAL mode handles concurrent readers + the single
// writer just fine).
//
// dataDir is the root that contains the workspaces/ subdir. If non-empty,
// the daily sweep also reaps orphan workspace directories (audit H8): a
// workspaces/{siteID}/ dir whose siteID is not in the sites table gets
// removed. Pass "" to disable the orphan sweep (e.g. in tests).
func NewManager(queries *store.Queries, db *sql.DB) *Manager {
	return &Manager{
		queries: queries,
		db:      db,
	}
}

// SetDataDir configures the data directory root. Call before Start. When
// set, the daily sweep also reaps orphan workspace directories (audit H8).
func (m *Manager) SetDataDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dataDir = dir
}

// Start launches the sweep loop in a goroutine. The first sweep fires
// PauseAfterStartup after Start returns, then every SweepInterval. Cancel
// the parent ctx to stop, or call Stop for a bounded-wait shutdown.
func (m *Manager) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})

	go func() {
		defer close(m.done)
		// Initial delay so we don't slam the DB on boot. Cancellable via ctx.
		select {
		case <-time.After(PauseAfterStartup):
		case <-ctx.Done():
			return
		}

		m.runOnce(ctx)
		ticker := time.NewTicker(SweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.runOnce(ctx)
			}
		}
	}()
}

// Stop cancels the loop and waits up to ctx's deadline for the current
// sweep to finish. Safe to call when Start was never invoked.
func (m *Manager) Stop(ctx context.Context) {
	if m.cancel == nil {
		return
	}
	m.cancel()
	if m.done == nil {
		return
	}
	select {
	case <-m.done:
	case <-ctx.Done():
	}
}

// LastResult returns the most recent SweepResult (zero value if no sweep
// has completed yet). Read-only snapshot under the manager's mutex.
func (m *Manager) LastResult() SweepResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastResult
}

// RunOnce triggers an out-of-band sweep, intended for an admin debug
// endpoint. Returns the SweepResult so the caller can render it in the
// response. Errors landing inside individual site purges are logged and
// surfaced in SweepResult.Errors; the function itself only errors when it
// can't list the sites at all.
func (m *Manager) RunOnce(ctx context.Context) (SweepResult, error) {
	return m.runOnce(ctx)
}

func (m *Manager) runOnce(ctx context.Context) (SweepResult, error) {
	res := SweepResult{StartedAt: time.Now().UTC()}

	sites, err := m.queries.ListSites(ctx)
	if err != nil {
		res.Errors = append(res.Errors, "list sites: "+err.Error())
		res.FinishedAt = time.Now().UTC()
		res.DurationMs = res.FinishedAt.Sub(res.StartedAt).Milliseconds()
		m.recordResult(res)
		return res, err
	}
	res.SitesScanned = len(sites)

	for _, site := range sites {
		if ctx.Err() != nil {
			res.Errors = append(res.Errors, "ctx cancelled mid-sweep")
			break
		}
		ps := m.purgeSite(ctx, site.ID)
		res.PerSite = append(res.PerSite, ps)
		res.VisitEvents += ps.VisitEventsDeleted
		res.Engagement += ps.EngagementDeleted
		res.Sessions += ps.SessionsDeleted
		res.Consent += ps.ConsentDeleted
	}

	// Salt purge is process-level, not per-site. Salts older than 30 days
	// are unreferenced once the consent purge runs (which uses >= 30d
	// retention) and there's no per-tenant story for them.
	// Audit H8: orphan workspace sweep. Looks at every directory under
	// {dataDir}/workspaces/ and removes any whose name is no longer in
	// the sites table. Disabled when dataDir is unset (tests).
	if dir := m.getDataDir(); dir != "" {
		if n, err := m.sweepOrphanWorkspaces(ctx, dir); err != nil {
			res.Errors = append(res.Errors, "orphan workspace sweep: "+err.Error())
		} else if n > 0 {
			slog.Info("retention: orphan workspaces removed", "count", n)
		}
	}

	saltCutoff := time.Now().UTC().AddDate(0, 0, -SaltRetentionDays).Format("2006-01-02")
	if err := m.queries.DeleteConsentSaltsOlderThan(ctx, saltCutoff); err != nil {
		res.Errors = append(res.Errors, "delete consent salts: "+err.Error())
	}

	res.FinishedAt = time.Now().UTC()
	res.DurationMs = res.FinishedAt.Sub(res.StartedAt).Milliseconds()

	slog.Info("retention: sweep complete",
		"sites", res.SitesScanned,
		"visit_events", res.VisitEvents,
		"engagement", res.Engagement,
		"sessions", res.Sessions,
		"consent", res.Consent,
		"duration_ms", res.DurationMs,
		"errors", len(res.Errors),
	)

	m.recordResult(res)
	return res, nil
}

func (m *Manager) recordResult(r SweepResult) {
	m.mu.Lock()
	m.lastSweep = r.FinishedAt
	m.lastResult = r
	m.mu.Unlock()
}

// purgeSite resolves the site's retention windows from settings, then
// runs the four DELETEs. Errors land in slog and the returned struct so
// they don't abort the rest of the sweep.
func (m *Manager) purgeSite(ctx context.Context, siteID string) PerSiteSweepResult {
	settings, err := m.queries.ListSettingsBySite(ctx, siteID)
	if err != nil {
		slog.Warn("retention: list settings", "site_id", siteID, "err", err)
		// fall through with defaults; one missing settings read shouldn't
		// stop the purge for the rest of the site.
	}
	settingsMap := map[string]string{}
	for _, s := range settings {
		settingsMap[s.Category+"."+s.Key] = s.Value
	}

	analyticsDays := readDays(settingsMap, "general.analytics_retention_days", DefaultAnalyticsDays)
	consentDays := readDays(settingsMap, "general.consent_retention_days", DefaultConsentDays)
	engagementDays := readDays(settingsMap, "general.engagement_retention_days", DefaultEngagementDays)

	now := time.Now().UTC()
	tsCutoff := now.AddDate(0, 0, -analyticsDays).Format(time.RFC3339)
	engagementCutoff := now.AddDate(0, 0, -engagementDays).Format(time.RFC3339)
	consentCutoffMs := now.AddDate(0, 0, -consentDays).UnixMilli()

	res := PerSiteSweepResult{
		SiteID:         siteID,
		AnalyticsDays:  analyticsDays,
		ConsentDays:    consentDays,
		EngagementDays: engagementDays,
	}

	// Order: engagement -> events -> sessions. Sessions last so any future
	// FK or analytics aggregator that joins events->sessions still has the
	// session row when events are dropped.
	if n, err := m.queries.DeleteVisitEngagementBySiteOlderThan(ctx, store.DeleteVisitEngagementBySiteOlderThanParams{
		SiteID: siteID, Ts: engagementCutoff,
	}); err != nil {
		slog.Warn("retention: delete engagement", "site_id", siteID, "err", err)
	} else {
		res.EngagementDeleted = n
	}

	if n, err := m.queries.DeleteVisitEventsBySiteOlderThan(ctx, store.DeleteVisitEventsBySiteOlderThanParams{
		SiteID: siteID, Ts: tsCutoff,
	}); err != nil {
		slog.Warn("retention: delete visit events", "site_id", siteID, "err", err)
	} else {
		res.VisitEventsDeleted = n
	}

	if n, err := m.queries.DeleteVisitSessionsBySiteOlderThan(ctx, store.DeleteVisitSessionsBySiteOlderThanParams{
		SiteID: siteID, LastSeenAt: tsCutoff,
	}); err != nil {
		slog.Warn("retention: delete sessions", "site_id", siteID, "err", err)
	} else {
		res.SessionsDeleted = n
	}

	if n, err := m.queries.DeleteConsentBySiteOlderThan(ctx, store.DeleteConsentBySiteOlderThanParams{
		SiteID: siteID, CreatedAt: consentCutoffMs,
	}); err != nil {
		slog.Warn("retention: delete consent", "site_id", siteID, "err", err)
	} else {
		res.ConsentDeleted = n
	}

	return res
}

// readDays parses an int settings value and clamps to [MinRetentionDays,
// MaxRetentionDays]. Falls back to fallbackDays for empty / unparseable /
// out-of-range values.
func readDays(m map[string]string, key string, fallbackDays int) int {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return fallbackDays
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallbackDays
	}
	if n < MinRetentionDays || n > MaxRetentionDays {
		return fallbackDays
	}
	return n
}

// ErrNotStarted is returned by RunOnce when called before Start. Reserved
// for future use; current implementation is permissive and runs even
// without Start since the manager is otherwise idempotent.
var ErrNotStarted = errors.New("retention: manager not started")

// getDataDir returns the configured data dir under the manager mutex.
func (m *Manager) getDataDir() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dataDir
}

// sweepOrphanWorkspaces removes any directory under {dataDir}/workspaces/
// whose name is no longer in the sites table (audit H8). Returns the
// count of removed dirs. Errors do not abort the sweep — every reapable
// dir is given a chance.
//
// Safety: only paths matching the 24-hex siteID shape are considered. Any
// other directory name (e.g. user-created scratch dir) is left alone.
// The directory traversal stays inside `{dataDir}/workspaces/` so a
// symlink doesn't escape the workspaces root.
func (m *Manager) sweepOrphanWorkspaces(ctx context.Context, dataDir string) (int, error) {
	wsRoot := filepath.Join(dataDir, "workspaces")
	entries, err := os.ReadDir(wsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // No workspaces dir yet; nothing to do.
		}
		return 0, err
	}

	// Build the set of live site IDs.
	sites, err := m.queries.ListSites(ctx)
	if err != nil {
		return 0, err
	}
	live := make(map[string]bool, len(sites))
	for _, s := range sites {
		live[s.ID] = true
	}

	removed := 0
	for _, entry := range entries {
		if ctx.Err() != nil {
			return removed, ctx.Err()
		}
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !workspaceNamePattern.MatchString(name) {
			// Doesn't look like a siteID; leave it alone (defensive).
			continue
		}
		if live[name] {
			continue
		}
		path := filepath.Join(wsRoot, name)
		if err := os.RemoveAll(path); err != nil {
			slog.Warn("retention: orphan workspace remove failed",
				"path", path, "err", err)
			continue
		}
		removed++
	}
	return removed, nil
}

// workspaceNamePattern matches the 24-char hex shape that newID() produces
// for siteIDs. Anything else under workspaces/ is left untouched by the
// orphan sweep so a stray scratch dir doesn't get nuked.
var workspaceNamePattern = regexp.MustCompile(`^[a-fA-F0-9]{24}$`)
