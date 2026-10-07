//go:build integration

// Package integration holds end-to-end tests that run against a real Postgres
// database. They are guarded by the `integration` build tag AND require
// APP_ENV=test plus TEST_DATABASE_URL, so they can never touch production.
//
// Run with:
//
//	APP_ENV=test TEST_DATABASE_URL=... go test -tags=integration ./test/...
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"shorturl/internal/config"
	"shorturl/internal/handler"
	"shorturl/internal/middleware"
	"shorturl/internal/repository"
	"shorturl/internal/router"
	"shorturl/internal/service"

	"log/slog"
)

// newTestServer builds the full HTTP stack against TEST_DATABASE_URL.
func newTestServer(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("APP_ENV") != "test" {
		t.Skip("integration tests require APP_ENV=test")
	}
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	cfg := &config.Config{
		DatabaseURL:     dbURL,
		BaseURL:         "http://localhost",
		AppEnv:          "test",
		SessionTTL:      time.Hour,
		CacheSize:       1000,
		CacheTTL:        time.Second,
		CacheNegTTL:     time.Second,
		MaxURLsUser:     1000,
		ClickFlushEvery: 50 * time.Millisecond,
		RateRedirectIP:  config.Rate{PerMinute: 100000, Burst: 100000},
		RateCreateAnon:  config.Rate{PerMinute: 100000, Burst: 100000},
		RateCreateUser:  config.Rate{PerMinute: 100000, Burst: 100000},
		RateLoginIP:     config.Rate{PerMinute: 100000, Burst: 100000},
		RateLoginFail:   config.Rate{PerMinute: 100000, Burst: 100000},
		RateRegisterIP:  config.Rate{PerMinute: 100000, Burst: 100000},
		RateAPIGeneral:  config.Rate{PerMinute: 100000, Burst: 100000},
		BlockedHosts:    map[string]struct{}{},
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})).With("test", t.Name())
	users := repository.NewUsers(pool)
	sessions := repository.NewSessions(pool)
	urls := repository.NewURLs(pool)
	authSvc := service.NewAuthService(cfg, users, sessions, log)
	urlSvc := service.NewURLService(cfg, urls, log)
	statsSvc := service.NewStatsService(urls)
	lim := middleware.NewLimiter()
	ctx, cancel := context.WithCancel(context.Background())
	urlSvc.StartClickFlusher(ctx)
	t.Cleanup(cancel)
	oauthSvc := service.NewOAuthService(cfg, users, authSvc, log)
	api := handler.NewAPI(cfg, urlSvc, authSvc, oauthSvc, statsSvc, lim, nil, func(ctx context.Context) bool { return pool.Ping(ctx) == nil })
	root := router.New(cfg, api, log, nil)
	srv := httptest.NewServer(root)
	t.Cleanup(func() { srv.Close(); pool.Close() })
	return srv, pool
}

// Fixture cleanup: wipe tables between tests (test DB only).
func clean(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE url_daily_clicks, short_urls, sessions, users CASCADE")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateAndRedirect(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)

	code := createURL(t, srv, `{"target_url":"https://example.com/a?b=1"}`, nil)
	// AT-03: 201 + short_url shape
	if len(code) != 7 {
		t.Fatalf("expected 7-char code, got %q", code)
	}
	// AT-04: redirect
	res := get(t, srv, "/"+code, nil, false)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("want 302, got %d", res.StatusCode)
	}
	if loc := res.Header.Get("Location"); loc != "https://example.com/a?b=1" {
		t.Fatalf("bad Location: %q", loc)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("want no-store, got %q", cc)
	}
}

func TestAnonymousAliasForbidden(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	// AT-10
	res := post(t, srv, "/api/v1/urls", `{"target_url":"https://ok.example","alias":"myalias"}`, nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", res.StatusCode)
	}
	var env struct {
		Error struct{ Code string } `json:"error"`
	}
	decode(t, res, &env)
	if env.Error.Code != "alias_requires_login" {
		t.Fatalf("want alias_requires_login, got %q", env.Error.Code)
	}
}

func TestAliasAndAuth(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	jar := newJar(t)

	// register (AT-16 step 1)
	res := post(t, srv, "/api/v1/auth/register", `{"email":"a@example.com","password":"password123"}`, jar)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", res.StatusCode)
	}
	t.Logf("set-cookie: %v", res.Header.Values("Set-Cookie"))
	u, _ := url.Parse(srv.URL)
	t.Logf("jar cookies: %v", jar.Cookies(u))
	// me (AT-16 step 2)
	if r := get(t, srv, "/api/v1/auth/me", jar, false); r.StatusCode != 200 {
		t.Fatalf("want 200 me, got %d", r.StatusCode)
	}
	// create with alias (AT-07)
	code := createURL(t, srv, `{"target_url":"https://go.example/x","alias":"my-project"}`, jar)
	if code != "my-project" {
		t.Fatalf("want my-project, got %q", code)
	}
	if r := get(t, srv, "/my-project", nil, false); r.StatusCode != 302 {
		t.Fatalf("want 302 for alias, got %d", r.StatusCode)
	}
	// duplicate alias (AT-08)
	res = post(t, srv, "/api/v1/urls", `{"target_url":"https://go.example/y","alias":"my-project"}`, jar)
	if res.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("want 409, got %d body=%s", res.StatusCode, b)
	}
	// logout (AT-16 step 3) then me -> 401 (step 4)
	if r := post(t, srv, "/api/v1/auth/logout", "", jar); r.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204 logout, got %d", r.StatusCode)
	}
	if r := get(t, srv, "/api/v1/auth/me", jar, false); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 after logout, got %d", r.StatusCode)
	}
}

func TestInvalidAlias(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	jar := newJar(t)
	post(t, srv, "/api/v1/auth/register", `{"email":"b@example.com","password":"password123"}`, jar)
	for _, a := range []string{"ab", "a b", "-abc", "api", "LOGIN"} {
		res := post(t, srv, "/api/v1/urls", fmt.Sprintf(`{"target_url":"https://ok.example","alias":%q}`, a), jar)
		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("alias %q: want 422, got %d", a, res.StatusCode)
		}
	}
}

func TestOwnershipIsolation(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	jarA, jarB := newJar(t), newJar(t)
	post(t, srv, "/api/v1/auth/register", `{"email":"a2@example.com","password":"password123"}`, jarA)
	post(t, srv, "/api/v1/auth/register", `{"email":"b2@example.com","password":"password123"}`, jarB)

	code := createURL(t, srv, `{"target_url":"https://a-only.example/secret"}`, jarA)
	// find id as A
	res := get(t, srv, "/api/v1/urls?limit=10", jarA, false)
	var list struct {
		Items []struct{ ID string } `json:"items"`
	}
	decode(t, res, &list)
	if len(list.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(list.Items))
	}
	id := list.Items[0].ID
	_ = code
	// AT-15: B must get 404
	if r := get(t, srv, "/api/v1/urls/"+id, jarB, false); r.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 cross-user get, got %d", r.StatusCode)
	}
	if r := patch(t, srv, "/api/v1/urls/"+id, `{"is_active":false}`, jarB); r.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 cross-user patch, got %d", r.StatusCode)
	}
	if r := del(t, srv, "/api/v1/urls/"+id, jarB); r.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 cross-user delete, got %d", r.StatusCode)
	}
}

func TestClicksAndStats(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	jar := newJar(t)
	post(t, srv, "/api/v1/auth/register", `{"email":"c@example.com","password":"password123"}`, jar)
	code := createURL(t, srv, `{"target_url":"https://click.example/x","alias":"clicks"}`, jar)
	for i := 0; i < 3; i++ {
		get(t, srv, "/clicks", nil, false)
	}
	// wait for buffer flush
	time.Sleep(300 * time.Millisecond)

	res := get(t, srv, "/api/v1/urls?limit=10", jar, false)
	var list struct {
		Items []struct {
			ID     string `json:"id"`
			Clicks int64  `json:"clicks"`
		} `json:"items"`
	}
	decode(t, res, &list)
	if len(list.Items) != 1 || list.Items[0].Clicks != 3 {
		t.Fatalf("want 3 clicks, got %+v", list.Items)
	}
	_ = code
	res = get(t, srv, "/api/v1/urls/"+list.Items[0].ID+"/stats?days=7", jar, false)
	var st struct {
		Days        int   `json:"days"`
		RangeClicks int64 `json:"range_clicks"`
		Series      []struct {
			Date   string `json:"date"`
			Clicks int64  `json:"clicks"`
		} `json:"series"`
	}
	decode(t, res, &st)
	if st.Days != 7 || len(st.Series) != 7 {
		t.Fatalf("want 7 series entries, got %d (days %d)", len(st.Series), st.Days)
	}
	if st.RangeClicks != 3 {
		t.Fatalf("want range_clicks 3, got %d", st.RangeClicks)
	}
}

func TestMalformedAndLimits(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	// AT-14: malformed codes -> 404 without DB
	for _, c := range []string{"!!bad", "ab"} {
		if r := get(t, srv, "/"+c, nil, false); r.StatusCode != http.StatusNotFound {
			t.Fatalf("code %q: want 404, got %d", c, r.StatusCode)
		}
	}
	// AT-20
	// broken JSON
	if r := postRaw(t, srv, "/api/v1/urls", "{not json", "application/json", nil); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("broken json: want 400, got %d", r.StatusCode)
	}
	// unknown field
	if r := postRaw(t, srv, "/api/v1/urls", `{"target_url":"https://ok.example","bogus":1}`, "application/json", nil); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown field: want 400, got %d", r.StatusCode)
	}
	// bad content type
	if r := postRaw(t, srv, "/api/v1/urls", `{"target_url":"https://ok.example"}`, "text/plain", nil); r.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("bad ct: want 415, got %d", r.StatusCode)
	}
	// too large
	big := bytes.Repeat([]byte("x"), 20*1024)
	body := append([]byte(`{"target_url":"https://ok.example","alias":"`), big...)
	body = append(body, []byte(`"}`)...)
	if r := postRaw(t, srv, "/api/v1/urls", string(body), "application/json", nil); r.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("too large: want 413, got %d", r.StatusCode)
	}
}

func TestExpiredAndDisabled(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	jar := newJar(t)
	post(t, srv, "/api/v1/auth/register", `{"email":"d@example.com","password":"password123"}`, jar)
	exp := time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
	code := createURL(t, srv, fmt.Sprintf(`{"target_url":"https://exp.example","alias":"expire","expires_at":%q}`, exp), jar)
	// manually force expiry in DB
	if _, err := pool.Exec(context.Background(), "UPDATE short_urls SET expires_at = now() - interval '1 minute' WHERE code = $1", code); err != nil {
		t.Fatal(err)
	}
	// AT-11: 410 (cache TTL 1s in test; sleep to bypass)
	time.Sleep(1200 * time.Millisecond)
	if r := get(t, srv, "/expire", nil, false); r.StatusCode != http.StatusGone {
		t.Fatalf("want 410, got %d", r.StatusCode)
	}
	// disable (AT-12)
	code2 := createURL(t, srv, `{"target_url":"https://dis.example","alias":"disableme"}`, jar)
	res := get(t, srv, "/api/v1/urls?limit=10", jar, false)
	var list struct {
		Items []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		} `json:"items"`
	}
	decode(t, res, &list)
	var id string
	for _, it := range list.Items {
		if it.Code == code2 {
			id = it.ID
		}
	}
	if id == "" {
		t.Fatal("created url not found in list")
	}
	if r := patch(t, srv, "/api/v1/urls/"+id, `{"is_active":false}`, jar); r.StatusCode != 200 {
		t.Fatalf("patch: want 200, got %d", r.StatusCode)
	}
	if r := get(t, srv, "/disableme", nil, false); r.StatusCode != http.StatusNotFound {
		t.Fatalf("disabled: want 404, got %d", r.StatusCode)
	}
	// re-enable -> 302
	if r := patch(t, srv, "/api/v1/urls/"+id, `{"is_active":true}`, jar); r.StatusCode != 200 {
		t.Fatalf("re-enable patch: want 200, got %d", r.StatusCode)
	}
	if r := get(t, srv, "/disableme", nil, false); r.StatusCode != http.StatusFound {
		t.Fatalf("re-enabled: want 302, got %d", r.StatusCode)
	}
}

func TestCursorPagination(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	jar := newJar(t)
	post(t, srv, "/api/v1/auth/register", `{"email":"e@example.com","password":"password123"}`, jar)
	for i := 0; i < 45; i++ {
		code := createURL(t, srv, fmt.Sprintf(`{"target_url":"https://page.example/%d"}`, i), jar)
		if code == "" {
			t.Fatal("create failed")
		}
	}
	seen := map[string]bool{}
	cursor := ""
	pages := 0
	for {
		url := "/api/v1/urls?limit=20"
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		res := get(t, srv, url, jar, false)
		var page struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
			NextCursor *string `json:"next_cursor"`
		}
		decode(t, res, &page)
		pages++
		for _, it := range page.Items {
			if seen[it.ID] {
				t.Fatalf("duplicate id across pages: %s", it.ID)
			}
			seen[it.ID] = true
		}
		if page.NextCursor == nil || *page.NextCursor == "" {
			break
		}
		cursor = *page.NextCursor
	}
	if pages != 3 || len(seen) != 45 {
		t.Fatalf("AT-21: want 3 pages / 45 items, got %d pages / %d items", pages, len(seen))
	}
	// bad cursor -> 400 (AT-20)
	if r := get(t, srv, "/api/v1/urls?cursor=!!!bad", jar, false); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad cursor: want 400, got %d", r.StatusCode)
	}
}

func TestLoginErrorsIdentical(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	post(t, srv, "/api/v1/auth/register", `{"email":"f@example.com","password":"password123"}`, nil)
	r1 := post(t, srv, "/api/v1/auth/login", `{"email":"f@example.com","password":"wrongpass1"}`, nil)
	r2 := post(t, srv, "/api/v1/auth/login", `{"email":"nobody@example.com","password":"wrongpass1"}`, nil)
	if r1.StatusCode != 401 || r2.StatusCode != 401 {
		t.Fatalf("want both 401, got %d and %d", r1.StatusCode, r2.StatusCode)
	}
	var e1, e2 struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	decode(t, r1, &e1)
	decode(t, r2, &e2)
	if e1.Error.Code != e2.Error.Code || e1.Error.Message != e2.Error.Message {
		t.Fatalf("AT-17: responses differ: %+v vs %+v", e1.Error, e2.Error)
	}
}

func TestPasswordChangeRevokesSessions(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	jarA, jarB := newJar(t), newJar(t)
	post(t, srv, "/api/v1/auth/register", `{"email":"g@example.com","password":"password123"}`, jarA)
	post(t, srv, "/api/v1/auth/login", `{"email":"g@example.com","password":"password123"}`, jarB)
	if r := put(t, srv, "/api/v1/auth/password", `{"current_password":"password123","new_password":"newpassword456"}`, jarA); r.StatusCode != 204 {
		t.Fatalf("password change: want 204, got %d", r.StatusCode)
	}
	if r := get(t, srv, "/api/v1/auth/me", jarB, false); r.StatusCode != 401 {
		t.Fatalf("AT-18: other session should be invalid, got %d", r.StatusCode)
	}
	if r := get(t, srv, "/api/v1/auth/me", jarA, false); r.StatusCode != 200 {
		t.Fatalf("current session should stay valid, got %d", r.StatusCode)
	}
	if r := post(t, srv, "/api/v1/auth/login", `{"email":"g@example.com","password":"password123"}`, nil); r.StatusCode != 401 {
		t.Fatalf("old password should fail, got %d", r.StatusCode)
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	res := get(t, srv, "/healthz", nil, false)
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "X-Frame-Options", "Permissions-Policy"} {
		if res.Header.Get(h) == "" {
			t.Errorf("AT-23: missing header %s", h)
		}
	}
	// API noindex + no-store
	res = get(t, srv, "/api/v1/stats/overview", nil, false)
	if res.Header.Get("X-Robots-Tag") == "" || res.Header.Get("Cache-Control") != "no-store" {
		t.Error("AT-23: API should have noindex + no-store")
	}
}

func TestOverview(t *testing.T) {
	srv, pool := newTestServer(t)
	clean(t, pool)
	jar := newJar(t)
	post(t, srv, "/api/v1/auth/register", `{"email":"h@example.com","password":"password123"}`, jar)
	createURL(t, srv, `{"target_url":"https://o1.example"}`, jar)
	createURL(t, srv, `{"target_url":"https://o2.example"}`, jar)
	res := get(t, srv, "/api/v1/stats/overview", jar, false)
	var o struct {
		TotalURLs  int64 `json:"total_urls"`
		ActiveURLs int64 `json:"active_urls"`
	}
	decode(t, res, &o)
	if o.TotalURLs != 2 || o.ActiveURLs != 2 {
		t.Fatalf("want 2 total / 2 active, got %d / %d", o.TotalURLs, o.ActiveURLs)
	}
}

// ---- helpers ----

func createURL(t *testing.T, srv *httptest.Server, body string, jar http.CookieJar) string {
	t.Helper()
	res := post(t, srv, "/api/v1/urls", body, jar)
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("create: want 201, got %d body=%s", res.StatusCode, b)
	}
	var out struct {
		Code string `json:"code"`
	}
	decode(t, res, &out)
	return out.Code
}

func newJar(t *testing.T) http.CookieJar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return jar
}

func post(t *testing.T, srv *httptest.Server, path, body string, jar http.CookieJar) *http.Response {
	return postRaw(t, srv, path, body, "application/json", jar)
}

func postRaw(t *testing.T, srv *httptest.Server, path, body, ct string, jar http.CookieJar) *http.Response {
	t.Helper()
	return do(t, srv, http.MethodPost, path, body, ct, jar)
}

func put(t *testing.T, srv *httptest.Server, path, body string, jar http.CookieJar) *http.Response {
	t.Helper()
	return do(t, srv, http.MethodPut, path, body, "application/json", jar)
}

func patch(t *testing.T, srv *httptest.Server, path, body string, jar http.CookieJar) *http.Response {
	t.Helper()
	return do(t, srv, http.MethodPatch, path, body, "application/json", jar)
}

func del(t *testing.T, srv *httptest.Server, path string, jar http.CookieJar) *http.Response {
	t.Helper()
	return do(t, srv, http.MethodDelete, path, "", "", jar)
}

func get(t *testing.T, srv *httptest.Server, path string, jar http.CookieJar, follow bool) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	if !follow {
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func do(t *testing.T, srv *httptest.Server, method, path, body, ct string, jar http.CookieJar) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewBufferString(body)
	}
	req, err := http.NewRequest(method, srv.URL+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func decode(t *testing.T, res *http.Response, dst any) {
	t.Helper()
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(b, dst); err != nil {
		t.Fatalf("decode error: %v body=%s", err, b)
	}
}
