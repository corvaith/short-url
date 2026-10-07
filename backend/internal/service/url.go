// Package service implements business rules on top of repositories.
package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"shorturl/internal/config"
	"shorturl/internal/model"
	"shorturl/internal/repository"
	"shorturl/internal/validate"
)

// Business errors surfaced by services; handlers map them to API responses.
var (
	ErrAliasTaken         = errors.New("alias_taken")
	ErrAliasRequiresLogin = errors.New("alias_requires_login")
	ErrURLLimitReached    = errors.New("url_limit_reached")
	ErrEmailTaken         = errors.New("email_taken")
	ErrInvalidCredentials = errors.New("invalid_credentials")
	ErrValidation         = errors.New("validation_error")
	ErrNotFound           = errors.New("not_found")
	ErrGone               = errors.New("gone")
	ErrRateLimited        = errors.New("rate_limited")
	ErrInternal           = errors.New("internal_error")
)

// FieldError carries a per-field validation message.
type FieldError struct {
	Field  string
	Detail string
}

// ValidationError bundles multiple field errors.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return "validation_error" }

// Add appends a field error.
func (e *ValidationError) Add(field, detail string) {
	e.Fields = append(e.Fields, FieldError{Field: field, Detail: detail})
}

// Has reports whether any field error exists.
func (e *ValidationError) Has() bool { return len(e.Fields) > 0 }

const (
	codeLen      = 7
	codeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	maxGenTries  = 5
	dummyHash    = "$argon2id$v=19$m=65536,t=3,p=2$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
)

// URLService handles short URL business logic.
type URLService struct {
	cfg    *config.Config
	urls   *repository.URLs
	cache  *redirectCache
	clicks *ClickBuffer
	log    *slog.Logger
}

// NewURLService wires a URLService.
func NewURLService(cfg *config.Config, urls *repository.URLs, log *slog.Logger) *URLService {
	return &URLService{
		cfg:    cfg,
		urls:   urls,
		cache:  newRedirectCache(cfg.CacheSize, cfg.CacheTTL, cfg.CacheNegTTL),
		clicks: NewClickBuffer(urls, cfg.ClickFlushEvery, log),
		log:    log,
	}
}

// StartClickFlusher launches the background flush loop.
func (s *URLService) StartClickFlusher(ctx context.Context) { s.clicks.Start(ctx) }

// FlushNow forces a click-buffer flush (used at shutdown and in tests).
func (s *URLService) FlushNow(ctx context.Context) error { return s.clicks.Flush(ctx) }

// Create validates and stores a new short URL. userID is "" for anonymous.
func (s *URLService) Create(ctx context.Context, userID, rawURL, alias string, expiresAt *time.Time) (*model.ShortURL, error) {
	ve := &ValidationError{}

	if userID == "" && alias != "" {
		return nil, ErrAliasRequiresLogin
	}

	params := validate.Params{OwnHost: s.cfg.OwnHost(), BlockedHosts: s.cfg.BlockedHosts}
	target, err := validate.URL(rawURL, params)
	if err != nil {
		ve.Add("target_url", describeURLErr(err))
	}
	if alias != "" {
		if verr := validate.Alias(alias); verr != nil {
			ve.Add("alias", describeAliasErr(verr))
		}
	}
	if expiresAt != nil {
		now := time.Now().UTC()
		if expiresAt.Before(now.Add(time.Minute)) || expiresAt.After(now.Add(5*365*24*time.Hour)) {
			ve.Add("expires_at", "expiration must be between 1 minute and 5 years from now")
		}
	}
	if ve.Has() {
		return nil, ve
	}

	if userID != "" {
		n, err := s.urls.CountByUser(ctx, userID)
		if err != nil {
			return nil, ErrInternal
		}
		if n >= int64(s.cfg.MaxURLsUser) {
			return nil, ErrURLLimitReached
		}
	}

	m := &model.ShortURL{
		UserID:    strPtrOrEmpty(userID),
		Code:      alias,
		TargetURL: target,
		ExpiresAt: expiresAt,
	}
	if alias == "" {
		for i := 0; i < maxGenTries; i++ {
			code, err := generateCode()
			if err != nil {
				return nil, ErrInternal
			}
			if validate.IsReserved(code) {
				continue
			}
			m.Code = code
			err = s.urls.Insert(ctx, m)
			if errors.Is(err, repository.ErrConflict) {
				continue // 23505 on generated code: retry
			}
			if err != nil {
				return nil, ErrInternal
			}
			s.cache.Invalidate(m.Code)
			return m, nil
		}
		return nil, ErrInternal
	}

	if err := s.urls.Insert(ctx, m); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrAliasTaken
		}
		s.log.Error("insert failed", "err", err, "code", m.Code)
		return nil, ErrInternal
	}
	s.cache.Invalidate(m.Code)
	return m, nil
}

// Resolve performs the redirect lookup: cache first, then one DB query.
// Returns (targetURL, status) where status nil means OK to redirect.
func (s *URLService) Resolve(ctx context.Context, code string) (string, error) {
	if entry, ok := s.cache.Get(code); ok {
		if entry == nil {
			return "", ErrNotFound // negative cache
		}
		return s.judge(entry, code)
	}

	guard := s.cache.singleflight(code)
	if guard == nil { // someone else loaded it while we waited
		if entry, ok := s.cache.Get(code); ok {
			if entry == nil {
				return "", ErrNotFound
			}
			return s.judge(entry, code)
		}
		return "", ErrInternal
	}
	defer guard()

	row, err := s.urls.ForRedirect(ctx, code)
	if errors.Is(err, repository.ErrNotFound) {
		s.cache.PutNegative(code)
		return "", ErrNotFound
	}
	if err != nil {
		return "", ErrInternal
	}
	entry := &cacheEntry{
		ID: row.ID, TargetURL: row.TargetURL, IsActive: row.IsActive, ExpiresAt: row.ExpiresAt,
	}
	s.cache.Put(code, entry)
	return s.judge(entry, code)
}

func (s *URLService) judge(e *cacheEntry, code string) (string, error) {
	now := time.Now().UTC()
	if !e.IsActive {
		return "", ErrNotFound
	}
	if e.ExpiresAt != nil && !e.ExpiresAt.After(now) {
		return "", ErrGone
	}
	s.clicks.Add(code, now)
	return e.TargetURL, nil
}

// ResolveNoClick is HEAD /{code}: same lookup, no click recorded.
func (s *URLService) ResolveNoClick(ctx context.Context, code string) (string, error) {
	if entry, ok := s.cache.Get(code); ok {
		if entry == nil {
			return "", ErrNotFound
		}
		return judgeNoClick(entry)
	}
	guard := s.cache.singleflight(code)
	if guard == nil {
		if entry, ok := s.cache.Get(code); ok {
			if entry == nil {
				return "", ErrNotFound
			}
			return judgeNoClick(entry)
		}
		return "", ErrInternal
	}
	defer guard()
	row, err := s.urls.ForRedirect(ctx, code)
	if errors.Is(err, repository.ErrNotFound) {
		s.cache.PutNegative(code)
		return "", ErrNotFound
	}
	if err != nil {
		return "", ErrInternal
	}
	entry := &cacheEntry{ID: row.ID, TargetURL: row.TargetURL, IsActive: row.IsActive, ExpiresAt: row.ExpiresAt}
	s.cache.Put(code, entry)
	return judgeNoClick(entry)
}

func judgeNoClick(e *cacheEntry) (string, error) {
	now := time.Now().UTC()
	if !e.IsActive {
		return "", ErrNotFound
	}
	if e.ExpiresAt != nil && !e.ExpiresAt.After(now) {
		return "", ErrGone
	}
	return e.TargetURL, nil
}

// Invalidate removes a code from the redirect cache (create/patch/delete).
func (s *URLService) Invalidate(code string) { s.cache.Invalidate(code) }

func describeURLErr(err error) string {
	switch {
	case errors.Is(err, validate.ErrURLEmpty):
		return "URL is required."
	case errors.Is(err, validate.ErrURLTooLong):
		return fmt.Sprintf("URL must be at most %d characters.", validate.MaxURLLen)
	case errors.Is(err, validate.ErrURLBadChars):
		return "URL must not contain spaces or control characters."
	case errors.Is(err, validate.ErrURLScheme):
		return "URL must start with http:// or https://."
	case errors.Is(err, validate.ErrURLUserinfo):
		return "URL must not contain a username or password."
	case errors.Is(err, validate.ErrURLBlocked):
		return "This host is not allowed."
	case errors.Is(err, validate.ErrURLBadPort):
		return "URL port is out of range."
	default:
		return "Enter a valid URL."
	}
}

func describeAliasErr(err error) string {
	if errors.Is(err, validate.ErrAliasInvalid) {
		return "Use 3-32 characters: letters, numbers, underscore or hyphen; must start with a letter or number."
	}
	return "Alias is not allowed."
}

// generateCode produces a random base62 code without modulo bias.
func generateCode() (string, error) {
	out := make([]byte, codeLen)
	max := big.NewInt(int64(len(codeAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = codeAlphabet[n.Int64()]
	}
	return string(out), nil
}

func strPtrOrEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ---- redirect cache ----

type cacheEntry struct {
	ID        string
	TargetURL string
	IsActive  bool
	ExpiresAt *time.Time
}

type redirectCache struct {
	mu     sync.Mutex
	posLRU *expirable.LRU[string, *cacheEntry]
	neg    map[string]time.Time
	negTTL time.Duration
	sf     *singleflight
}

func newRedirectCache(size int, ttl, negTTL time.Duration) *redirectCache {
	return &redirectCache{
		posLRU: expirable.NewLRU[string, *cacheEntry](size, nil, ttl),
		neg:    map[string]time.Time{},
		negTTL: negTTL,
		sf:     newSingleflight(),
	}
}

func (c *redirectCache) Get(code string) (*cacheEntry, bool) {
	if e, ok := c.posLRU.Get(code); ok {
		return e, true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.neg[code]; ok {
		if time.Now().Before(t) {
			return nil, true
		}
		delete(c.neg, code)
	}
	return nil, false
}

func (c *redirectCache) Put(code string, e *cacheEntry) { c.posLRU.Add(code, e) }

func (c *redirectCache) PutNegative(code string) {
	c.mu.Lock()
	c.neg[code] = time.Now().Add(c.negTTL)
	c.mu.Unlock()
}

func (c *redirectCache) Invalidate(code string) {
	c.posLRU.Remove(code)
	c.mu.Lock()
	delete(c.neg, code)
	c.mu.Unlock()
}

// singleflight ensures concurrent cache misses on the same code run one DB
// query. Do returns a release func for the elected loader, or nil for callers
// that should instead wait for the result to land in the cache.
func (c *redirectCache) singleflight(code string) func() { return c.sf.Do(code) }

// ---- singleflight ----

type call struct {
	done chan struct{}
}

type singleflight struct {
	mu    sync.Mutex
	infly map[string]*call
}

func newSingleflight() *singleflight {
	return &singleflight{infly: map[string]*call{}}
}

// Do returns a release func if the caller is elected to load key, or nil if
// another load is already in flight (the caller then waits for it to finish
// and re-reads the cache).
func (sf *singleflight) Do(key string) func() {
	sf.mu.Lock()
	if c, ok := sf.infly[key]; ok {
		sf.mu.Unlock()
		<-c.done
		return nil
	}
	c := &call{done: make(chan struct{})}
	sf.infly[key] = c
	sf.mu.Unlock()

	return func() {
		sf.mu.Lock()
		if sf.infly[key] == c {
			delete(sf.infly, key)
		}
		sf.mu.Unlock()
		close(c.done)
	}
}
