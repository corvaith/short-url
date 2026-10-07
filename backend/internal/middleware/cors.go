package middleware

import (
	"net/http"
	"strings"
)

// CORS applies an allowlist-based CORS policy. In production the app is
// same-origin so this is usually a no-op; in dev it lets the Vite server call
// the API. Wildcards are never combined with credentials.
func CORS(allowed []string) func(http.Handler) http.Handler {
	set := map[string]struct{}{}
	for _, o := range allowed {
		set[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := set[strings.TrimRight(origin, "/")]; ok {
					h := w.Header()
					h.Set("Access-Control-Allow-Origin", origin)
					h.Set("Access-Control-Allow-Credentials", "true")
					h.Set("Vary", "Origin")
					h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, HEAD, OPTIONS")
					h.Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
					h.Set("Access-Control-Max-Age", "600")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CSRF enforces that unsafe methods (a) are application/json (already checked
// by the JSON decoder for those routes) and (b) if an Origin header is
// present, it matches the app origin or the allowlist.
func CSRF(baseURL string, allowed []string) func(http.Handler) http.Handler {
	allowedSet := map[string]struct{}{strings.TrimRight(baseURL, "/"): {}}
	for _, o := range allowed {
		allowedSet[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
				origin := r.Header.Get("Origin")
				if origin != "" {
					if _, ok := allowedSet[strings.TrimRight(origin, "/")]; !ok {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
						w.WriteHeader(http.StatusForbidden)
						_, _ = w.Write([]byte(`{"error":{"code":"forbidden","message":"Origin not allowed."}}`))
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
