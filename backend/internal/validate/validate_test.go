package validate

import (
	"errors"
	"testing"
)

func TestURLTable(t *testing.T) {
	p := Params{OwnHost: "short.example", BlockedHosts: map[string]struct{}{"evil.example": {}}}

	cases := []struct {
		name string
		in   string
		want error // nil = valid; specific sentinel otherwise
	}{
		{"valid https", "https://example.com/a?b=1#c", nil},
		{"valid http", "http://example.com", nil},
		{"valid IDN", "https://bücher.example", nil},
		{"empty", "", ErrURLEmpty},
		{"whitespace only", "   ", ErrURLEmpty},
		{"too long", "https://example.com/" + string(make([]byte, 2100)), ErrURLTooLong},
		{"javascript scheme", "javascript:alert(1)", ErrURLScheme},
		{"data scheme", "data:text/html,x", ErrURLScheme},
		{"ftp scheme", "ftp://x.com", ErrURLScheme},
		{"no host", "http://", ErrURLNoHost},
		{"userinfo", "http://user:pass@example.com", ErrURLUserinfo},
		{"localhost", "http://localhost:3000", ErrURLBlocked},
		{"subdomain localhost", "http://api.localhost", ErrURLBlocked},
		{"loopback v4", "http://127.0.0.1", ErrURLBlocked},
		{"private v4", "http://10.0.0.5", ErrURLBlocked},
		{"loopback v6", "http://[::1]", ErrURLBlocked},
		{"link-local", "http://169.254.1.1", ErrURLBlocked},
		{"single label", "http://intranet/", ErrURLBlocked},
		{"own host", "https://short.example/x", ErrURLBlocked},
		{"blocked list", "https://evil.example", ErrURLBlocked},
		{"internal suffix", "https://foo.internal", ErrURLBlocked},
		{"space inside", "https://exa mple.com", ErrURLBadChars},
		{"bad port", "https://example.com:99999", ErrURLBadPort},
		{"port zero", "https://example.com:0", ErrURLBadPort},
		{"unparseable", "https://exa\tmple.com", ErrURLBadChars},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := URL(tc.in, p)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("expected valid, got error %v", err)
				}
				if out == "" {
					t.Fatal("expected normalized output")
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestURLPreservesPathQuery(t *testing.T) {
	p := Params{OwnHost: "own.example", BlockedHosts: map[string]struct{}{}}
	in := "https://EXAMPLE.com/a/b?z=1&y=2#frag"
	out, err := URL(in, p)
	if err != nil {
		t.Fatal(err)
	}
	// host lowercased, path/query/fragment untouched, no trailing slash removal
	if out != "https://example.com/a/b?z=1&y=2#frag" {
		t.Fatalf("unexpected normalization: %q", out)
	}
}

func TestAlias(t *testing.T) {
	valid := []string{"abc", "aB92x", "my-project", "a_1", "0123456789", "a1b"}
	for _, a := range valid {
		if err := Alias(a); err != nil {
			t.Errorf("alias %q should be valid: %v", a, err)
		}
	}
	invalid := []string{"ab", "a b", "-abc", "_abc", "", "33_character_alias_xxxxxxxxxxxxxxx", "api", "LOGIN", "register", "with.dot"}
	for _, a := range invalid {
		if err := Alias(a); err == nil {
			t.Errorf("alias %q should be invalid", a)
		}
	}
}

func TestEmail(t *testing.T) {
	out, err := Email("  User@Example.COM ")
	if err != nil || out != "user@example.com" {
		t.Fatalf("want user@example.com, got %q err %v", out, err)
	}
	bad := []string{"nope", "a@b", "@x.com", "x@", "a b@c.com", ""}
	for _, e := range bad {
		if _, err := Email(e); err == nil {
			t.Errorf("email %q should be invalid", e)
		}
	}
}

func TestPassword(t *testing.T) {
	if err := Password("goodpassword1", "u@example.com"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
	if err := Password("short", "u@example.com"); !errors.Is(err, ErrPasswordLen) {
		t.Error("short password should fail length")
	}
	if err := Password("U@example.com", "u@example.com"); !errors.Is(err, ErrPasswordSame) {
		t.Error("password same as email should fail")
	}
}

func TestReserved(t *testing.T) {
	for _, n := range []string{"api", "API", "Login", "dashboard", "healthz"} {
		if !IsReserved(n) {
			t.Errorf("%q should be reserved", n)
		}
	}
	if IsReserved("mylink") {
		t.Error("mylink should not be reserved")
	}
}
