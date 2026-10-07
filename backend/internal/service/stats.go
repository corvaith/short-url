package service

import (
	"context"
	"errors"

	"shorturl/internal/model"
	"shorturl/internal/repository"
	"time"
)

// StatsService exposes dashboard statistics.
type StatsService struct {
	urls *repository.URLs
}

// NewStatsService wires a StatsService.
func NewStatsService(urls *repository.URLs) *StatsService { return &StatsService{urls: urls} }

// Overview returns aggregate counts for a user.
func (s *StatsService) Overview(ctx context.Context, userID string) (*model.Overview, error) {
	o, err := s.urls.Overview(ctx, userID)
	if err != nil {
		return nil, ErrInternal
	}
	return o, nil
}

// GetStats returns the daily click series for a URL owned by userID.
// days must be 7 or 30.
func (s *StatsService) GetStats(ctx context.Context, userID, urlID string, days int, now time.Time) (*model.StatsResponse, error) {
	if days != 7 && days != 30 {
		days = 30
	}
	resp, err := s.urls.Stats(ctx, urlID, userID, days, now)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrInternal
	}
	return resp, nil
}

// List returns one page of the user's URLs.
func (s *URLService) List(ctx context.Context, userID string, limit int, cursor *repository.Cursor) (*model.URLPage, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	page, err := s.urls.ListPage(ctx, userID, limit, cursor)
	if err != nil {
		return nil, ErrInternal
	}
	return page, nil
}

// Recent returns the user's newest n URLs (home widget, D-19).
func (s *URLService) Recent(ctx context.Context, userID string, n int) ([]*model.ShortURL, error) {
	out, err := s.urls.Recent(ctx, userID, n)
	if err != nil {
		return nil, ErrInternal
	}
	return out, nil
}

// Get returns one URL owned by userID or ErrNotFound.
func (s *URLService) Get(ctx context.Context, userID, id string) (*model.ShortURL, error) {
	m, err := s.urls.ByID(ctx, id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrInternal
	}
	return m, nil
}

// SetActive enables/disables a URL owned by userID.
func (s *URLService) SetActive(ctx context.Context, userID, id string, active bool) (*model.ShortURL, error) {
	m, err := s.urls.SetActive(ctx, id, userID, active)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrInternal
	}
	s.Invalidate(m.Code)
	return m, nil
}

// Delete removes a URL owned by userID.
func (s *URLService) Delete(ctx context.Context, userID, id string) error {
	m, err := s.urls.ByID(ctx, id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return ErrInternal
	}
	if err := s.urls.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}
	s.Invalidate(m.Code)
	return nil
}
