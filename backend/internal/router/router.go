// Package router assembles all routes in the PRD §5.2 order.
package router

import (
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"shorturl/internal/config"
	"shorturl/internal/handler"
	"shorturl/internal/middleware"
)

// New builds the root http.Handler.
func New(cfg *config.Config, api *handler.API, log *slog.Logger, trustedProxies []*net.IPNet) http.Handler {
	_ = trustedProxies
	r := chi.NewRouter()

	// 1. global middleware
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.Recoverer(log))
	r.Use(middleware.Logging(log, nil))
	r.Use(middleware.SecureHeaders(cfg.IsProduction(), false))

	// 2. health
	r.Get("/healthz", api.HandleHealthz)
	r.Get("/readyz", api.HandleReadyz)

	// 3. API
	r.Route("/api/v1", func(ar chi.Router) {
		ar.Use(middleware.NoStore)
		ar.Use(middleware.NoIndex)
		ar.Post("/urls", api.HandleCreateURL)
		ar.Get("/urls", api.HandleListURLs)
		ar.Get("/urls/{id}", api.HandleGetURL)
		ar.Patch("/urls/{id}", api.HandlePatchURL)
		ar.Delete("/urls/{id}", api.HandleDeleteURL)
		ar.Get("/urls/{id}/stats", api.HandleStats)
		ar.Get("/stats/overview", api.HandleOverview)
		ar.Post("/auth/register", api.HandleRegister)
		ar.Post("/auth/login", api.HandleLogin)
		ar.Post("/auth/logout", api.HandleLogout)
		ar.Get("/auth/me", api.HandleMe)
		ar.Put("/auth/password", api.HandleChangePassword)
	})

	// 4. static SPA
	if cfg.StaticDir != "" {
		fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir(filepath.Join(cfg.StaticDir, "assets"))))
		r.Method(http.MethodGet, "/assets/*", cacheable(fileServer))
		r.Method(http.MethodGet, "/favicon.ico", cacheable(file(filePath(cfg.StaticDir, "favicon.ico"))))
		r.Method(http.MethodGet, "/favicon.svg", cacheable(file(filePath(cfg.StaticDir, "favicon.svg"))))
		r.Method(http.MethodGet, "/robots.txt", cacheable(file(filePath(cfg.StaticDir, "robots.txt"))))

		// 5. SPA routes → index.html
		for _, p := range []string{"/", "/login", "/register", "/dashboard", "/settings"} {
			r.Method(http.MethodGet, p, spaHandler(cfg.StaticDir))
		}
		r.Method(http.MethodGet, "/urls/{id}", spaHandler(cfg.StaticDir))
	}

	// 6. redirect — single path segment only (chi {code} never matches "/").
	r.Method(http.MethodGet, "/{code}", http.HandlerFunc(api.HandleRedirect))
	r.Method(http.MethodHead, "/{code}", http.HandlerFunc(api.HandleRedirectNoClick))

	// 7. everything else → 404 page
	r.NotFound(api.HandleNotFound)

	return r
}

func filePath(base, name string) string { return filepath.Join(base, name) }

func file(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	})
}

// cacheable marks static assets as immutable for one year; index gets no-cache.
func cacheable(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

func spaHandler(dir string) http.Handler {
	indexPath := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, indexPath)
	})
}
