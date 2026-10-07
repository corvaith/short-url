package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"crypto/rand"

	"shorturl/internal/config"
	"shorturl/internal/model"
	"shorturl/internal/repository"
)

// OAuth provider configuration. Client secrets come from env:
// GITHUB_CLIENT_ID/GITHUB_CLIENT_SECRET, DISCORD_CLIENT_ID/DISCORD_CLIENT_SECRET.
// An empty client ID disables that provider.
type OAuthService struct {
	cfg    *config.Config
	users  *repository.Users
	auth   *AuthService
	client *http.Client
	log    *slog.Logger
}

func NewOAuthService(cfg *config.Config, users *repository.Users, auth *AuthService, log *slog.Logger) *OAuthService {
	return &OAuthService{
		cfg:    cfg,
		users:  users,
		auth:   auth,
		client: &http.Client{Timeout: 10 * time.Second},
		log:    log,
	}
}

// state holds the anti-CSRF value for one OAuth round trip, kept in memory
// (single-instance deployment per PRD §20) with a short TTL.
type oauthState struct {
	provider string
	next     string
	exp      time.Time
}

var oauthStates = map[string]oauthState{}

// NewState issues a one-time state value for the provider redirect.
func (s *OAuthService) NewState(provider, next string) string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	val := encodeBase64URL(b)
	oauthStates[val] = oauthState{provider: provider, next: next, exp: time.Now().Add(10 * time.Minute)}
	// opportunistic cleanup
	for k, st := range oauthStates {
		if time.Now().After(st.exp) {
			delete(oauthStates, k)
		}
	}
	return val
}

// ConsumeState validates and removes a state value. Returns the stored next URL.
func (s *OAuthService) ConsumeState(provider, state string) (string, bool) {
	st, ok := oauthStates[state]
	if !ok || st.provider != provider || time.Now().After(st.exp) {
		return "", false
	}
	delete(oauthStates, state)
	if !strings.HasPrefix(st.next, "/") || strings.HasPrefix(st.next, "//") {
		return "", true
	}
	return st.next, true
}

// AuthCodeURL builds the provider consent redirect URL.
func (s *OAuthService) AuthCodeURL(provider, state string) (string, error) {
	redirect := s.redirectURI(provider)
	switch provider {
	case "github":
		return "https://github.com/login/oauth/authorize?" + url.Values{
			"client_id":    {s.cfg.GitHubClientID},
			"redirect_uri": {redirect},
			"state":        {state},
			"scope":        {"read:user user:email"},
		}.Encode(), nil
	case "discord":
		return "https://discord.com/oauth2/authorize?" + url.Values{
			"client_id":     {s.cfg.DiscordClientID},
			"redirect_uri":  {redirect},
			"state":         {state},
			"response_type": {"code"},
			"scope":         {"identify email"},
		}.Encode(), nil
	}
	return "", ErrOAuthProvider
}

type oauthToken struct {
	AccessToken string `json:"access_token"`
}

type oauthUser struct {
	ID    string
	Email string
}

// ExchangeIdentity turns an authorization code into a provider identity.
func (s *OAuthService) ExchangeIdentity(ctx context.Context, provider, code string) (*oauthUser, error) {
	tokenURL, userURL, headers := s.endpoints(provider)
	if tokenURL == "" {
		return nil, ErrOAuthProvider
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.redirectURI(provider)},
		"client_id":     {s.clientID(provider)},
		"client_secret": {s.clientSecret(provider)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: status %d", res.StatusCode)
	}
	var tok oauthToken
	if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
		return nil, errors.New("token exchange: no access token")
	}

	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req2.Header.Set(k, v)
	}
	req2.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req2.Header.Set("Accept", "application/json")
	res2, err := s.client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("user request: %w", err)
	}
	defer res2.Body.Close()
	body2, _ := io.ReadAll(io.LimitReader(res2.Body, 1<<20))
	if res2.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user fetch failed: status %d", res2.StatusCode)
	}

	switch provider {
	case "github":
		var gu struct {
			ID    int64  `json:"id"`
			Login string `json:"login"`
			Email string `json:"email"`
		}
		if err := json.Unmarshal(body2, &gu); err != nil {
			return nil, err
		}
		if gu.ID == 0 {
			return nil, errors.New("github: no user id")
		}
		return &oauthUser{ID: fmt.Sprint(gu.ID), Email: gu.Email}, nil
	case "discord":
		var du struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}
		if err := json.Unmarshal(body2, &du); err != nil {
			return nil, err
		}
		if du.ID == "" {
			return nil, errors.New("discord: no user id")
		}
		return &oauthUser{ID: du.ID, Email: du.Email}, nil
	}
	return nil, ErrOAuthProvider
}

// LoginWithCode performs the full code -> identity -> user -> session flow and
// returns the session token for cookie issuance.
func (s *OAuthService) LoginWithCode(ctx context.Context, provider, code string) (*model.User, string, error) {
	idu, err := s.ExchangeIdentity(ctx, provider, code)
	if err != nil {
		s.log.Warn("oauth exchange failed", "provider", provider, "err", err)
		return nil, "", ErrOAuthExchange
	}
	u, err := s.users.FindOrCreateOAuthUser(ctx, provider, idu.ID, idu.Email)
	if err != nil {
		return nil, "", ErrInternal
	}
	token, err := s.auth.newSession(ctx, u.ID)
	if err != nil {
		return nil, "", ErrInternal
	}
	return u, token, nil
}

func (s *OAuthService) redirectURI(provider string) string {
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	return base + "/api/v1/auth/oauth/" + provider + "/callback"
}

func (s *OAuthService) clientID(provider string) string {
	if provider == "github" {
		return s.cfg.GitHubClientID
	}
	return s.cfg.DiscordClientID
}

func (s *OAuthService) clientSecret(provider string) string {
	if provider == "github" {
		return s.cfg.GitHubClientSecret
	}
	return s.cfg.DiscordClientSecret
}

func (s *OAuthService) endpoints(provider string) (tokenURL, userURL string, headers map[string]string) {
	if provider == "github" {
		return "https://github.com/login/oauth/access_token",
			"https://api.github.com/user",
			map[string]string{"Accept": "application/json"}
	}
	return "https://discord.com/api/oauth2/token",
		"https://discord.com/api/users/@me",
		nil
}

const base64chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

func encodeBase64URL(b []byte) string {
	// stdlib-free base64url without padding.
	var sb strings.Builder
	for i := 0; i < len(b); i += 3 {
		var chunk [3]byte
		n := copy(chunk[:], b[i:])
		v := uint32(chunk[0])<<16 | uint32(chunk[1])<<8 | uint32(chunk[2])
		sb.WriteByte(base64chars[(v>>18)&63])
		sb.WriteByte(base64chars[(v>>12)&63])
		if n > 1 {
			sb.WriteByte(base64chars[(v>>6)&63])
		}
		if n > 2 {
			sb.WriteByte(base64chars[v&63])
		}
	}
	return sb.String()
}

// ClientID exposes a provider's configured client ID (empty = disabled).
func (s *OAuthService) ClientID(provider string) string { return s.clientID(provider) }
