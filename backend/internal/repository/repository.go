// Package repository holds the Postgres data access layer. All SQL lives
// here; queries are parameterized exclusively.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"shorturl/internal/model"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict wraps unique-violation duplicates (alias/email taken).
var ErrConflict = errors.New("conflict")

// Users implements user persistence.
type Users struct {
	pool *pgxpool.Pool
}

// NewUsers builds a Users repository.
func NewUsers(pool *pgxpool.Pool) *Users { return &Users{pool: pool} }

// Create inserts a user and returns it.
func (u *Users) Create(ctx context.Context, email, passwordHash string) (*model.User, error) {
	row := u.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at`,
		email, passwordHash)
	var m model.User
	if err := row.Scan(&m.ID, &m.Email, &m.PasswordHash, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

// ByEmail finds a user by (lowercased) email.
func (u *Users) ByEmail(ctx context.Context, email string) (*model.User, error) {
	row := u.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1`, email)
	var m model.User
	if err := row.Scan(&m.ID, &m.Email, &m.PasswordHash, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

// ByID finds a user by ID.
func (u *Users) ByID(ctx context.Context, id string) (*model.User, error) {
	row := u.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE id = $1`, id)
	var m model.User
	if err := row.Scan(&m.ID, &m.Email, &m.PasswordHash, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

// UpdatePassword replaces the password hash and bumps updated_at.
func (u *Users) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	ct, err := u.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`,
		id, passwordHash)
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Sessions implements session persistence. Only token hashes are stored.
type Sessions struct {
	pool *pgxpool.Pool
}

// NewSessions builds a Sessions repository.
func NewSessions(pool *pgxpool.Pool) *Sessions { return &Sessions{pool: pool} }

// Create inserts a session.
func (s *Sessions) Create(ctx context.Context, userID string, tokenHash []byte, ttl time.Duration) (*model.Session, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, now() + make_interval(secs => $3))
		RETURNING id, user_id, token_hash, created_at, expires_at, last_used_at`,
		userID, tokenHash, ttl.Seconds())
	var m model.Session
	if err := row.Scan(&m.ID, &m.UserID, &m.TokenHash, &m.CreatedAt, &m.ExpiresAt, &m.LastUsedAt); err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

// ByTokenHash finds a valid (non-expired) session by token hash.
func (s *Sessions) ByTokenHash(ctx context.Context, tokenHash []byte) (*model.Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, created_at, expires_at, last_used_at
		FROM sessions
		WHERE token_hash = $1 AND expires_at > now()`, tokenHash)
	var m model.Session
	if err := row.Scan(&m.ID, &m.UserID, &m.TokenHash, &m.CreatedAt, &m.ExpiresAt, &m.LastUsedAt); err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

// TouchLastUsed refreshes last_used_at (rate-limited by the caller).
func (s *Sessions) TouchLastUsed(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sessions SET last_used_at = now() WHERE id = $1`, id)
	return err
}

// Delete removes a session by token hash (logout).
func (s *Sessions) Delete(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

// DeleteOthers removes all sessions of a user except the given one.
func (s *Sessions) DeleteOthers(ctx context.Context, userID, keepID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM sessions WHERE user_id = $1 AND id <> $2`, userID, keepID)
	return err
}

// DeleteExpired removes expired sessions; returns rows affected.
func (s *Sessions) DeleteExpired(ctx context.Context) (int64, error) {
	ct, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// URLRow is the redirect-relevant projection of a short URL.
type URLRow struct {
	ID        string
	TargetURL string
	IsActive  bool
	ExpiresAt *time.Time
}

// URLs implements short URL persistence.
type URLs struct {
	pool *pgxpool.Pool
}

// NewURLs builds a URLs repository.
func NewURLs(pool *pgxpool.Pool) *URLs { return &URLs{pool: pool} }

// ForRedirect fetches the minimal row needed to serve a redirect.
func (u *URLs) ForRedirect(ctx context.Context, code string) (*URLRow, error) {
	row := u.pool.QueryRow(ctx, `
		SELECT id, target_url, is_active, expires_at
		FROM short_urls WHERE code = $1`, code)
	var r URLRow
	if err := row.Scan(&r.ID, &r.TargetURL, &r.IsActive, &r.ExpiresAt); err != nil {
		return nil, mapErr(err)
	}
	return &r, nil
}

// Insert stores a new short URL.
func (u *URLs) Insert(ctx context.Context, m *model.ShortURL) error {
	row := u.pool.QueryRow(ctx, `
		INSERT INTO short_urls (user_id, code, target_url, expires_at)
		VALUES (NULLIF($1,'')::uuid, $2, $3, $4)
		RETURNING id, clicks, is_active, created_at, updated_at`,
		derefOrNil(m.UserID), m.Code, m.TargetURL, m.ExpiresAt)
	if err := row.Scan(&m.ID, &m.Clicks, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return mapErr(err)
	}
	return nil
}

// ByID fetches a URL the user owns; non-owned or missing → ErrNotFound.
func (u *URLs) ByID(ctx context.Context, id, userID string) (*model.ShortURL, error) {
	row := u.pool.QueryRow(ctx, `
		SELECT id, user_id::text, code, target_url, clicks, last_accessed_at,
		       expires_at, is_active, created_at, updated_at
		FROM short_urls WHERE id = $1 AND user_id = $2::uuid`, id, userID)
	return scanURL(row)
}

// ListPage returns one cursor page for a user, newest first.
// cursor is (created_at, id) of the last item of the previous page.
func (u *URLs) ListPage(ctx context.Context, userID string, limit int, cur *Cursor) (*model.URLPage, error) {
	q := `
		SELECT id, user_id::text, code, target_url, clicks, last_accessed_at,
		       expires_at, is_active, created_at, updated_at
		FROM short_urls
		WHERE user_id = $1::uuid`
	args := []any{userID}
	if cur != nil {
		q += ` AND (created_at, id) < ($2::timestamptz, $3::uuid)`
		args = append(args, cur.CreatedAt, cur.ID)
	}
	q += fmt.Sprintf(` ORDER BY created_at DESC, id DESC LIMIT $%d`, len(args)+1)
	args = append(args, limit+1)

	rows, err := u.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	page := &model.URLPage{Items: []*model.ShortURL{}}
	for rows.Next() {
		m, err := scanURLRow(rows)
		if err != nil {
			return nil, err
		}
		page.Items = append(page.Items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(page.Items) > limit {
		last := page.Items[limit-1]
		page.NextCursor = EncodeCursor(last.CreatedAt, last.ID)
		page.Items = page.Items[:limit]
	}
	return page, nil
}

// CountByUser returns how many URLs a user owns.
func (u *URLs) CountByUser(ctx context.Context, userID string) (int64, error) {
	var n int64
	err := u.pool.QueryRow(ctx,
		`SELECT count(*) FROM short_urls WHERE user_id = $1::uuid`, userID).Scan(&n)
	return n, err
}

// SetActive flips is_active; ownership enforced by caller.
func (u *URLs) SetActive(ctx context.Context, id, userID string, active bool) (*model.ShortURL, error) {
	row := u.pool.QueryRow(ctx, `
		UPDATE short_urls SET is_active = $3, updated_at = now()
		WHERE id = $1 AND user_id = $2::uuid
		RETURNING id, user_id::text, code, target_url, clicks, last_accessed_at,
		          expires_at, is_active, created_at, updated_at`,
		id, userID, active)
	return scanURL(row)
}

// Delete removes a URL owned by userID.
func (u *URLs) Delete(ctx context.Context, id, userID string) error {
	ct, err := u.pool.Exec(ctx,
		`DELETE FROM short_urls WHERE id = $1 AND user_id = $2::uuid`, id, userID)
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FlushClicks applies buffered click counts in one transaction. Buffer maps
// code -> (urlID may be resolved by caller, delta, lastAccess).
func (u *URLs) FlushClicks(ctx context.Context, entries []ClickDelta) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, e := range entries {
		var id string
		var last time.Time
		err := tx.QueryRow(ctx, `
			UPDATE short_urls
			SET clicks = clicks + $2,
			    last_accessed_at = GREATEST(COALESCE(last_accessed_at, to_timestamp(0)), $3::timestamptz)
			WHERE code = $1
			RETURNING id, last_accessed_at`,
			e.Code, e.Clicks, e.LastAccess).Scan(&id, &last)
		if errors.Is(err, pgx.ErrNoRows) {
			continue // URL deleted before flush; clicks are dropped
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO url_daily_clicks (url_id, day, clicks)
			VALUES ($1, $2::date, $3)
			ON CONFLICT (url_id, day)
			DO UPDATE SET clicks = url_daily_clicks.clicks + EXCLUDED.clicks`,
			id, e.Day, e.Clicks); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ClickDelta is one buffered click increment for a code.
type ClickDelta struct {
	Code       string
	Clicks     int64
	Day        string // YYYY-MM-DD UTC
	LastAccess time.Time
}

// Stats returns the daily series for a URL owned by userID.
func (u *URLs) Stats(ctx context.Context, id, userID string, days int, today time.Time) (*model.StatsResponse, error) {
	// Ensure ownership + totals.
	m, err := u.ByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	start := today.AddDate(0, 0, -(days - 1)).UTC()
	rows, err := u.pool.Query(ctx, `
		SELECT day::text, clicks
		FROM url_daily_clicks
		WHERE url_id = $1 AND day >= $2::date AND day <= $3::date
		ORDER BY day ASC`, m.ID, start.Format("2006-01-02"), today.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byDay := map[string]int64{}
	for rows.Next() {
		var day string
		var n int64
		if err := rows.Scan(&day, &n); err != nil {
			return nil, err
		}
		byDay[day] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	resp := &model.StatsResponse{ID: m.ID, Days: days, TotalClicks: m.Clicks, Series: make([]model.StatsSeries, 0, days)}
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		n := byDay[d]
		resp.RangeClicks += n
		resp.Series = append(resp.Series, model.StatsSeries{Date: d, Clicks: n})
	}
	return resp, nil
}

// Overview returns aggregate stats for a user.
func (u *URLs) Overview(ctx context.Context, userID string) (*model.Overview, error) {
	o := &model.Overview{}
	err := u.pool.QueryRow(ctx, `
		SELECT count(*),
		       COALESCE(sum(clicks), 0),
		       count(*) FILTER (WHERE is_active AND (expires_at IS NULL OR expires_at > now()))
		FROM short_urls WHERE user_id = $1::uuid`, userID).
		Scan(&o.TotalURLs, &o.TotalClicks, &o.ActiveURLs)
	return o, err
}

// Recent returns the newest N URLs of a user.
func (u *URLs) Recent(ctx context.Context, userID string, n int) ([]*model.ShortURL, error) {
	rows, err := u.pool.Query(ctx, `
		SELECT id, user_id::text, code, target_url, clicks, last_accessed_at,
		       expires_at, is_active, created_at, updated_at
		FROM short_urls WHERE user_id = $1::uuid
		ORDER BY created_at DESC, id DESC LIMIT $2`, userID, n)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := []*model.ShortURL{}
	for rows.Next() {
		m, err := scanURLRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(dest ...any) error }

func scanURL(row rowScanner) (*model.ShortURL, error) {
	var m model.ShortURL
	err := row.Scan(&m.ID, &m.UserID, &m.Code, &m.TargetURL, &m.Clicks,
		&m.LastAccessedAt, &m.ExpiresAt, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

func scanURLRow(rows pgx.Rows) (*model.ShortURL, error) {
	return scanURL(rows)
}

func derefOrNil(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return translate(err)
}

// translate converts a Postgres error into a repository error, mapping unique
// violations (SQLSTATE 23505) to ErrConflict.
func translate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}
