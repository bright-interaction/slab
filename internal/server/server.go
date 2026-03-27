// Package server sets up the chi router and wires all handlers.
package server

import (
	"database/sql"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/handlers"
	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// FrontendFS is the embedded frontend dist. Populated by cmd/server/main.go.
var FrontendFS fs.FS

// Server holds all dependencies.
type Server struct {
	cfg     *config.Config
	db      *sql.DB
	queries *store.Queries
	authMW  *authmw.AuthMiddleware
}

// New creates a Server.
func New(cfg *config.Config, db *sql.DB, queries *store.Queries) *Server {
	return &Server{
		cfg:     cfg,
		db:      db,
		queries: queries,
		authMW:  authmw.NewAuthMiddleware(cfg, queries),
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

	// Authenticated routes
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
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
