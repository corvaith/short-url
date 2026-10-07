// Package model defines domain entities and API DTOs.
package model

import "time"

// User is a registered account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserDTO is the public representation of a user.
type UserDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ToDTO converts a User to its public representation (timestamps in UTC).
func (u *User) ToDTO() UserDTO {
	return UserDTO{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt.UTC()}
}

// Status is the derived status of a short URL.
type Status string

const (
	StatusActive   Status = "active"
	StatusExpired  Status = "expired"
	StatusDisabled Status = "disabled"
)

// ShortURL is a shortened link.
type ShortURL struct {
	ID             string
	UserID         *string
	Code           string
	TargetURL      string
	Clicks         int64
	LastAccessedAt *time.Time
	ExpiresAt      *time.Time
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Status derives the status from is_active and expires_at.
func (s *ShortURL) Status(now time.Time) Status {
	if !s.IsActive {
		return StatusDisabled
	}
	if s.ExpiresAt != nil && !s.ExpiresAt.After(now) {
		return StatusExpired
	}
	return StatusActive
}

// URLDTO is the API representation of a short URL. short_url is filled in by
// the handler layer (needs BASE_URL).
type URLDTO struct {
	ID             string     `json:"id"`
	Code           string     `json:"code"`
	ShortURL       string     `json:"short_url"`
	TargetURL      string     `json:"target_url"`
	Clicks         int64      `json:"clicks"`
	Status         Status     `json:"status"`
	IsActive       bool       `json:"is_active"`
	ExpiresAt      *time.Time `json:"expires_at"`
	LastAccessedAt *time.Time `json:"last_accessed_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ToDTO converts a ShortURL to its API representation. baseURL has no trailing
// slash.
func (s *ShortURL) ToDTO(baseURL string, now time.Time) URLDTO {
	return URLDTO{
		ID:             s.ID,
		Code:           s.Code,
		ShortURL:       baseURL + "/" + s.Code,
		TargetURL:      s.TargetURL,
		Clicks:         s.Clicks,
		Status:         s.Status(now),
		IsActive:       s.IsActive,
		ExpiresAt:      utcPtr(s.ExpiresAt),
		LastAccessedAt: utcPtr(s.LastAccessedAt),
		CreatedAt:      s.CreatedAt.UTC(),
	}
}

// utcPtr converts an optional timestamp to UTC.
func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

// Session is a login session row. The raw token is never stored.
type Session struct {
	ID         string
	UserID     string
	TokenHash  []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastUsedAt time.Time
}

// StatsSeries is one day of click counts.
type StatsSeries struct {
	Date   string `json:"date"` // YYYY-MM-DD (UTC)
	Clicks int64  `json:"clicks"`
}

// StatsResponse is the payload for GET /urls/{id}/stats.
type StatsResponse struct {
	ID          string        `json:"id"`
	Days        int           `json:"days"`
	RangeClicks int64         `json:"range_clicks"`
	TotalClicks int64         `json:"total_clicks"`
	Series      []StatsSeries `json:"series"`
}

// Overview is the payload for GET /stats/overview.
type Overview struct {
	TotalURLs   int64 `json:"total_urls"`
	TotalClicks int64 `json:"total_clicks"`
	ActiveURLs  int64 `json:"active_urls"`
}

// URLPage is a cursor-paginated page of short URLs.
type URLPage struct {
	Items      []*ShortURL
	NextCursor string // empty when exhausted
}
