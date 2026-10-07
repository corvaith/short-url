package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"shorturl/internal/repository"
)

// ClickBuffer accumulates click increments in memory and flushes them to the
// database in batches. Adds are non-blocking; redirects never wait on DB
// writes (PRD §8.4).
type ClickBuffer struct {
	mu       sync.Mutex
	byCode   map[string]*clickAccum
	urls     *repository.URLs
	interval time.Duration
	log      *slog.Logger
}

type clickAccum struct {
	clicks     int64
	lastAccess time.Time
}

// NewClickBuffer builds a ClickBuffer.
func NewClickBuffer(urls *repository.URLs, interval time.Duration, log *slog.Logger) *ClickBuffer {
	return &ClickBuffer{
		byCode:   map[string]*clickAccum{},
		urls:     urls,
		interval: interval,
		log:      log,
	}
}

// Add records one click for code (non-blocking, in-memory).
func (b *ClickBuffer) Add(code string, at time.Time) {
	b.mu.Lock()
	a, ok := b.byCode[code]
	if !ok {
		a = &clickAccum{}
		b.byCode[code] = a
	}
	a.clicks++
	if at.After(a.lastAccess) {
		a.lastAccess = at
	}
	b.mu.Unlock()
}

// Start runs the periodic flush loop until ctx is cancelled.
func (b *ClickBuffer) Start(ctx context.Context) {
	ticker := time.NewTicker(b.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := b.Flush(context.Background()); err != nil {
					b.log.Error("click flush failed", "err", err)
				}
			}
		}
	}()
}

// Flush drains the buffer and applies all increments in one transaction.
// Additive updates make this safe across instances.
func (b *ClickBuffer) Flush(ctx context.Context) error {
	b.mu.Lock()
	if len(b.byCode) == 0 {
		b.mu.Unlock()
		return nil
	}
	drained := b.byCode
	b.byCode = map[string]*clickAccum{}
	b.mu.Unlock()

	// Merge per code (a code could have accumulated while flushing before).
	entries := make([]repository.ClickDelta, 0, len(drained))
	for code, a := range drained {
		day := a.lastAccess.UTC().Format("2006-01-02")
		entries = append(entries, repository.ClickDelta{
			Code: code, Clicks: a.clicks, Day: day, LastAccess: a.lastAccess.UTC(),
		})
	}
	// Deterministic order for tests/logs.
	sort.Slice(entries, func(i, j int) bool { return strings.Compare(entries[i].Code, entries[j].Code) < 0 })

	if err := b.urls.FlushClicks(ctx, entries); err != nil {
		// Put the clicks back so they are not lost; next flush retries.
		b.mu.Lock()
		for _, e := range entries {
			a := b.byCode[e.Code]
			if a == nil {
				a = &clickAccum{}
				b.byCode[e.Code] = a
			}
			a.clicks += e.Clicks
			if e.LastAccess.After(a.lastAccess) {
				a.lastAccess = e.LastAccess
			}
		}
		b.mu.Unlock()
		return err
	}
	return nil
}
