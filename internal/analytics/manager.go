package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync"

	"github.com/bright-interaction/atomicsite/internal/builder"
	"github.com/bright-interaction/atomicsite/internal/store"
)

// Manager owns one Parser goroutine per site with analytics enabled.
// Lifecycle: Start on server boot, Reload when site_settings change,
// Stop on graceful shutdown.
type Manager struct {
	queries  *store.Queries
	baseSalt string

	mu      sync.Mutex
	parsers map[string]*runningParser // key = site_id
	wg      sync.WaitGroup
}

// trackedSite is one row in the desired-set returned by desiredSites:
// site_id stays the FK on visit_events, slug drives the log file path.
type trackedSite struct {
	ID   string
	Slug string
}

type runningParser struct {
	parser *Parser
	cancel context.CancelFunc
}

// NewManager constructs a Manager. baseSalt is the global secret from
// AnalyticsSalt env var (or auto-generated on first run); per-site salts are
// derived from baseSalt + site_id so the on-disk salt rotation is one operation.
func NewManager(queries *store.Queries, baseSalt string) *Manager {
	return &Manager{
		queries:  queries,
		baseSalt: baseSalt,
		parsers:  make(map[string]*runningParser),
	}
}

// Start scans all sites, finds those with analytics enabled, and spawns a
// Parser per site. Idempotent.
func (m *Manager) Start(ctx context.Context) error {
	return m.Reload(ctx)
}

// Reload rescans site_settings and starts/stops parsers to match the new
// desired set. Sites that turn analytics off are stopped; new ones are started.
func (m *Manager) Reload(ctx context.Context) error {
	desired, err := m.desiredSites(ctx)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop parsers no longer wanted
	for siteID, rp := range m.parsers {
		if _, keep := desired[siteID]; !keep {
			rp.cancel()
			rp.parser.Stop()
			delete(m.parsers, siteID)
			slog.Info("analytics: stopped parser", "site_id", siteID)
		}
	}

	// Start parsers newly desired
	for siteID, ts := range desired {
		if _, running := m.parsers[siteID]; running {
			continue
		}
		m.spawn(ctx, ts)
	}
	return nil
}

// Stop cancels every running parser and waits for them to exit.
func (m *Manager) Stop(ctx context.Context) {
	m.mu.Lock()
	for siteID, rp := range m.parsers {
		rp.cancel()
		rp.parser.Stop()
		delete(m.parsers, siteID)
	}
	m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// spawn must be called with m.mu held.
func (m *Manager) spawn(ctx context.Context, ts trackedSite) {
	salt := m.siteSalt(ctx, ts.ID)
	p := NewParser(ts.ID, builder.NginxLogPath(ts.Slug), salt, m.queries)
	pCtx, cancel := context.WithCancel(ctx)
	m.parsers[ts.ID] = &runningParser{parser: p, cancel: cancel}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		p.Run(pCtx)
	}()
	slog.Info("analytics: started parser", "site_id", ts.ID, "slug", ts.Slug)
}

// desiredSites returns the set of sites where analytics tracking is enabled,
// keyed by site_id with the slug attached so the parser can derive the right
// log path. We only spawn parsers when atomicsite_tracking_enabled=true
// (cookieproof alone doesn't write to our log; it lives client-side).
func (m *Manager) desiredSites(ctx context.Context) (map[string]trackedSite, error) {
	sites, err := m.queries.ListSites(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]trackedSite)
	for _, s := range sites {
		settings, err := m.queries.ListSettingsByCategory(ctx, store.ListSettingsByCategoryParams{
			SiteID:   s.ID,
			Category: "analytics",
		})
		if err != nil {
			continue
		}
		for _, kv := range settings {
			if kv.Key == "atomicsite_tracking_enabled" && isTrue(kv.Value) {
				out[s.ID] = trackedSite{ID: s.ID, Slug: s.Slug}
				break
			}
		}
	}
	return out, nil
}

// siteSalt derives a per-site salt. If the site has analytics.salt set, use it.
// Otherwise compute baseSalt+site_id, persist, and return. Salt rotation is out
// of scope for v1: we never overwrite an existing salt.
func (m *Manager) siteSalt(ctx context.Context, siteID string) string {
	existing, err := m.queries.GetSetting(ctx, store.GetSettingParams{
		SiteID:   siteID,
		Category: "analytics",
		Key:      "salt",
	})
	if err == nil && existing.Value != "" {
		return existing.Value
	}

	base := m.baseSalt
	if base == "" {
		// No global salt configured: random per-site, persisted, never logged.
		base = newID()
	}
	h := sha256.Sum256([]byte(base + ":" + siteID))
	salt := hex.EncodeToString(h[:])

	_ = m.queries.UpsertSetting(ctx, store.UpsertSettingParams{
		ID:       newID(),
		SiteID:   siteID,
		Category: "analytics",
		Key:      "salt",
		Value:    salt,
	})
	return salt
}

func isTrue(v string) bool {
	switch v {
	case "1", "true", "yes", "on", "TRUE", "True":
		return true
	}
	return false
}
