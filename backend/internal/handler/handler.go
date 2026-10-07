// Package handler implements HTTP handlers. No SQL here; handlers decode,
// call services, and write responses.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"shorturl/internal/config"
	"shorturl/internal/middleware"
	"shorturl/internal/model"
	"shorturl/internal/repository"
	"shorturl/internal/service"
	"shorturl/internal/validate"
)

// API is the collection of HTTP handlers.
type API struct {
	cfg            *config.Config
	urls           *service.URLService
	auth           *service.AuthService
	oauth          *service.OAuthService
	stats          *service.StatsService
	lim            *middleware.Limiter
	trustedProxies []*net.IPNet
	ready          func(ctx context.Context) bool
}

// NewAPI builds the handler collection.
func NewAPI(cfg *config.Config, urls *service.URLService, auth *service.AuthService, oauth *service.OAuthService, stats *service.StatsService, lim *middleware.Limiter, trustedProxies []*net.IPNet, ready func(ctx context.Context) bool) *API {
	return &API{cfg: cfg, urls: urls, auth: auth, oauth: oauth, stats: stats, lim: lim, trustedProxies: trustedProxies, ready: ready}
}

// ---- OAuth (GitHub / Discord, option B alongside password auth) ----

var oauthProviders = map[string]bool{"github": true, "discord": true}

// HandleOAuthStart redirects the browser to the provider consent screen.
// GET /api/v1/auth/oauth/{provider}/start?next=/dashboard
func (a *API) HandleOAuthStart(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !oauthProviders[provider] || a.oauth == nil || a.oauth.ClientID(provider) == "" {
		a.serveErrorPage(w, r, http.StatusNotFound)
		return
	}
	next := r.URL.Query().Get("next")
	state := a.oauth.NewState(provider, next)
	target, err := a.oauth.AuthCodeURL(provider, state)
	if err != nil {
		a.serveErrorPage(w, r, http.StatusNotFound)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

// HandleOAuthCallback exchanges the code, signs the user in, and redirects to
// the SPA. Errors land on /login?error=... for user-facing messaging.
// GET /api/v1/auth/oauth/{provider}/callback?code=..&state=..
func (a *API) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	fail := func(code string) {
		http.Redirect(w, r, "/login?error="+url.QueryEscape(code), http.StatusFound)
	}
	if !oauthProviders[provider] || a.oauth == nil {
		fail("oauth_unavailable")
		return
	}
	if e := r.URL.Query().Get("error"); e != "" {
		fail("oauth_denied")
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		fail("oauth_invalid")
		return
	}
	next, ok := a.oauth.ConsumeState(provider, state)
	if !ok {
		fail("oauth_state")
		return
	}
	u, token, err := a.oauth.LoginWithCode(r.Context(), provider, code)
	if err != nil {
		if errors.Is(err, service.ErrOAuthExchange) {
			fail("oauth_exchange")
			return
		}
		fail("oauth_error")
		return
	}
	a.setSessionCookie(w, token, int(a.cfg.SessionTTL.Seconds()))
	if next == "" {
		next = "/dashboard"
	}
	_ = u
	http.Redirect(w, r, next, http.StatusFound)
}

// ---- error envelope ----

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: apiError{Code: code, Message: message, Fields: fields}})
}

const msgGeneric = "Something went wrong. Please try again."

func writeInternal(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, "internal_error", msgGeneric, nil)
}

// writeServiceError maps a service error to the API error response.
func (a *API) writeServiceError(w http.ResponseWriter, err error) {
	var ve *service.ValidationError
	switch {
	case errors.As(err, &ve):
		fields := map[string]string{}
		for _, f := range ve.Fields {
			fields[f.Field] = f.Detail
		}
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Please check the highlighted fields.", fields)
	case errors.Is(err, service.ErrAliasTaken):
		writeError(w, http.StatusConflict, "alias_taken", "This alias is already taken.", nil)
	case errors.Is(err, service.ErrAliasRequiresLogin):
		writeError(w, http.StatusForbidden, "alias_requires_login", "Log in to use a custom alias.", nil)
	case errors.Is(err, service.ErrURLLimitReached):
		writeError(w, http.StatusForbidden, "url_limit_reached", "You have reached the maximum number of links.", nil)
	case errors.Is(err, service.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "An account with this email already exists.", nil)
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.", nil)
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Not found.", nil)
	default:
		writeInternal(w)
	}
}

// ---- strict JSON body decoding ----

const maxBodyBytes = 16 * 1024

// decodeJSON strictly decodes r's JSON body into dst (DisallowUnknownFields).
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodDelete {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json.", nil)
			return false
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large.", nil)
			return false
		}
		writeError(w, http.StatusBadRequest, "bad_request", "Request body is not valid JSON.", nil)
		return false
	}
	// Reject trailing content.
	if dec.More() {
		writeError(w, http.StatusBadRequest, "bad_request", "Request body must contain a single JSON object.", nil)
		return false
	}
	return true
}

// ---- auth helpers ----

const cookieName = "session"

func (a *API) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

// currentUser resolves the session cookie to a user. Returns the user and the
// session id (second value empty when anonymous).
func (a *API) currentUser(r *http.Request) (*model.User, string) {
	token := ""
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		token = c.Value
	}
	if token == "" {
		return nil, ""
	}
	u, sess, err := a.auth.Authenticate(r.Context(), token)
	if err != nil {
		return nil, ""
	}
	return u, sess.ID
}

// requireAuth returns the user or writes 401 and returns nil.
func (a *API) requireAuth(w http.ResponseWriter, r *http.Request) *model.User {
	u, _ := a.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "You need to log in.", nil)
		return nil
	}
	return u
}

// ---- auth handlers ----

func (a *API) handleRegister(w http.ResponseWriter, r *http.Request) {
	ip := middleware.ClientIP(r, a.trustedProxies)
	if ok, wait := a.lim.Allow("reg:"+ip, a.cfg.RateRegisterIP.PerMinute, a.cfg.RateRegisterIP.Burst); !ok {
		a.writeTooMany(w, wait)
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	u, token, err := a.auth.Register(r.Context(), body.Email, body.Password)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	a.setSessionCookie(w, token, int(a.cfg.SessionTTL.Seconds()))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"user": u.ToDTO()})
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := middleware.ClientIP(r, a.trustedProxies)
	if ok, wait := a.lim.Allow("login:"+ip, a.cfg.RateLoginIP.PerMinute, a.cfg.RateLoginIP.Burst); !ok {
		a.writeTooMany(w, wait)
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	emailLower := strings.ToLower(strings.TrimSpace(body.Email))
	failKey := "loginfail:" + ip + ":" + emailLower
	if a.lim.Failures(failKey, 15*time.Minute) >= a.cfg.RateLoginFail.PerMinute {
		a.writeTooMany(w, 15*time.Minute)
		return
	}
	u, token, err := a.auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			a.lim.Fail(failKey)
		}
		a.writeServiceError(w, err)
		return
	}
	a.lim.ClearFailures(failKey)
	a.setSessionCookie(w, token, int(a.cfg.SessionTTL.Seconds()))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"user": u.ToDTO()})
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		_ = a.auth.Logout(r.Context(), c.Value)
	}
	a.setSessionCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	u := a.requireAuth(w, r)
	if u == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"user": u.ToDTO()})
}

func (a *API) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u, sessID := a.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "You need to log in.", nil)
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := a.auth.ChangePassword(r.Context(), u.ID, sessID, body.CurrentPassword, body.NewPassword); err != nil {
		a.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) writeTooMany(w http.ResponseWriter, wait time.Duration) {
	secs := int(wait.Seconds())
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	writeError(w, http.StatusTooManyRequests, "rate_limited", fmt.Sprintf("Too many requests. Try again in %d seconds.", secs), nil)
}

// ---- URL handlers ----

func (a *API) handleCreateURL(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	var body struct {
		TargetURL string     `json:"target_url"`
		Alias     string     `json:"alias"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	// D-17: frontend adds https:// when missing; server stays strict on the
	// actual content. Here: also normalize a bare scheme-less host the same
	// way the frontend would have (server re-validates fully afterwards).
	if body.TargetURL != "" && !strings.Contains(body.TargetURL, "://") {
		body.TargetURL = "https://" + body.TargetURL
	}
	userID := ""
	if user != nil {
		userID = user.ID
		// rate limit per user
		if ok, wait := a.lim.Allow("createu:"+userID, a.cfg.RateCreateUser.PerMinute, a.cfg.RateCreateUser.Burst); !ok {
			a.writeTooMany(w, wait)
			return
		}
	} else {
		ip := middleware.ClientIP(r, a.trustedProxies)
		if ok, wait := a.lim.Allow("createa:"+ip, a.cfg.RateCreateAnon.PerMinute, a.cfg.RateCreateAnon.Burst); !ok {
			a.writeTooMany(w, wait)
			return
		}
	}
	m, err := a.urls.Create(r.Context(), userID, body.TargetURL, strings.TrimSpace(body.Alias), body.ExpiresAt)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	dto := m.ToDTO(a.cfg.BaseURL, time.Now().UTC())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(dto)
}

func (a *API) handleListURLs(w http.ResponseWriter, r *http.Request) {
	u := a.requireAuth(w, r)
	if u == nil {
		return
	}
	if ok, wait := a.lim.Allow("api:"+u.ID, a.cfg.RateAPIGeneral.PerMinute, a.cfg.RateAPIGeneral.Burst); !ok {
		a.writeTooMany(w, wait)
		return
	}
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			writeError(w, http.StatusBadRequest, "bad_request", "limit must be an integer between 1 and 100.", nil)
			return
		}
		limit = n
	}
	var cursor *repository.Cursor
	if v := r.URL.Query().Get("cursor"); v != "" {
		c, err := repository.DecodeCursor(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "cursor is not valid.", nil)
			return
		}
		cursor = c
	}
	page, err := a.urls.List(r.Context(), u.ID, limit, cursor)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	items := make([]model.URLDTO, 0, len(page.Items))
	for _, m := range page.Items {
		items = append(items, m.ToDTO(a.cfg.BaseURL, time.Now().UTC()))
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "next_cursor": next})
}

func (a *API) handleGetURL(w http.ResponseWriter, r *http.Request) {
	u := a.requireAuth(w, r)
	if u == nil {
		return
	}
	m, err := a.urls.Get(r.Context(), u.ID, chi.URLParam(r, "id"))
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(m.ToDTO(a.cfg.BaseURL, time.Now().UTC()))
}

func (a *API) handlePatchURL(w http.ResponseWriter, r *http.Request) {
	u := a.requireAuth(w, r)
	if u == nil {
		return
	}
	var body struct {
		IsActive *bool `json:"is_active"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.IsActive == nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Please check the highlighted fields.", map[string]string{"is_active": "is_active is required."})
		return
	}
	m, err := a.urls.SetActive(r.Context(), u.ID, chi.URLParam(r, "id"), *body.IsActive)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(m.ToDTO(a.cfg.BaseURL, time.Now().UTC()))
}

func (a *API) handleDeleteURL(w http.ResponseWriter, r *http.Request) {
	u := a.requireAuth(w, r)
	if u == nil {
		return
	}
	if err := a.urls.Delete(r.Context(), u.ID, chi.URLParam(r, "id")); err != nil {
		a.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleStats(w http.ResponseWriter, r *http.Request) {
	u := a.requireAuth(w, r)
	if u == nil {
		return
	}
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || (n != 7 && n != 30) {
			writeError(w, http.StatusBadRequest, "bad_request", "days must be 7 or 30.", nil)
			return
		}
		days = n
	}
	resp, err := a.stats.GetStats(r.Context(), u.ID, chi.URLParam(r, "id"), days, time.Now().UTC())
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func (a *API) handleOverview(w http.ResponseWriter, r *http.Request) {
	u := a.requireAuth(w, r)
	if u == nil {
		return
	}
	o, err := a.stats.Overview(r.Context(), u.ID)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(o)
}

// ---- redirect handler (critical path) ----

func (a *API) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	// 1. shape check before any DB access
	if !validate.CodeRe.MatchString(code) {
		a.serveErrorPage(w, r, http.StatusNotFound)
		return
	}
	// rate limit per IP
	ip := middleware.ClientIP(r, a.trustedProxies)
	if ok, wait := a.lim.Allow("redir:"+ip, a.cfg.RateRedirectIP.PerMinute, a.cfg.RateRedirectIP.Burst); !ok {
		a.writeTooMany(w, wait)
		return
	}
	target, err := a.urls.Resolve(r.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			a.serveErrorPage(w, r, http.StatusNotFound)
		case errors.Is(err, service.ErrGone):
			a.serveErrorPage(w, r, http.StatusGone)
		default:
			writeInternal(w)
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	http.Redirect(w, r, target, http.StatusFound)
}

// handleRedirectNoClick is HEAD /{code}: same resolution, no click recorded.
func (a *API) handleRedirectNoClick(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if !validate.CodeRe.MatchString(code) {
		a.serveErrorPage(w, r, http.StatusNotFound)
		return
	}
	target, err := a.urls.ResolveNoClick(r.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			a.serveErrorPage(w, r, http.StatusNotFound)
		case errors.Is(err, service.ErrGone):
			a.serveErrorPage(w, r, http.StatusGone)
		default:
			writeInternal(w)
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusFound)
}

// ---- error pages (D-09) ----

func (a *API) serveErrorPage(w http.ResponseWriter, r *http.Request, status int) {
	title, msg := "Page not found", "The link you followed does not exist or is no longer available."
	if status == http.StatusGone {
		title, msg = "Link expired", "This short link has expired and is no longer available."
	}
	// JSON variant for API clients
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
		code := "not_found"
		if status == http.StatusGone {
			code = "gone"
		}
		writeError(w, status, code, msg, nil)
		return
	}
	body := fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>%d %s — %s</title>
<style>body{font-family:system-ui,sans-serif;background:#F6F7F9;color:#12161C;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}main{background:#fff;border:1px solid #D9DEE5;border-radius:16px;padding:2rem;max-width:24rem;text-align:center}h1{font-size:1.25rem;margin:0 0 .5rem}p{color:#566070;font-size:.9375rem;line-height:1.5;margin:0 0 1.25rem}a{color:#4F46E5;font-weight:600;text-decoration:none}</style>
</head><body><main><h1>%s</h1><p>%s</p><a href="/">Go to homepage</a></main></body></html>`,
		status, html.EscapeString(title), html.EscapeString(a.cfg.BaseURL), html.EscapeString(title), html.EscapeString(msg))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// health/ready
func (a *API) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (a *API) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r)
	defer cancel()
	if a.ready != nil && !a.ready(ctx) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func contextWithTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 2*time.Second)
}
