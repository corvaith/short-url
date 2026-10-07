package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/crypto/argon2"

	"shorturl/internal/config"
	"shorturl/internal/model"
	"shorturl/internal/repository"
	"shorturl/internal/validate"
)

// AuthService handles registration, login, sessions and password changes.
type AuthService struct {
	cfg      *config.Config
	users    *repository.Users
	sessions *repository.Sessions
	log      *slog.Logger
}

// NewAuthService wires an AuthService.
func NewAuthService(cfg *config.Config, users *repository.Users, sessions *repository.Sessions, log *slog.Logger) *AuthService {
	return &AuthService{cfg: cfg, users: users, sessions: sessions, log: log}
}

// Register creates a user and a session (auto-login). Returns the raw session
// token to set as a cookie.
func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, string, error) {
	norm, err := validate.Email(email)
	if err != nil {
		ve := &ValidationError{}
		ve.Add("email", "Enter a valid email address.")
		return nil, "", ve
	}
	if verr := validate.Password(password, norm); verr != nil {
		ve := &ValidationError{}
		ve.Add("password", passwordDetail(verr))
		return nil, "", ve
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, "", ErrInternal
	}
	u, err := s.users.Create(ctx, norm, hash)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, "", ErrEmailTaken
		}
		return nil, "", ErrInternal
	}
	token, err := s.newSession(ctx, u.ID)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// Login verifies credentials and creates a session. Email-not-found and
// wrong-password both return ErrInvalidCredentials.
func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	norm, _ := validate.Email(email)
	u, err := s.users.ByEmail(ctx, norm)
	if errors.Is(err, repository.ErrNotFound) {
		// Constant-time-ish: hash against a dummy so timing is comparable.
		_ = verifyPassword(password, dummyHash)
		return nil, "", ErrInvalidCredentials
	}
	if err != nil {
		return nil, "", ErrInternal
	}
	if !verifyPassword(password, u.PasswordHash) {
		return nil, "", ErrInvalidCredentials
	}
	token, err := s.newSession(ctx, u.ID)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// Logout deletes the session identified by the raw token.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, hashToken(token))
}

// Authenticate resolves a raw token to a user and the session. Refreshes
// last_used_at at most once per hour.
func (s *AuthService) Authenticate(ctx context.Context, token string) (*model.User, *model.Session, error) {
	if token == "" {
		return nil, nil, ErrInvalidCredentials
	}
	sess, err := s.sessions.ByTokenHash(ctx, hashToken(token))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, ErrInternal
	}
	u, err := s.users.ByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}
	if time.Since(sess.LastUsedAt) > time.Hour {
		_ = s.sessions.TouchLastUsed(ctx, sess.ID)
	}
	return u, sess, nil
}

// ChangePassword verifies the current password, sets a new hash and revokes
// every other session of the user.
func (s *AuthService) ChangePassword(ctx context.Context, userID, keepSessionID, current, next string) error {
	u, err := s.users.ByID(ctx, userID)
	if err != nil {
		return ErrInternal
	}
	if !verifyPassword(current, u.PasswordHash) {
		ve := &ValidationError{}
		ve.Add("current_password", "Current password is incorrect.")
		return ve
	}
	if verr := validate.Password(next, u.Email); verr != nil {
		ve := &ValidationError{}
		ve.Add("new_password", passwordDetail(verr))
		return ve
	}
	hash, err := hashPassword(next)
	if err != nil {
		return ErrInternal
	}
	if err := s.users.UpdatePassword(ctx, userID, hash); err != nil {
		s.log.Error("pw update failed", "err", err)
		return ErrInternal
	}
	if err := s.sessions.DeleteOthers(ctx, userID, keepSessionID); err != nil {
		s.log.Error("session revoke failed", "err", err)
		return ErrInternal
	}
	return nil
}

// CleanupExpiredSessions runs hourly in the background.
func (s *AuthService) CleanupExpiredSessions(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := s.sessions.DeleteExpired(context.Background()); err != nil {
					s.log.Error("session cleanup failed", "err", err)
				} else if n > 0 {
					s.log.Info("expired sessions removed", "count", n)
				}
			}
		}
	}()
}

func (s *AuthService) newSession(ctx context.Context, userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", ErrInternal
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if _, err := s.sessions.Create(ctx, userID, hashToken(token), s.cfg.SessionTTL); err != nil {
		return "", ErrInternal
	}
	return token, nil
}

// hashToken returns SHA-256 of a raw session token.
func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// ---- argon2id password hashing (PHC string format) ----

const (
	argonMemory  = 64 * 1024 // 64 MiB
	argonTime    = 3
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return encodePHC(key, salt), nil
}

func encodePHC(key, salt []byte) string {
	b64 := base64.RawStdEncoding.EncodeToString
	return "$argon2id$v=19$m=65536,t=3,p=2$" + b64(salt) + "$" + b64(key)
}

func verifyPassword(password, phc string) bool {
	params, salt, want, ok := decodePHC(phc)
	if !ok {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, params.t, params.m, params.p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

type argonParams struct {
	m uint32
	t uint32
	p uint8
}

func decodePHC(phc string) (argonParams, []byte, []byte, bool) {
	// Format: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
	var p argonParams
	if len(phc) == 0 || phc[0] != '$' {
		return p, nil, nil, false
	}
	parts := splitDollar(phc)
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, nil, nil, false
	}
	var version int
	if _, err := sscan(parts[2], "v=", &version); err != nil || version != 19 {
		return p, nil, nil, false
	}
	var m, t int
	var pp int
	if _, err := sscan(parts[3], "m=", &m); err != nil {
		return p, nil, nil, false
	}
	if !scanCommaInts(parts[3], &t, &pp) {
		return p, nil, nil, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return p, nil, nil, false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return p, nil, nil, false
	}
	p = argonParams{m: uint32(m), t: uint32(t), p: uint8(pp)}
	return p, salt, want, true
}

func splitDollar(s string) []string {
	out := []string{}
	start := 1
	for i := 1; i < len(s); i++ {
		if s[i] == '$' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return append([]string{""}, out...)
}

func sscan(s, prefix string, dst *int) (int, error) {
	if len(s) < len(prefix) || s[:len(prefix)] != prefix {
		return 0, errors.New("bad prefix")
	}
	n := 0
	for _, r := range s[len(prefix):] {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	*dst = n
	return n, nil
}

// scanCommaInts extracts the "t=" and "p=" values from the params segment.
func scanCommaInts(seg string, t, p *int) bool {
	var m int
	_, err := sscan(seg, "m=", &m)
	if err != nil {
		return false
	}
	// find ",t=" and ",p="
	ti, pi := indexOf(seg, ",t="), indexOf(seg, ",p=")
	if ti < 0 || pi < 0 {
		return false
	}
	if _, err := sscan(seg[ti+1:], "t=", t); err != nil {
		return false
	}
	if _, err := sscan(seg[pi+1:], "p=", p); err != nil {
		return false
	}
	return true
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func passwordDetail(err error) string {
	if errors.Is(err, validate.ErrPasswordLen) {
		return "Password must be 8-128 characters."
	}
	if errors.Is(err, validate.ErrPasswordSame) {
		return "Password must be different from your email."
	}
	return "Password is not valid."
}
