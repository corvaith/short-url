// Package middleware provides HTTP middlewares: request id, recover, logging,
// security headers, CORS, rate limiting, sessions and CSRF checks.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

type ctxKey int

const (
	ctxRequestID ctxKey = iota
	ctxUserToken
	ctxUserID
	ctxSessionID
)

// RequestID returns the request id stored by the RequestID middleware.
func RequestID(r *http.Request) string {
	if v, ok := r.Context().Value(ctxRequestID).(string); ok {
		return v
	}
	return ""
}

// WithUser stores the authenticated user + session ids in the request ctx.
func WithUser(r *http.Request, userID, sessionID, token string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxSessionID, sessionID)
	ctx = context.WithValue(ctx, ctxUserToken, token)
	return r.WithContext(ctx)
}

// UserID returns the authenticated user id ("" when anonymous).
func UserID(r *http.Request) string {
	if v, ok := r.Context().Value(ctxUserID).(string); ok {
		return v
	}
	return ""
}

// SessionID returns the current session id ("" when anonymous).
func SessionID(r *http.Request) string {
	if v, ok := r.Context().Value(ctxSessionID).(string); ok {
		return v
	}
	return ""
}

// Token returns the raw session token ("" when anonymous).
func Token(r *http.Request) string {
	if v, ok := r.Context().Value(ctxUserToken).(string); ok {
		return v
	}
	return ""
}

// RequestIDMiddleware attaches a request id (honoring an upstream one) and
// echoes it back as X-Request-ID.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > 64 {
			b := make([]byte, 12)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxRequestID, id)))
	})
}

// statusRecorder captures the response status for logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.status == 0 {
		sr.status = http.StatusOK
	}
	n, err := sr.ResponseWriter.Write(b)
	sr.bytes += n
	return n, err
}

// Logging logs one line per request in slog format.
func Logging(log *slog.Logger, trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sr := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(sr, r)
			if sr.status == 0 {
				sr.status = http.StatusOK
			}
			log.LogAttrs(r.Context(), slog.LevelInfo, "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sr.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.Int("bytes", sr.bytes),
				slog.String("ip", ClientIP(r, trustedProxies)),
			)
		})
	}
}

// Recoverer converts panics into 500 responses.
func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", "err", rec, "path", r.URL.Path)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders sets the security headers from PRD §13.2. errorCSP allows the
// minimal inline-CSS error pages (self-hosted HTML).
func SecureHeaders(production bool, errorCSP bool) func(http.Handler) http.Handler {
	csp := "default-src 'self'; script-src 'self'; style-src 'self'" +
		stringsIf(errorCSP, " 'unsafe-inline'") +
		"; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", csp)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if production {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func stringsIf(cond bool, s string) string {
	if cond {
		return s
	}
	return ""
}

// NoIndex adds X-Robots-Tag: noindex to non-public responses.
func NoIndex(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}

// NoStore disables caching for API responses.
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// ClientIP resolves the client IP, only honoring X-Forwarded-For when the
// direct peer is a trusted proxy.
func ClientIP(r *http.Request, trusted []*net.IPNet) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer != nil && containsIP(trusted, peer) {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			// leftmost = original client
			cand := strings.TrimSpace(parts[0])
			if ip := net.ParseIP(cand); ip != nil {
				return ip.String()
			}
		}
	}
	return peer.String()
}

func containsIP(nets []*net.IPNet, ip net.IP) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ParseTrustedProxies converts CIDR strings to IPNets; single IPs are treated
// as /32 or /128.
func ParseTrustedProxies(list []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(list))
	for _, s := range list {
		if !strings.Contains(s, "/") {
			if ip := net.ParseIP(s); ip != nil {
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				s = s + fmtSlash(bits)
			}
		}
		if _, n, err := net.ParseCIDR(s); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func fmtSlash(bits int) string {
	b := []byte{'/', byte('0' + bits/10), byte('0' + bits%10)}
	if bits < 10 {
		return "/" + string([]byte{byte('0' + bits)})
	}
	return string(b)
}
