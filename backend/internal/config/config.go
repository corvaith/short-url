// Package config loads and validates environment configuration at startup.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL string
	Port        string
	AppEnv      string // development | test | production
	BaseURL     string // origin without trailing slash, e.g. https://domain.com
	StaticDir   string

	AllowedOrigins  []string
	CookieSecure    bool
	TrustedProxies  []string
	SessionTTL      time.Duration
	LogLevel        string
	DBMaxConns      int32
	ClickFlushEvery time.Duration

	CacheSize    int
	CacheTTL     time.Duration
	CacheNegTTL  time.Duration
	MaxURLsUser  int
	BlockedHosts map[string]struct{}

	// Rate limits (requests per minute, burst)
	RateRedirectIP Rate // per IP
	RateCreateAnon Rate // per IP
	RateCreateUser Rate // per user
	RateLoginIP    Rate // per IP
	RateLoginFail  Rate // per (IP+email) per 15 min
	RateRegisterIP Rate // per IP, per hour
	RateAPIGeneral Rate // other endpoints, per IP/user
}

// Rate bundles a requests-per-minute limit with a burst allowance.
type Rate struct {
	PerMinute int
	Burst     int
}

func envStr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid integer %q", key, v)
	}
	return n, nil
}

func envBool(key string, def bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: invalid boolean %q", key, v)
	}
	return b, nil
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s: invalid duration %q", key, v)
	}
	return d, nil
}

func splitList(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func rateEnv(prefix string, def Rate) (Rate, error) {
	pm, err := envInt(prefix+"_PER_MINUTE", def.PerMinute)
	if err != nil {
		return Rate{}, err
	}
	b, err := envInt(prefix+"_BURST", def.Burst)
	if err != nil {
		return Rate{}, err
	}
	return Rate{PerMinute: pm, Burst: b}, nil
}

// Load reads environment variables and validates the result. It fails fast
// on missing or invalid required settings.
func Load() (*Config, error) {
	c := &Config{
		DatabaseURL: envStr("DATABASE_URL", ""),
		Port:        envStr("PORT", "8080"),
		AppEnv:      envStr("APP_ENV", "development"),
		BaseURL:     strings.TrimRight(envStr("BASE_URL", ""), "/"),
		StaticDir:   envStr("STATIC_DIR", ""),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.BaseURL == "" {
		return nil, fmt.Errorf("BASE_URL is required")
	}
	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return nil, fmt.Errorf("BASE_URL must start with http:// or https://")
	}
	switch c.AppEnv {
	case "development", "test", "production":
	default:
		return nil, fmt.Errorf("APP_ENV must be development, test or production")
	}
	if c.AppEnv == "production" {
		if !strings.HasPrefix(c.BaseURL, "https://") {
			return nil, fmt.Errorf("BASE_URL must use https:// in production")
		}
	}

	var err error
	c.AllowedOrigins = splitList(envStr("ALLOWED_ORIGINS", ""))
	if c.CookieSecure, err = envBool("COOKIE_SECURE", c.AppEnv == "production"); err != nil {
		return nil, err
	}
	c.TrustedProxies = splitList(envStr("TRUSTED_PROXIES", ""))
	ttlH, err := envInt("SESSION_TTL_HOURS", 720)
	if err != nil {
		return nil, err
	}
	c.SessionTTL = time.Duration(ttlH) * time.Hour
	c.LogLevel = envStr("LOG_LEVEL", "info")
	dbConns, err := envInt("DB_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}
	c.DBMaxConns = int32(dbConns)
	if c.ClickFlushEvery, err = envDuration("CLICK_FLUSH_INTERVAL", 5*time.Second); err != nil {
		return nil, err
	}
	if c.CacheSize, err = envInt("CACHE_SIZE", 10000); err != nil {
		return nil, err
	}
	if c.CacheTTL, err = envDuration("CACHE_TTL", 60*time.Second); err != nil {
		return nil, err
	}
	if c.CacheNegTTL, err = envDuration("CACHE_NEG_TTL", 10*time.Second); err != nil {
		return nil, err
	}
	if c.MaxURLsUser, err = envInt("MAX_URLS_PER_USER", 1000); err != nil {
		return nil, err
	}

	c.BlockedHosts = map[string]struct{}{}
	for _, h := range splitList(envStr("BLOCKED_HOSTS", "")) {
		c.BlockedHosts[strings.ToLower(h)] = struct{}{}
	}

	if c.RateRedirectIP, err = rateEnv("RATE_REDIRECT", Rate{PerMinute: 240, Burst: 60}); err != nil {
		return nil, err
	}
	if c.RateCreateAnon, err = rateEnv("RATE_CREATE_ANON", Rate{PerMinute: 10, Burst: 5}); err != nil {
		return nil, err
	}
	if c.RateCreateUser, err = rateEnv("RATE_CREATE_USER", Rate{PerMinute: 30, Burst: 10}); err != nil {
		return nil, err
	}
	if c.RateLoginIP, err = rateEnv("RATE_LOGIN", Rate{PerMinute: 10, Burst: 10}); err != nil {
		return nil, err
	}
	if c.RateLoginFail, err = rateEnv("RATE_LOGIN_FAIL", Rate{PerMinute: 20, Burst: 5}); err != nil {
		return nil, err
	}
	if c.RateRegisterIP, err = rateEnv("RATE_REGISTER", Rate{PerMinute: 10, Burst: 10}); err != nil {
		return nil, err
	}
	if c.RateAPIGeneral, err = rateEnv("RATE_API", Rate{PerMinute: 120, Burst: 120}); err != nil {
		return nil, err
	}

	return c, nil
}

// IsProduction reports whether APP_ENV is production.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// OwnHost returns the host (with port, if non-default) of BaseURL.
func (c *Config) OwnHost() string {
	i := strings.Index(c.BaseURL, "://")
	if i < 0 {
		return c.BaseURL
	}
	return c.BaseURL[i+3:]
}
