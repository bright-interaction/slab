// Package server sets up the chi router and wires all handlers.
package server

import (
	"database/sql"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/handlers"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/store"
)

// FrontendFS is the embedded frontend dist. Populated by cmd/server/main.go.
var FrontendFS fs.FS

// Server holds all dependencies.
type Server struct {
	cfg      *config.Config
	db       *sql.DB
	queries  *store.Queries
	authMW   *authmw.AuthMiddleware
	agentMW  *authmw.AgentAuthMiddleware
}

// New creates a Server.
func New(cfg *config.Config, db *sql.DB, queries *store.Queries) *Server {
	return &Server{
		cfg:     cfg,
		db:      db,
		queries: queries,
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

	// Auth (public)
	ah := handlers.NewAuthHandler(s.cfg, s.queries)
	r.Post("/api/auth/login", ah.Login)
	r.Post("/api/auth/logout", ah.Logout)

	// Authenticated admin routes
	r.Group(func(r chi.Router) {
		r.Use(s.authMW.Middleware)

		// Auth
		r.Get("/api/auth/me", ah.Me)
		r.Post("/api/auth/change-password", ah.ChangePassword)

		// Sites
		sh := handlers.NewSiteHandler(s.cfg, s.queries)
		r.Get("/api/sites", sh.List)
		r.Post("/api/sites", sh.Create)
		r.Get("/api/sites/{siteID}", sh.Get)
		r.Patch("/api/sites/{siteID}", sh.Update)
		r.Delete("/api/sites/{siteID}", sh.Delete)

		// Pages
		ph := handlers.NewPageHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/pages", ph.List)
		r.Post("/api/sites/{siteID}/pages", ph.Create)
		r.Get("/api/sites/{siteID}/pages/{pageID}", ph.Get)
		r.Patch("/api/sites/{siteID}/pages/{pageID}", ph.Update)
		r.Delete("/api/sites/{siteID}/pages/{pageID}", ph.Delete)
		r.Post("/api/sites/{siteID}/pages/reorder", ph.Reorder)

		// Blocks
		bh := handlers.NewBlockHandler(s.cfg, s.queries, s.db)
		r.Get("/api/sites/{siteID}/pages/{pageID}/blocks", bh.List)
		r.Post("/api/sites/{siteID}/pages/{pageID}/blocks", bh.Create)
		r.Patch("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}", bh.Update)
		r.Delete("/api/sites/{siteID}/pages/{pageID}/blocks/{blockID}", bh.Delete)
		r.Post("/api/sites/{siteID}/pages/{pageID}/blocks/reorder", bh.Reorder)
		r.Put("/api/sites/{siteID}/pages/{pageID}/blocks/bulk", bh.BulkSave)

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
		r.Get("/api/sites/{siteID}/settings", seth.List)
		r.Get("/api/sites/{siteID}/settings/{category}", seth.ListByCategory)
		r.Put("/api/sites/{siteID}/settings", seth.Upsert)
		r.Put("/api/sites/{siteID}/settings/bulk", seth.BulkUpsert)
		r.Delete("/api/sites/{siteID}/settings/{settingID}", seth.Delete)

		// Agent keys (admin management)
		agh := handlers.NewAgentHandler(s.cfg, s.queries)
		r.Get("/api/sites/{siteID}/agent-keys", agh.ListAgentKeys)
		r.Post("/api/sites/{siteID}/agent-keys", agh.GenerateAgentKey)
		r.Delete("/api/sites/{siteID}/agent-keys/{keyID}", agh.RevokeAgentKey)
	})

	// Agent API routes (API key auth)
	r.Group(func(r chi.Router) {
		r.Use(s.agentMW.Middleware)

		agentH := handlers.NewAgentHandler(s.cfg, s.queries)

		// Context
		r.Get("/api/agent/context", agentH.Context)

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

		// Evaluation
		r.Get("/api/agent/evaluation/{buildID}", agentH.GetEvaluation)

		// Media
		r.Get("/api/agent/media", agentH.ListMedia)
	})

	// Serve embedded frontend (SPA)
	s.mountFrontend(r)

	return r
}

func (s *Server) mountFrontend(r chi.Router) {
	if FrontendFS == nil {
		return
	}
	staticFS := http.FS(FrontendFS)
	fileServer := http.FileServer(staticFS)
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		f, err := FrontendFS.Open(req.URL.Path[1:]) // strip leading /
		if err != nil {
			// SPA fallback: serve index.html
			req.URL.Path = "/index.html"
			fileServer.ServeHTTP(w, req)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, req)
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
	// Agent API requests use API keys, not cookies, so CORS is less critical.
	// But we still restrict to known origins for cookie-based admin endpoints.
	return false
}
