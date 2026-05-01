// Package server sets up the chi router and wires all handlers.
package server

import (
	"bytes"
	"context"
	"database/sql"
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
	"github.com/bright-interaction/slab/internal/handlers"
	"github.com/bright-interaction/slab/internal/mcp"
	authmw "github.com/bright-interaction/slab/internal/middleware"
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
	// OnAnalyticsSettingsChange, when set, is invoked after settings writes that
	// touch the analytics category so the analytics Manager can rescan parsers.
	OnAnalyticsSettingsChange func(context.Context)
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

	// Health (public)
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
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
		})

		// Sites
		r.Get("/api/sites", sh.List)
		r.Post("/api/sites", sh.Create)
		r.Post("/api/sites/seed", sh.Seed)
		r.Get("/api/sites/{siteID}", sh.Get)
		r.Patch("/api/sites/{siteID}", sh.Update)
		r.Delete("/api/sites/{siteID}", sh.Delete)
		r.Get("/api/sites/{siteID}/silos", sh.ListSilos)

		// Pages
		ph := handlers.NewPageHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/pages", ph.List)
		r.Post("/api/sites/{siteID}/pages", ph.Create)
		r.Get("/api/sites/{siteID}/pages/{pageID}", ph.Get)
		r.Get("/api/sites/{siteID}/pages/{pageID}/preview", ph.PreviewSource)
		r.Patch("/api/sites/{siteID}/pages/{pageID}", ph.Update)
		r.Delete("/api/sites/{siteID}/pages/{pageID}", ph.Delete)
		r.Post("/api/sites/{siteID}/pages/reorder", ph.Reorder)

		// Blocks
		bh := handlers.NewBlockHandler(s.cfg, s.queries, s.db)
		r.Get("/api/sites/{siteID}/pages/{pageID}/blocks", bh.List)
		r.Post("/api/sites/{siteID}/pages/{pageID}/blocks", bh.Create)
		r.Get("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}/preview", bh.Preview)
		r.Patch("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}", bh.Update)
		r.Delete("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}", bh.Delete)
		r.Post("/api/sites/{siteID}/pages/{pageID}/blocks/reorder", bh.Reorder)
		r.Put("/api/sites/{siteID}/pages/{pageID}/blocks/bulk", bh.BulkSave)

		// Global blocks (header / footer slots)
		gbh := handlers.NewGlobalBlockHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/global-blocks", gbh.List)
		r.Post("/api/sites/{siteID}/global-blocks", gbh.Create)
		r.Get("/api/sites/{siteID}/global-blocks/{blockID}", gbh.Get)
		r.Patch("/api/sites/{siteID}/global-blocks/{blockID}", gbh.Update)
		r.Delete("/api/sites/{siteID}/global-blocks/{blockID}", gbh.Delete)

		// Knowledgebase
		kbh := handlers.NewKnowledgebaseHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/knowledgebase", kbh.List)
		r.Post("/api/sites/{siteID}/knowledgebase", kbh.Create)
		r.Get("/api/sites/{siteID}/knowledgebase/{entryID}", kbh.Get)
		r.Patch("/api/sites/{siteID}/knowledgebase/{entryID}", kbh.Update)
		r.Delete("/api/sites/{siteID}/knowledgebase/{entryID}", kbh.Delete)

		// Components
		ch := handlers.NewComponentHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/components", ch.List)
		r.Post("/api/sites/{siteID}/components", ch.Create)
		r.Get("/api/sites/{siteID}/components/{componentID}", ch.Get)
		r.Patch("/api/sites/{siteID}/components/{componentID}", ch.Update)
		r.Delete("/api/sites/{siteID}/components/{componentID}", ch.Delete)

		// CSS Classes
		cch := handlers.NewCSSClassHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/css-classes", cch.List)
		r.Post("/api/sites/{siteID}/css-classes", cch.Create)
		r.Get("/api/sites/{siteID}/css-classes/{classID}", cch.Get)
		r.Patch("/api/sites/{siteID}/css-classes/{classID}", cch.Update)
		r.Delete("/api/sites/{siteID}/css-classes/{classID}", cch.Delete)

		// Settings
		seth := handlers.NewSettingsHandler(s.cfg, s.queries)
		if s.OnAnalyticsSettingsChange != nil {
			seth.OnAnalyticsChange(s.OnAnalyticsSettingsChange)
		}
		r.Get("/api/sites/{siteID}/settings", seth.List)
		r.Get("/api/sites/{siteID}/settings/security/preview", seth.SecurityPreview)
		r.Get("/api/sites/{siteID}/settings/cookie-presets", seth.CookiePresets)
		r.Get("/api/sites/{siteID}/settings/{category}", seth.ListByCategory)
		r.Put("/api/sites/{siteID}/settings", seth.Upsert)
		r.Put("/api/sites/{siteID}/settings/bulk", seth.BulkUpsert)
		r.Delete("/api/sites/{siteID}/settings/{settingID}", seth.Delete)

		// Guardrails
		grh := handlers.NewGuardrailHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/guardrails", grh.List)
		r.Post("/api/sites/{siteID}/guardrails", grh.Create)
		r.Get("/api/sites/{siteID}/guardrails/{ruleID}", grh.Get)
		r.Patch("/api/sites/{siteID}/guardrails/{ruleID}", grh.Update)
		r.Delete("/api/sites/{siteID}/guardrails/{ruleID}", grh.Delete)

		// Profile (business info, legal contacts)
		ph2 := handlers.NewProfileHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/profile", ph2.Get)
		r.Put("/api/sites/{siteID}/profile", ph2.Upsert)

		// Allowed scripts (CSP whitelist)
		ash := handlers.NewAllowedScriptHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/allowed-scripts", ash.List)
		r.Post("/api/sites/{siteID}/allowed-scripts", ash.Create)
		r.Patch("/api/sites/{siteID}/allowed-scripts/{scriptID}", ash.Update)
		r.Delete("/api/sites/{siteID}/allowed-scripts/{scriptID}", ash.Delete)

		// Media
		mh := handlers.NewMediaHandler(s.cfg, s.queries, s.storage)
		r.Get("/api/sites/{siteID}/media", mh.List)
		r.Post("/api/sites/{siteID}/media", mh.Upload)
		r.Get("/api/sites/{siteID}/media/folders", mh.ListFolders)
		r.Post("/api/sites/{siteID}/media/folders", mh.CreateFolder)
		r.Delete("/api/sites/{siteID}/media/folders/{folderName}", mh.DeleteFolder)
		r.Get("/api/sites/{siteID}/media/{mediaID}", mh.Get)
		r.Patch("/api/sites/{siteID}/media/{mediaID}", mh.Update)
		r.Delete("/api/sites/{siteID}/media/{mediaID}", mh.Delete)

		// Builds (admin)
		buildH := handlers.NewBuildHandler(s.cfg, s.queries)
		r.Post("/api/sites/{siteID}/build", buildH.TriggerBuildAdmin)
		r.Get("/api/sites/{siteID}/builds/{buildID}/status", buildH.BuildStatusAdmin)
		r.Post("/api/sites/{siteID}/deploy", buildH.Deploy)

		// Deploy targets (admin)
		dh := handlers.NewDeployHandler(s.cfg, s.queries, s.db)
		r.Get("/api/sites/{siteID}/deploy-targets", dh.ListTargets)
		r.Post("/api/sites/{siteID}/deploy-targets", dh.CreateTarget)
		r.Get("/api/sites/{siteID}/deploy-targets/{targetID}", dh.GetTarget)
		r.Patch("/api/sites/{siteID}/deploy-targets/{targetID}", dh.UpdateTarget)
		r.Delete("/api/sites/{siteID}/deploy-targets/{targetID}", dh.DeleteTarget)
		r.Post("/api/sites/{siteID}/deploy-targets/{targetID}/default", dh.SetDefault)

		// Evaluations (admin)
		evalH := handlers.NewEvaluationHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/evaluations", evalH.ListBySite)
		r.Get("/api/sites/{siteID}/evaluations/{buildID}", evalH.ListByBuild)

		// Analytics reads (admin) - visit_events populated by the nginx log tailer.
		anH := handlers.NewAnalyticsHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/visit-events", anH.VisitEvents)
		r.Get("/api/sites/{siteID}/analytics/overview", anH.AnalyticsOverview)
		r.Get("/api/sites/{siteID}/analytics/sessions", anH.AnalyticsSessions)
		r.Get("/api/sites/{siteID}/analytics/conversion-paths", anH.AnalyticsConversionPaths)
		r.Get("/api/sites/{siteID}/analytics/tracked-fields", anH.AnalyticsTrackedFields)

		// Consent records (GDPR proof log). atomicsite became system of record
		// for tenant consent after the CookieProof fold-in (2026-04-30); these
		// endpoints back the dashboard's Cookies section.
		coH := handlers.NewConsentHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/consent/records", coH.List)
		r.Get("/api/sites/{siteID}/consent/records/{recordID}", coH.Get)
		r.Get("/api/sites/{siteID}/consent/stats", coH.Stats)
		r.Get("/api/sites/{siteID}/consent/stats-by-category", coH.StatsByCategory)
		r.Get("/api/sites/{siteID}/consent/export.csv", coH.ExportCSV)
		r.Delete("/api/sites/{siteID}/consent/purge", coH.Purge)

		// Stitched cookie analytics — DuckDB ATTACH layer over the SQLite
		// file. Returns 503 from each endpoint when the DuckDB layer
		// failed to open at boot (e.g. a CGO-disabled build), so the
		// dashboard can fall back gracefully.
		caH := handlers.NewCookieAnalyticsHandler(s.cfg, s.queries, s.AnalyticsDB)
		r.Get("/api/sites/{siteID}/cookies/analytics/funnel", caH.Funnel)
		r.Get("/api/sites/{siteID}/cookies/analytics/pre-consent", caH.PreConsent)
		r.Get("/api/sites/{siteID}/cookies/analytics/time-to-consent", caH.TimeToConsent)
		r.Get("/api/sites/{siteID}/cookies/analytics/engaged-accept-rate", caH.EngagedAcceptRate)
		r.Get("/api/sites/{siteID}/cookies/analytics/top-accepting-pages", caH.TopAcceptingPages)
		r.Get("/api/sites/{siteID}/cookies/analytics/daily", caH.DailySplit)
		r.Get("/api/sites/{siteID}/cookies/analytics/journey/{fingerprint}", caH.VisitorJourney)

		// Agent keys (admin management)
		agh := handlers.NewAgentHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/agent-keys", agh.ListAgentKeys)
		r.Post("/api/sites/{siteID}/agent-keys", agh.GenerateAgentKey)
		r.Delete("/api/sites/{siteID}/agent-keys/{keyID}", agh.RevokeAgentKey)

		// One-click agent bootstrap (Phase 14): generates a key + returns
		// a download-ready CLAUDE.md / .env / smoke-test bundle. Lets a
		// user wire an agent in one click instead of four manual steps.
		r.Post("/api/sites/{siteID}/agent-bootstrap", agh.AgentBootstrap)

		// Figma design-tokens import (admin)
		fh := handlers.NewFigmaHandler(s.cfg, s.queries)
		r.Post("/api/sites/{siteID}/figma/import", fh.ImportDesignTokens)

		// Self-hosted woff2 fonts (admin upload + delete; public serves Serve below)
		fonth := handlers.NewFontsHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/fonts", fonth.List)
		r.Post("/api/sites/{siteID}/fonts", fonth.Upload)
		r.Delete("/api/sites/{siteID}/fonts/{fontID}", fonth.Delete)

		// GitHub design references (admin)
		drh := handlers.NewDesignReferencesHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/design-references", drh.List)
		r.Post("/api/sites/{siteID}/design-references", drh.Create)
		r.Post("/api/sites/{siteID}/design-references/{refID}/refresh", drh.Refresh)
		r.Delete("/api/sites/{siteID}/design-references/{refID}", drh.Delete)
	})

	// Public font serving (no auth, long cache, CORS *).
	publicFontsH := handlers.NewFontsHandler(s.cfg, s.queries)
	r.Get("/atomicsite-fonts/{siteID}/{fontID}.woff2", publicFontsH.Serve)

	// Agent API routes (API key auth)
	r.Group(func(r chi.Router) {
		r.Use(s.agentMW.Middleware)

		agentH := handlers.NewAgentHandler(s.cfg, s.queries)

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
		// SSRF-locked to atomicsite tenant subdomains + brightinteraction.com.
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

func corsMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && isAllowedOrigin(cfg, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Agent-Key")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(cfg *config.Config, origin string) bool {
	// In development, allow localhost origins
	if strings.HasPrefix(cfg.BaseURL, "http://localhost") {
		if strings.HasPrefix(origin, "http://localhost") {
			return true
		}
	}
	// Allow the configured base URL origin
	if strings.HasPrefix(origin, cfg.BaseURL) {
		return true
	}
	// Phase 18: built sites live at <slug>.slab.example.com
	// (or whatever the BUILT_SITE_SUFFIX env points at) and need to reach
	// the admin host's /t/visitor + /t/inbound + /t/consent endpoints
	// cross-origin. Match by hostname suffix to cover any slug.
	if cfg.BuiltSiteSuffix != "" {
		// Strip protocol and port to compare just the host.
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
		if strings.HasSuffix(host, cfg.BuiltSiteSuffix) {
			return true
		}
	}
	// Agent API requests use API keys, not cookies, so CORS is less critical.
	// But we still restrict to known origins for cookie-based admin endpoints.
	return false
}

