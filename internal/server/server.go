// Package server sets up the chi router and wires all handlers.
package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/bright-interaction/slab/internal/analyticsdb"
	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/domains"
	"github.com/bright-interaction/slab/internal/handlers"
	"github.com/bright-interaction/slab/internal/mcp"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/retention"
	"github.com/bright-interaction/slab/internal/storage"
	"github.com/bright-interaction/slab/internal/store"
)

// FrontendFS is the embedded frontend dist. Populated by cmd/server/main.go.
var FrontendFS fs.FS

// Server holds all dependencies.
type Server struct {
	cfg     *config.Config
	db      *sql.DB
	queries *store.Queries
	storage storage.Store
	authMW  *authmw.AuthMiddleware
	agentMW *authmw.AgentAuthMiddleware
	// AnalyticsDB is the read-only DuckDB ATTACH on the SQLite file.
	// Optional: nil when DuckDB couldn't open at boot. Handlers that
	// depend on it return 503 in that case rather than crashing.
	AnalyticsDB *analyticsdb.Manager
	// RetentionMgr exposes the last sweep result via /api/admin/metrics
	// (audit I2). Optional; nil disables the retention metrics field.
	RetentionMgr *retention.Manager
	// OnAnalyticsSettingsChange, when set, is invoked after settings writes that
	// touch the analytics category so the analytics Manager can rescan parsers.
	OnAnalyticsSettingsChange func(context.Context)
	// DomainReconciler drives custom-hostname provisioning (DNS verify
	// → certbot → nginx vhost → live probe). Optional: nil keeps the
	// admin endpoints functional but no edge changes occur. Wired in
	// cmd/server/main.go after Server is constructed.
	DomainReconciler *domains.Reconciler
}

// New creates a Server.
func New(cfg *config.Config, db *sql.DB, queries *store.Queries, st storage.Store) *Server {
	if cfg.BrightCRMWebhookURL != "" && cfg.BrightCRMWebhookSecret != "" {
		slog.Info("brightcrm integration enabled (outbound + /t/inbound verify)", "url", cfg.BrightCRMWebhookURL)
	} else {
		slog.Warn("brightcrm integration DISABLED (BRIGHTCRM_WEBHOOK_URL or BRIGHTCRM_WEBHOOK_SECRET unset)")
	}
	return &Server{
		cfg:     cfg,
		db:      db,
		queries: queries,
		storage: st,
		authMW:  authmw.NewAuthMiddleware(cfg, queries),
		agentMW: authmw.NewAgentAuthMiddleware(queries),
	}
}

// Router builds and returns the chi router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(s.cfg))
	// Global body-size cap. Sets an upper bound on every request body
	// before any handler reads it, sized to cfg.MaxUploadSize so the
	// JSON-API surface (which never legitimately needs more than a
	// few MB) and the multipart-upload surface (which can hit the
	// configured ceiling) share the same limit. Without this, a
	// malicious client can POST gigabytes of JSON to /api/sites/...
	// and exhaust the goroutine's memory before json.Decode finishes.
	r.Use(jsonBodySizeMiddleware(s.cfg.MaxUploadSize))

	// Health (public). Audit M4: probes real component status instead of
	// returning a fixed `{status:ok}`. Kubernetes / Dockyard / external
	// monitors that hit this endpoint now see actionable green/yellow/red.
	//
	// Components probed (each is "ok" / "fail" / "unconfigured"):
	//   - sqlite: SELECT 1 against the OLTP DB (fails only when the file
	//     handle is broken; WAL contention surfaces here as slowness, not
	//     a false fail).
	//   - duckdb: only when AnalyticsDB is non-nil; `analyticsdb.Healthy`
	//     does a trivial query against the ATTACHed schema.
	//
	// The endpoint returns 200 when sqlite is ok regardless of duckdb so
	// load balancers don't take the whole service out for a degraded
	// analytics layer.
	r.Get("/api/health", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		dbOK := func() bool {
			row := s.db.QueryRowContext(ctx, "SELECT 1")
			var n int
			return row.Scan(&n) == nil
		}()
		analyticsOK := "unconfigured"
		if s.AnalyticsDB != nil {
			if s.AnalyticsDB.Healthy(ctx) {
				analyticsOK = "ok"
			} else {
				analyticsOK = "fail"
			}
		}
		status := "ok"
		httpStatus := http.StatusOK
		if !dbOK {
			status = "fail"
			httpStatus = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(httpStatus)
		body := `{"status":"` + status +
			`","sqlite":"` + boolToOKFail(dbOK) +
			`","analyticsdb":"` + analyticsOK + `"}`
		_, _ = w.Write([]byte(body))
	})

	// Public analytics receiver (/t/*). Mounted high in the router so it
	// can't be shadowed by the SPA fallback. No auth: built sites POST
	// here cross-origin (same-origin if behind same hostname). Fingerprint
	// middleware runs only on this group so admin requests don't pick up
	// visitor cookies.
	trackH := handlers.NewTrackHandler(s.cfg, s.queries, s.db)
	r.Group(func(r chi.Router) {
		r.Use(authmw.FingerprintMiddleware(s.cfg))
		r.Post("/t/consent", trackH.Consent)
		r.Post("/t/pageview", trackH.PageView)
		r.Post("/t/engagement", trackH.Engagement)
		// Bidirectional CRM personalization (Phase 18.1).
		// /t/inbound accepts CRM-pushed metadata for a known visitor;
		// HMAC-SHA256 (same secret as outbound) on X-Atomicsite-Signature.
		r.Post("/t/inbound", trackH.Inbound)
		// /t/visitor (Phase 18.2): per-visitor metadata read endpoint for
		// the built-site hydration script. Re-derives fingerprint from
		// request headers; cross-origin friendly.
		r.Get("/t/visitor", trackH.Visitor)
	})

	// Admin reload endpoint. Dockyard's rotation engine POSTs new shared
	// secrets here with Authorization: Bearer <ADMIN_RELOAD_TOKEN>. The
	// handler does its own bearer check and hot-swaps the in-memory
	// HMAC verifier + signer without restarting the container.
	adminReloadH := handlers.NewAdminReloadHandler(s.cfg, trackH)
	r.Post("/admin/reload-secrets", adminReloadH.ReloadSecrets)

	// Public domain-verify endpoint. The host nginx atomicsite-acme.conf
	// block proxies /.well-known/atomic-verify/* here. No auth: the
	// token is the credential. We expose it BEFORE the auth middleware
	// group so a freshly-pointed custom domain can prove ownership
	// before TLS / login exists.
	publicDomH := handlers.NewDomainHandler(s.queries, nil, "")
	r.Get("/.well-known/atomic-verify/{token}", publicDomH.VerifyToken)

	// Auth (public)
	ah := handlers.NewAuthHandler(s.cfg, s.queries)
	r.Post("/api/auth/login", ah.Login)
	r.Post("/api/auth/logout", ah.Logout)

	// Public invite redemption (no auth - token in path is the credential).
	invH := handlers.NewInvitesHandler(s.cfg, s.queries)
	r.Get("/api/auth/signup/{token}", invH.PublicInfo)
	r.Post("/api/auth/signup/{token}", invH.Redeem)

	// Sites handler (used by both public and authenticated routes).
	sh := handlers.NewSiteHandler(s.cfg, s.queries, s.db)

	// Starter kit catalog (public): the onboarding wizard fetches this
	// before any site exists / before the admin is logged in.
	r.Get("/api/starter-kits", sh.ListStarterKits)

	// Authenticated admin routes
	r.Group(func(r chi.Router) {
		r.Use(s.authMW.Middleware)

		// Auth
		r.Get("/api/auth/me", ah.Me)
		r.Post("/api/auth/change-password", ah.ChangePassword)
		r.Post("/api/auth/sign-out-everywhere", ah.SignOutEverywhere)

		// Account profile + workspace member management.
		memh := handlers.NewMembersHandler(s.cfg, s.queries)
		r.Patch("/api/auth/me", memh.UpdateProfile)
		r.Get("/api/admin/members", memh.List)
		r.Group(func(r chi.Router) {
			r.Use(authmw.RequireAdmin)
			r.Patch("/api/admin/members/{userID}/role", memh.UpdateRole)
			r.Delete("/api/admin/members/{userID}", memh.Delete)
			r.Get("/api/admin/invites", invH.List)
			r.Post("/api/admin/invites", invH.Create)
			r.Delete("/api/admin/invites/{inviteID}", invH.Delete)

			// Admin observability surface (audit I1 + I2). Returns the
			// last retention sweep result, DB stats, and DuckDB
			// availability so the operator has one URL to check from
			// the browser when something feels off. Behind RequireAdmin
			// because it exposes per-site row counts.
			r.Get("/api/admin/metrics", func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "no-store")
				out := map[string]any{
					"sqlite_open_conns": s.db.Stats().OpenConnections,
					"sqlite_in_use":     s.db.Stats().InUse,
					"sqlite_idle":       s.db.Stats().Idle,
					"analyticsdb":       s.AnalyticsDB != nil,
				}
				if s.AnalyticsDB != nil {
					out["analyticsdb_healthy"] = s.AnalyticsDB.Healthy(req.Context())
				}
				if s.RetentionMgr != nil {
					out["retention_last"] = s.RetentionMgr.LastResult()
				}
				_ = json.NewEncoder(w).Encode(out)
			})

			// Audit log: workspace admin sees the global feed.
			auditH := handlers.NewAuditHandler(s.queries)
			r.Get("/api/admin/audit-log", auditH.GlobalFeed)
		})

		// Audit C1: every /api/sites/{siteID}/* route below must verify
		// the authenticated user has a site_members row for that siteID
		// (or is a workspace admin). The siteR sub-router applies
		// SiteAccessMiddleware to every route registered through it; the
		// non-siteID list/create/seed endpoints stay on plain `r`.
		siteAccessMW := authmw.SiteAccessMiddleware(s.queries)
		siteR := r.With(siteAccessMW)

		// Per-site audit feed: any member with site access can read.
		auditFeedH := handlers.NewAuditHandler(s.queries)
		siteR.Get("/api/sites/{siteID}/audit-log", auditFeedH.SiteFeed)

		// Sites
		r.Get("/api/sites", sh.List)
		r.Post("/api/sites", sh.Create)
		r.Post("/api/sites/seed", sh.Seed)
		siteR.Get("/api/sites/{siteID}", sh.Get)
		siteR.Patch("/api/sites/{siteID}", sh.Update)
		siteR.Delete("/api/sites/{siteID}", sh.Delete)
		siteR.Get("/api/sites/{siteID}/silos", sh.ListSilos)

		// Custom domains. The reconciler may be nil (no host integration);
		// the handler still records rows + serves the verify endpoint.
		var reconcileSignal func()
		if s.DomainReconciler != nil {
			reconcileSignal = s.DomainReconciler.Signal
		}
		reservedSuffix := s.cfg.BuiltSiteSuffix
		if reservedSuffix == "" {
			// Match the default in cmd/server/main.go so
			// *.slab.example.com stays reserved
			// even when BUILT_SITE_SUFFIX is unset.
			reservedSuffix = ".slab.example.com"
		}
		domH := handlers.NewDomainHandler(s.queries, reconcileSignal, reservedSuffix)
		siteR.Get("/api/sites/{siteID}/domains", domH.List)
		siteR.Post("/api/sites/{siteID}/domains", domH.Create)
		siteR.Post("/api/sites/{siteID}/domains/{domainID}/canonical", domH.SetCanonical)
		siteR.Post("/api/sites/{siteID}/domains/{domainID}/refresh", domH.Refresh)
		siteR.Delete("/api/sites/{siteID}/domains/{domainID}", domH.Delete)

		// Pages
		ph := handlers.NewPageHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/pages", ph.List)
		siteR.Post("/api/sites/{siteID}/pages", ph.Create)
		siteR.Get("/api/sites/{siteID}/pages/{pageID}", ph.Get)
		siteR.Get("/api/sites/{siteID}/pages/{pageID}/preview", ph.PreviewSource)
		siteR.Patch("/api/sites/{siteID}/pages/{pageID}", ph.Update)
		siteR.Delete("/api/sites/{siteID}/pages/{pageID}", ph.Delete)
		siteR.Post("/api/sites/{siteID}/pages/reorder", ph.Reorder)

		// Blocks
		bh := handlers.NewBlockHandler(s.cfg, s.queries, s.db)
		r.Get("/api/blocks/schemas", bh.Schemas)
		siteR.Get("/api/sites/{siteID}/pages/{pageID}/blocks", bh.List)
		siteR.Post("/api/sites/{siteID}/pages/{pageID}/blocks", bh.Create)
		siteR.Get("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}/preview", bh.Preview)
		siteR.Post("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}/preview-html", bh.PreviewHTML)
		siteR.Patch("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}", bh.Update)
		siteR.Delete("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}", bh.Delete)
		siteR.Post("/api/sites/{siteID}/pages/{pageID}/blocks/reorder", bh.Reorder)
		siteR.Put("/api/sites/{siteID}/pages/{pageID}/blocks/bulk", bh.BulkSave)

		// Global blocks (header / footer slots)
		gbh := handlers.NewGlobalBlockHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/global-blocks", gbh.List)
		siteR.Post("/api/sites/{siteID}/global-blocks", gbh.Create)
		siteR.Get("/api/sites/{siteID}/global-blocks/{blockID}", gbh.Get)
		siteR.Patch("/api/sites/{siteID}/global-blocks/{blockID}", gbh.Update)
		siteR.Delete("/api/sites/{siteID}/global-blocks/{blockID}", gbh.Delete)

		// Knowledgebase
		kbh := handlers.NewKnowledgebaseHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/knowledgebase", kbh.List)
		siteR.Post("/api/sites/{siteID}/knowledgebase", kbh.Create)
		siteR.Get("/api/sites/{siteID}/knowledgebase/{entryID}", kbh.Get)
		siteR.Patch("/api/sites/{siteID}/knowledgebase/{entryID}", kbh.Update)
		siteR.Delete("/api/sites/{siteID}/knowledgebase/{entryID}", kbh.Delete)

		// Components
		ch := handlers.NewComponentHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/components", ch.List)
		siteR.Post("/api/sites/{siteID}/components", ch.Create)
		siteR.Get("/api/sites/{siteID}/components/{componentID}", ch.Get)
		siteR.Patch("/api/sites/{siteID}/components/{componentID}", ch.Update)
		siteR.Delete("/api/sites/{siteID}/components/{componentID}", ch.Delete)

		// CSS Classes
		cch := handlers.NewCSSClassHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/css-classes", cch.List)
		siteR.Post("/api/sites/{siteID}/css-classes", cch.Create)
		siteR.Get("/api/sites/{siteID}/css-classes/{classID}", cch.Get)
		siteR.Patch("/api/sites/{siteID}/css-classes/{classID}", cch.Update)
		siteR.Delete("/api/sites/{siteID}/css-classes/{classID}", cch.Delete)

		// Settings
		seth := handlers.NewSettingsHandler(s.cfg, s.queries)
		if s.OnAnalyticsSettingsChange != nil {
			seth.OnAnalyticsChange(s.OnAnalyticsSettingsChange)
		}
		siteR.Get("/api/sites/{siteID}/settings", seth.List)
		siteR.Get("/api/sites/{siteID}/settings/security/preview", seth.SecurityPreview)
		siteR.Get("/api/sites/{siteID}/settings/cookie-presets", seth.CookiePresets)
		siteR.Get("/api/sites/{siteID}/settings/{category}", seth.ListByCategory)
		siteR.Put("/api/sites/{siteID}/settings", seth.Upsert)
		siteR.Put("/api/sites/{siteID}/settings/bulk", seth.BulkUpsert)
		siteR.Delete("/api/sites/{siteID}/settings/{settingID}", seth.Delete)

		// Guardrails
		grh := handlers.NewGuardrailHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/guardrails", grh.List)
		siteR.Post("/api/sites/{siteID}/guardrails", grh.Create)
		siteR.Get("/api/sites/{siteID}/guardrails/{ruleID}", grh.Get)
		siteR.Patch("/api/sites/{siteID}/guardrails/{ruleID}", grh.Update)
		siteR.Delete("/api/sites/{siteID}/guardrails/{ruleID}", grh.Delete)

		// Profile (business info, legal contacts)
		ph2 := handlers.NewProfileHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/profile", ph2.Get)
		siteR.Put("/api/sites/{siteID}/profile", ph2.Upsert)

		// Allowed scripts (CSP whitelist)
		ash := handlers.NewAllowedScriptHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/allowed-scripts", ash.List)
		siteR.Post("/api/sites/{siteID}/allowed-scripts", ash.Create)
		siteR.Patch("/api/sites/{siteID}/allowed-scripts/{scriptID}", ash.Update)
		siteR.Delete("/api/sites/{siteID}/allowed-scripts/{scriptID}", ash.Delete)

		// Media
		mh := handlers.NewMediaHandler(s.cfg, s.queries, s.storage)
		siteR.Get("/api/sites/{siteID}/media", mh.List)
		siteR.Post("/api/sites/{siteID}/media", mh.Upload)
		siteR.Get("/api/sites/{siteID}/media/folders", mh.ListFolders)
		siteR.Post("/api/sites/{siteID}/media/folders", mh.CreateFolder)
		siteR.Delete("/api/sites/{siteID}/media/folders/{folderName}", mh.DeleteFolder)
		siteR.Get("/api/sites/{siteID}/media/{mediaID}", mh.Get)
		siteR.Patch("/api/sites/{siteID}/media/{mediaID}", mh.Update)
		siteR.Delete("/api/sites/{siteID}/media/{mediaID}", mh.Delete)

		// Builds (admin)
		buildH := handlers.NewBuildHandler(s.cfg, s.queries)
		siteR.Post("/api/sites/{siteID}/build", buildH.TriggerBuildAdmin)
		siteR.Get("/api/sites/{siteID}/builds/{buildID}/status", buildH.BuildStatusAdmin)
		siteR.Post("/api/sites/{siteID}/builds/{buildID}/cancel", buildH.CancelBuild)
		siteR.Post("/api/sites/{siteID}/deploy", buildH.Deploy)

		// Deploy targets (admin)
		dh := handlers.NewDeployHandler(s.cfg, s.queries, s.db)
		siteR.Get("/api/sites/{siteID}/deploy-targets", dh.ListTargets)
		siteR.Post("/api/sites/{siteID}/deploy-targets", dh.CreateTarget)
		siteR.Get("/api/sites/{siteID}/deploy-targets/{targetID}", dh.GetTarget)
		siteR.Patch("/api/sites/{siteID}/deploy-targets/{targetID}", dh.UpdateTarget)
		siteR.Delete("/api/sites/{siteID}/deploy-targets/{targetID}", dh.DeleteTarget)
		siteR.Post("/api/sites/{siteID}/deploy-targets/{targetID}/default", dh.SetDefault)

		// Evaluations (admin)
		evalH := handlers.NewEvaluationHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/evaluations", evalH.ListBySite)
		siteR.Get("/api/sites/{siteID}/evaluations/{buildID}", evalH.ListByBuild)

		// Analytics reads (admin) - visit_events populated by the nginx log tailer.
		anH := handlers.NewAnalyticsHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/visit-events", anH.VisitEvents)
		siteR.Get("/api/sites/{siteID}/analytics/overview", anH.AnalyticsOverview)
		siteR.Get("/api/sites/{siteID}/analytics/sessions", anH.AnalyticsSessions)
		siteR.Get("/api/sites/{siteID}/analytics/conversion-paths", anH.AnalyticsConversionPaths)
		siteR.Get("/api/sites/{siteID}/analytics/tracked-fields", anH.AnalyticsTrackedFields)

		// Consent records (GDPR proof log). atomicsite became system of record
		// for tenant consent after the CookieProof fold-in (2026-04-30); these
		// endpoints back the dashboard's Cookies section.
		coH := handlers.NewConsentHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/consent/records", coH.List)
		siteR.Get("/api/sites/{siteID}/consent/records/{recordID}", coH.Get)
		siteR.Get("/api/sites/{siteID}/consent/stats", coH.Stats)
		siteR.Get("/api/sites/{siteID}/consent/stats-by-category", coH.StatsByCategory)
		siteR.Get("/api/sites/{siteID}/consent/export.csv", coH.ExportCSV)
		siteR.Delete("/api/sites/{siteID}/consent/purge", coH.Purge)

		// Stitched cookie analytics — DuckDB ATTACH layer over the SQLite
		// file. Returns 503 from each endpoint when the DuckDB layer
		// failed to open at boot (e.g. a CGO-disabled build), so the
		// dashboard can fall back gracefully.
		// Cookie banner live preview: returns an HTML page with the real
		// CookieProof widget mounted in previewMode and the preferences
		// modal auto-opened. The admin SPA loads this as a same-origin
		// iframe on the Settings -> Cookies page so editors see the
		// exact production banner (cookie tables, language toggle,
		// privacy policy link, branding colors) instead of a mockup.
		cpH := handlers.NewCookiesPreviewHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/cookies/preview", cpH.Render)

		caH := handlers.NewCookieAnalyticsHandler(s.cfg, s.queries, s.AnalyticsDB)
		siteR.Get("/api/sites/{siteID}/cookies/analytics/funnel", caH.Funnel)
		siteR.Get("/api/sites/{siteID}/cookies/analytics/pre-consent", caH.PreConsent)
		siteR.Get("/api/sites/{siteID}/cookies/analytics/time-to-consent", caH.TimeToConsent)
		siteR.Get("/api/sites/{siteID}/cookies/analytics/engaged-accept-rate", caH.EngagedAcceptRate)
		siteR.Get("/api/sites/{siteID}/cookies/analytics/top-accepting-pages", caH.TopAcceptingPages)
		siteR.Get("/api/sites/{siteID}/cookies/analytics/daily", caH.DailySplit)
		siteR.Get("/api/sites/{siteID}/cookies/analytics/journey/{fingerprint}", caH.VisitorJourney)

		// Agent keys (admin management)
		agh := handlers.NewAgentHandler(s.cfg, s.queries, s.db)
		siteR.Get("/api/sites/{siteID}/agent-keys", agh.ListAgentKeys)
		siteR.Post("/api/sites/{siteID}/agent-keys", agh.GenerateAgentKey)
		siteR.Delete("/api/sites/{siteID}/agent-keys/{keyID}", agh.RevokeAgentKey)

		// One-click agent bootstrap (Phase 14): generates a key + returns
		// a download-ready CLAUDE.md / .env / smoke-test bundle. Lets a
		// user wire an agent in one click instead of four manual steps.
		siteR.Post("/api/sites/{siteID}/agent-bootstrap", agh.AgentBootstrap)

		// Figma design-tokens import (admin)
		fh := handlers.NewFigmaHandler(s.cfg, s.queries)
		siteR.Post("/api/sites/{siteID}/figma/import", fh.ImportDesignTokens)

		// Self-hosted woff2 fonts (admin upload + delete; public serves Serve below)
		fonth := handlers.NewFontsHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/fonts", fonth.List)
		siteR.Post("/api/sites/{siteID}/fonts", fonth.Upload)
		siteR.Delete("/api/sites/{siteID}/fonts/{fontID}", fonth.Delete)

		// GitHub design references (admin)
		drh := handlers.NewDesignReferencesHandler(s.cfg, s.queries)
		siteR.Get("/api/sites/{siteID}/design-references", drh.List)
		siteR.Post("/api/sites/{siteID}/design-references", drh.Create)
		siteR.Post("/api/sites/{siteID}/design-references/{refID}/refresh", drh.Refresh)
		siteR.Delete("/api/sites/{siteID}/design-references/{refID}", drh.Delete)
	})

	// Public font serving (no auth, long cache, CORS *).
	publicFontsH := handlers.NewFontsHandler(s.cfg, s.queries)
	r.Get("/atomicsite-fonts/{siteID}/{fontID}.woff2", publicFontsH.Serve)

	// Agent API routes (API key auth)
	r.Group(func(r chi.Router) {
		r.Use(s.agentMW.Middleware)

		agentH := handlers.NewAgentHandler(s.cfg, s.queries, s.db)

		// Context
		r.Get("/api/agent/context", agentH.Context)

		// Bootstrap: re-fetch CLAUDE.md (text/markdown) any time. Lets an
		// agent re-read its own instructions on session start.
		r.Get("/api/agent/bootstrap", agentH.AgentSelfBootstrap)

		// Personalization debug (Phase 18.4): inspect what metadata the
		// CRM has pushed for a given visitor. Bound to the agent's site
		// via the agent identity.
		r.Get("/api/agent/visitor-metadata", agentH.GetVisitorMetadata)

		// Pages (slug via path or ?page= query param)
		r.Post("/api/agent/pages", agentH.CreatePage)
		r.Patch("/api/agent/pages/{slug}", agentH.UpdatePage)
		r.Delete("/api/agent/pages/{slug}", agentH.DeletePage)

		// Blocks (page slug via ?page= query param)
		r.Post("/api/agent/blocks", agentH.CreateBlock)
		r.Post("/api/agent/pages/{slug}/blocks", agentH.CreateBlock)
		r.Patch("/api/agent/blocks/{blockID}", agentH.UpdateBlock)
		r.Delete("/api/agent/blocks/{blockID}", agentH.DeleteBlock)

		// Components
		r.Post("/api/agent/components", agentH.CreateComponent)
		r.Patch("/api/agent/components/{name}", agentH.UpdateComponent)

		// CSS Classes
		r.Post("/api/agent/css-classes", agentH.CreateCSSClass)
		r.Patch("/api/agent/css-classes/{name}", agentH.UpdateCSSClass)

		// Global Blocks
		r.Put("/api/agent/global/{slot}", agentH.UpdateGlobalBlock)

		// Build
		agentBuildH := handlers.NewBuildHandler(s.cfg, s.queries)
		r.Post("/api/agent/build", agentBuildH.TriggerBuild)
		r.Get("/api/agent/build/{buildID}/status", agentBuildH.BuildStatus)

		// Screenshot. Closes the visual-feedback loop: agent edits blocks,
		// triggers build, calls /api/agent/screenshot to get a base64 PNG of
		// the rendered page, and can iterate against pixels in the same turn.
		// SSRF-locked to the configured tenant subdomains and primary domain.
		screenshotH := handlers.NewScreenshotHandler()
		r.Post("/api/agent/screenshot", screenshotH.Screenshot)

		// MCP server (Model Context Protocol). One JSON-RPC endpoint
		// translating tools/call + resources/read into the same store
		// + builder logic the REST handlers use. Allow-list driven so
		// future REST endpoints can't accidentally leak PII through MCP.
		// Auth: same X-Agent-Key as the rest of /api/agent/*.
		mcpServer := mcp.NewServer(s.queries, agentBuildH)
		r.Mount("/mcp", mcpServer.Handler())

		// Evaluation
		r.Get("/api/agent/evaluation/{buildID}", agentH.GetEvaluation)

		// Branding (Phase 12.9): agents can read + patch the same fields the
		// human admin edits in the Branding UI. Writes require "write" cap.
		r.Get("/api/agent/branding", agentH.GetBranding)
		r.Patch("/api/agent/branding", agentH.UpdateBranding)

		// Profile (Phase 14): agents can read + patch the site_profiles row
		// (business name, address, contact emails) so they can fill the
		// Organization JSON-LD, security.txt, and legal pages on their own.
		r.Get("/api/agent/profile", agentH.GetProfile)
		r.Patch("/api/agent/profile", agentH.UpdateProfile)

		// Settings (Phase 14): read any category, write to seo / analytics /
		// general only. Security, allowed-scripts, nginx, danger stay
		// admin-only because they affect attack surface.
		r.Get("/api/agent/settings", agentH.ListSettings)
		r.Get("/api/agent/settings/{category}", agentH.ListSettingsByCategory)
		r.Patch("/api/agent/settings", agentH.BulkUpsertSettings)

		// Trusted external domains (CSP allowlist) + resolved security
		// headers preview. Read-only for the agent: writes here widen
		// the site's attack surface so they stay admin-only via the
		// /api/sites/{id}/allowed-scripts endpoint and the
		// /api/sites/{id}/settings/security UI.
		r.Get("/api/agent/allowed-scripts", agentH.ListAllowedScripts)
		r.Get("/api/agent/security/preview", agentH.SecurityPreview)

		// Self-hosted woff2 fonts (Phase 12.9 agent parity).
		r.Get("/api/agent/fonts", agentH.ListFonts)
		r.Post("/api/agent/fonts", agentH.UploadFont)
		r.Delete("/api/agent/fonts/{fontID}", agentH.DeleteFont)

		// GitHub design references (Phase 12.9 agent parity).
		r.Get("/api/agent/design-references", agentH.ListDesignReferences)
		r.Post("/api/agent/design-references", agentH.AddDesignReference)
		r.Delete("/api/agent/design-references/{refID}", agentH.DeleteDesignReference)

		// Media
		agentMediaH := handlers.NewMediaHandler(s.cfg, s.queries, s.storage)
		r.Get("/api/agent/media", agentH.ListMedia)
		r.Get("/api/agent/media/folders", agentH.ListMediaFolders)
		r.Post("/api/agent/media/folders", agentH.CreateMediaFolder)
		r.Delete("/api/agent/media/folders/{folderName}", agentH.DeleteMediaFolder)
		r.Post("/api/agent/media", agentMediaH.AgentUpload)
		r.Post("/api/agent/media/from-url", agentMediaH.AgentUploadFromURL)
		r.Post("/api/agent/media/from-base64", agentMediaH.AgentUploadFromBase64)
		r.Patch("/api/agent/media/{mediaID}", agentMediaH.AgentUpdate)
		r.Delete("/api/agent/media/{mediaID}", agentMediaH.AgentDelete)
	})

	// Public media serving (no auth, long cache). Must come BEFORE mountFrontend.
	publicMediaH := handlers.NewMediaHandler(s.cfg, s.queries, s.storage)
	r.Get("/media/*", publicMediaH.ServePublic)
	r.Head("/media/*", publicMediaH.ServePublic)

	// Serve embedded frontend (SPA)
	s.mountFrontend(r)

	return r
}

func (s *Server) mountFrontend(r chi.Router) {
	if FrontendFS == nil {
		return
	}
	indexHTML, err := fs.ReadFile(FrontendFS, "index.html")
	if err != nil {
		slog.Warn("frontend index.html missing", "error", err)
		return
	}
	startedAt := time.Now()

	serveIndex := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(indexHTML)
	}

	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" {
			serveIndex(w)
			return
		}
		// Reject path traversal.
		if strings.Contains(path, "..") {
			http.NotFound(w, req)
			return
		}
		f, err := FrontendFS.Open(path)
		if err != nil {
			// Unknown path: SPA client-side router takes over.
			serveIndex(w)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			serveIndex(w)
			return
		}
		seeker, ok := f.(io.ReadSeeker)
		if !ok {
			data, err := io.ReadAll(f)
			if err != nil {
				http.Error(w, "read embed", http.StatusInternalServerError)
				return
			}
			seeker = bytes.NewReader(data)
		}
		if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		if strings.HasPrefix(path, "_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.ServeContent(w, req, info.Name(), startedAt, seeker)
	})
}

// corsMiddleware returns a path-aware CORS handler. The audit's C2
// finding: the previous unified policy granted Access-Control-Allow-
// Credentials: true to every `*.<BuiltSiteSuffix>` origin on every
// route, including `/api/sites/...`. Combined with the missing
// per-site authorization (audit C1), a tenant who controls their own
// built-site origin could pivot a single line of injected JS into a
// cross-tenant takeover.
//
// Two policies now:
//
//  1. Admin routes (`/api/*` and the SPA fallback): credentialed CORS
//     allowed ONLY from `cfg.BaseURL` exact origin (and localhost in
//     dev). Built-site subdomains do not need to call admin routes;
//     all dashboard traffic is same-origin.
//
//  2. Public visitor endpoints (`/t/*`): credentialed CORS allowed
//     from `cfg.BaseURL` AND any host suffix-matching
//     `cfg.BuiltSiteSuffix`. Built sites legitimately need to reach
//     `/t/consent`, `/t/visitor`, `/t/inbound` cross-origin to lift
//     anonymous visits to identified.
//
// Preflight OPTIONS uses the same path-based policy so the browser
// reflects the correct Access-Control-Allow-Origin during the
// preflight handshake.
func corsMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && isAllowedOriginForPath(cfg, origin, r.URL.Path) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Agent-Key")
				// Vary: Origin keeps cache layers from serving the wrong
				// allow-origin to a different visitor.
				w.Header().Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isPublicVisitorPath reports whether the path is one of the public
// `/t/*` endpoints that built tenant sites legitimately call cross-origin.
func isPublicVisitorPath(p string) bool {
	return strings.HasPrefix(p, "/t/")
}

// isAllowedOriginForPath is the path-aware CORS decision. Admin routes
// only accept the configured BaseURL (plus localhost in dev). Public
// `/t/*` routes additionally accept any built-site subdomain.
func isAllowedOriginForPath(cfg *config.Config, origin, path string) bool {
	// Always allow the configured base URL exactly.
	if origin == cfg.BaseURL || strings.HasPrefix(origin, cfg.BaseURL+"/") {
		return true
	}
	// Local dev: any localhost origin (admin + built-site preview both
	// run on localhost during `go run`).
	if cfg.IsLocalDev() {
		if strings.HasPrefix(origin, "http://localhost") ||
			strings.HasPrefix(origin, "http://127.0.0.1") {
			return true
		}
	}
	// Built-site subdomains: ONLY for the public visitor endpoints.
	// Admin routes (`/api/*`, `/admin/*`, SPA) do not need this and
	// granting it widens the attack surface without benefit (C2).
	if isPublicVisitorPath(path) && cfg.BuiltSiteSuffix != "" {
		host := stripScheme(origin)
		if strings.HasSuffix(host, cfg.BuiltSiteSuffix) {
			return true
		}
	}
	return false
}

// boolToOKFail maps a healthcheck bool to the "ok" / "fail" string the
// /api/health JSON response uses.
func boolToOKFail(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}

// stripScheme reduces an Origin header value (e.g. https://foo.example:443)
// to bare host for suffix comparison. Returns "" when the input is malformed.
func stripScheme(origin string) string {
	host := origin
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host
}


// jsonBodySizeMiddleware caps r.Body to maxBytes before any handler
// reads it. Without this, a malicious client can POST gigabytes of
// JSON and exhaust the goroutine's memory before json.Decoder hits a
// natural error. http.MaxBytesReader emits a clean 413-style failure
// at the decode site once the cap is exceeded.
//
// File-upload routes that need a higher cap call MaxBytesReader
// themselves with a larger value, which overrides the wrap below
// (the inner Read returns ErrBodyTooLarge once either limit hits).
//
// Small file like this lives next to the middleware it wraps for
// discoverability. If we add a third middleware we'll split into
// internal/server/middleware.go.
func jsonBodySizeMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.ContentLength != 0 {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
