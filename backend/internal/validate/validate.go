// Package validate implements URL, alias, email and password validation rules
// plus the central reserved-alias list.
package validate

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/idna"
)

// MaxURLLen is the maximum accepted target URL length.
const MaxURLLen = 2048

// AliasRe is the allowed custom alias shape (3-32 chars, case-sensitive).
var AliasRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{2,31}$`)

// CodeRe matches a stored/generated code (same shape as an alias).
var CodeRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{2,31}$`)

// emailRe is a deliberately simple email shape check (per PRD §9.6).
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// reserved holds aliases that must never be used as a short code. Compared
// case-insensitively.
var reserved = func() map[string]struct{} {
	names := []string{
		"api", "login", "logout", "register", "signup", "signin",
		"dashboard", "urls", "settings", "auth", "assets", "static",
		"healthz", "readyz", "admin", "about", "terms", "privacy",
		"help", "support", "www", "app", "new", "null", "undefined",
	}
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}()

// IsReserved reports whether name is reserved (case-insensitive).
func IsReserved(name string) bool {
	_, ok := reserved[strings.ToLower(name)]
	return ok
}

// Errors returned by validation helpers. Handlers map these to API errors.
var (
	ErrURLEmpty     = errors.New("url is required")
	ErrURLTooLong   = errors.New("url is too long")
	ErrURLBadChars  = errors.New("url contains control or whitespace characters")
	ErrURLParse     = errors.New("url is not valid")
	ErrURLScheme    = errors.New("url must use http or https")
	ErrURLNoHost    = errors.New("url must include a host")
	ErrURLUserinfo  = errors.New("url must not include credentials")
	ErrURLBadPort   = errors.New("url port is out of range")
	ErrURLBlocked   = errors.New("url host is not allowed")
	ErrAliasInvalid = errors.New("alias has an invalid format")
	ErrAliasTaken   = errors.New("alias is already taken")
	ErrEmailInvalid = errors.New("email is not valid")
	ErrPasswordLen  = errors.New("password must be 8-128 characters")
	ErrPasswordSame = errors.New("password must differ from email")
	ErrExpiryRange  = errors.New("expiration is out of range")
)

// Params carries configuration inputs for URL validation.
type Params struct {
	OwnHost      string              // host[:port] of BASE_URL, lowercased
	BlockedHosts map[string]struct{} // extra blocked hostnames (lowercased)
	AllowPrivate bool                // test-only escape hatch; never true in prod
}

// URL normalizes and validates a target URL, returning the normalized form.
// The path, query and fragment are preserved verbatim.
func URL(raw string, p Params) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ErrURLEmpty
	}
	if len(s) > MaxURLLen {
		return "", ErrURLTooLong
	}
	for _, r := range s {
		if r <= 0x20 || r == 0x7f {
			return "", ErrURLBadChars
		}
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", ErrURLParse
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", ErrURLScheme
	}
	if u.Host == "" {
		return "", ErrURLNoHost
	}
	if u.User != nil {
		return "", ErrURLUserinfo
	}

	host := u.Hostname()
	if host == "" {
		return "", ErrURLNoHost
	}
	port := u.Port()
	if port != "" {
		if !validPort(port) {
			return "", ErrURLBadPort
		}
	}

	// IP literals bypass IDN; bracketed IPv6 is handled via u.Hostname().
	var ascii string
	if ip := net.ParseIP(host); ip != nil {
		ascii = ip.String()
	} else {
		var err error
		ascii, err = idna.Lookup.ToASCII(strings.ToLower(host))
		if err != nil {
			return "", ErrURLParse
		}
		ascii = strings.TrimSuffix(ascii, ".")
	}

	if err := checkHost(ascii, p); err != nil {
		return "", err
	}

	// Rebuild: keep scheme lowercased, host normalized, path/query/fragment as-is.
	u.Scheme = scheme
	if strings.Contains(ascii, ":") {
		u.Host = "[" + ascii + "]"
		if port != "" {
			u.Host = "[" + ascii + "]:" + port
		}
	} else {
		u.Host = ascii
		if port != "" {
			u.Host = ascii + ":" + port
		}
	}
	return u.String(), nil
}

func validPort(p string) bool {
	n := 0
	for _, r := range p {
		if r < '0' || r > '9' {
			return false
		}
		n = n*10 + int(r-'0')
		if n > 65535 {
			return false
		}
	}
	return n >= 1
}

func checkHost(host string, p Params) error {
	if host == "" {
		return ErrURLNoHost
	}
	if _, blocked := p.BlockedHosts[host]; blocked {
		return ErrURLBlocked
	}
	if p.OwnHost != "" && strings.EqualFold(host, p.OwnHost) {
		return ErrURLBlocked
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return ErrURLBlocked
	}
	if ip := net.ParseIP(host); ip != nil {
		if !p.AllowPrivate && !ip.IsGlobalUnicast() {
			return ErrURLBlocked
		}
		if !p.AllowPrivate && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()) {
			return ErrURLBlocked
		}
		return nil
	}
	// Non-IP: single-label hosts are local names unless a dot is present.
	if !strings.Contains(lower, ".") {
		return ErrURLBlocked
	}
	if isPrivateLiteralName(lower) {
		return ErrURLBlocked
	}
	return nil
}

// isPrivateLiteralName blocks common internal-only suffixes.
func isPrivateLiteralName(host string) bool {
	for _, suf := range []string{".internal", ".local", ".intranet", ".corp", ".home", ".lan"} {
		if strings.HasSuffix(host, suf) {
			return true
		}
	}
	return false
}

// Alias validates a custom alias and reports whether it collides with the
// reserved list.
func Alias(alias string) error {
	if !AliasRe.MatchString(alias) {
		return ErrAliasInvalid
	}
	if IsReserved(alias) {
		return ErrAliasInvalid
	}
	return nil
}

// Email normalizes (lowercase, trim) and validates an email address.
func Email(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if len(s) < 3 || len(s) > 254 || !emailRe.MatchString(s) {
		return "", ErrEmailInvalid
	}
	return s, nil
}

// Password validates password length and that it differs from the email.
func Password(pw, email string) error {
	if len(pw) < 8 || len(pw) > 128 {
		return ErrPasswordLen
	}
	if strings.EqualFold(pw, email) {
		return ErrPasswordSame
	}
	return nil
}
