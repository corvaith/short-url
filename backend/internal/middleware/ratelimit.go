package middleware

import (
	"sync"
	"time"

	"golang.org/x/time/rate"

	"shorturl/internal/config"
)

// bucket is a token bucket plus last-seen time for cleanup.
type bucket struct {
	lim  *rate.Limiter
	seen time.Time
}

// Limiter is an in-memory per-key token bucket limiter.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	fail    map[string]*failWindow
}

// NewLimiter builds a Limiter.
func NewLimiter() *Limiter {
	l := &Limiter{buckets: map[string]*bucket{}, fail: map[string]*failWindow{}}
	go l.cleanupLoop()
	return l
}

// Allow reports whether key may proceed at (perMinute, burst). Returns the
// wait time when denied (for Retry-After).
func (l *Limiter) Allow(key string, perMinute, burst int) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{lim: rate.NewLimiter(rate.Limit(float64(perMinute)/60.0), burst)}
		l.buckets[key] = b
	}
	b.seen = time.Now()
	res := b.lim.Reserve()
	if !res.OK() {
		return false, time.Minute
	}
	delay := res.Delay()
	if delay <= 0 {
		res.Cancel()
		return true, 0
	}
	res.Cancel()
	return false, delay
}

// failWindow counts failures per (ip+email) over a rolling window.
type failWindow struct {
	at []time.Time
}

// Fail records a login failure for key.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, ok := l.fail[key]
	if !ok {
		f = &failWindow{}
		l.fail[key] = f
	}
	f.at = append(f.at, time.Now())
}

// Failures counts failures for key in the last window duration.
func (l *Limiter) Failures(key string, window time.Duration) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, ok := l.fail[key]
	if !ok {
		return 0
	}
	cut := time.Now().Add(-window)
	kept := f.at[:0]
	n := 0
	for _, t := range f.at {
		if t.After(cut) {
			kept = append(kept, t)
			n++
		}
	}
	f.at = kept
	return n
}

// ClearFailures resets failure counts for key (successful login).
func (l *Limiter) ClearFailures(key string) {
	l.mu.Lock()
	delete(l.fail, key)
	l.mu.Unlock()
}

func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		cut := time.Now().Add(-15 * time.Minute)
		l.mu.Lock()
		for k, b := range l.buckets {
			if b.seen.Before(cut) {
				delete(l.buckets, k)
			}
		}
		for k, f := range l.fail {
			if len(f.at) == 0 || f.at[len(f.at)-1].Before(cut) {
				delete(l.fail, k)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit gates requests by keyFn using cfg limits; on denial it writes a
// 429 with Retry-After and returns false.
func RateLimit(l *Limiter, key string, cfg config.Rate) (bool, time.Duration) {
	ok, wait := l.Allow(key, cfg.PerMinute, cfg.Burst)
	return ok, wait
}
