package repository

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// Cursor is a keyset pagination cursor: (created_at, id) of the last item.
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// EncodeCursor serializes a cursor to base64url.
func EncodeCursor(t time.Time, id string) string {
	raw := fmt.Sprintf("%d|%s", t.UnixNano(), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a cursor; returns error on malformed input.
func DecodeCursor(s string) (*Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("bad cursor")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return nil, fmt.Errorf("bad cursor")
	}
	ns, err := parseInt64(parts[0])
	if err != nil {
		return nil, fmt.Errorf("bad cursor")
	}
	t := time.Unix(0, ns).UTC()
	return &Cursor{CreatedAt: t, ID: parts[1]}, nil
}

func parseInt64(s string) (int64, error) {
	var n int64
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int64(r-'0')
	}
	return n, nil
}
