package service

import (
	"context"
	"testing"
	"time"
)

func TestGenerateCode(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 1000; i++ {
		c, err := generateCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != codeLen {
			t.Fatalf("code %q has wrong length", c)
		}
		for _, r := range c {
			ok := (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
			if !ok {
				t.Fatalf("code %q has non-base62 char", c)
			}
		}
		seen[c] = struct{}{}
	}
	if len(seen) < 990 {
		t.Fatalf("codes not unique enough: %d unique of 1000", len(seen))
	}
}

func TestClickBufferAccumulates(t *testing.T) {
	b := &ClickBuffer{byCode: map[string]*clickAccum{}}
	now := time.Now().UTC()
	b.Add("abc", now)
	b.Add("abc", now.Add(time.Second))
	b.Add("xyz", now)
	if len(b.byCode) != 2 {
		t.Fatalf("want 2 codes, got %d", len(b.byCode))
	}
	if b.byCode["abc"].clicks != 2 {
		t.Fatalf("want 2 clicks for abc, got %d", b.byCode["abc"].clicks)
	}
	if !b.byCode["abc"].lastAccess.Equal(now.Add(time.Second)) {
		t.Fatal("lastAccess should track the newest click")
	}
}

func TestClickBufferEmptyFlushNoop(t *testing.T) {
	b := NewClickBuffer(nil, time.Second, testLogger())
	if err := b.Flush(context.Background()); err != nil {
		t.Fatalf("empty flush should be a no-op, got %v", err)
	}
}

func TestRedirectCacheTTLAndNegative(t *testing.T) {
	c := newRedirectCache(100, 50*time.Millisecond, 30*time.Millisecond)
	e := &cacheEntry{ID: "1", TargetURL: "https://x.com", IsActive: true}
	c.Put("abc", e)
	got, ok := c.Get("abc")
	if !ok || got != e {
		t.Fatal("positive cache miss")
	}
	c.PutNegative("gone")
	if g, ok := c.Get("gone"); !ok || g != nil {
		t.Fatalf("negative cache should return (nil,true), got (%v,%v)", g, ok)
	}
	time.Sleep(60 * time.Millisecond)
	if _, ok := c.Get("abc"); ok {
		t.Fatal("positive entry should expire")
	}
	if _, ok := c.Get("gone"); ok {
		t.Fatal("negative entry should expire")
	}
	c.Put("abc", e)
	c.Invalidate("abc")
	if _, ok := c.Get("abc"); ok {
		t.Fatal("entry should be invalidated")
	}
}

func TestSingleflightElection(t *testing.T) {
	sf := newSingleflight()
	release := sf.Do("k")
	if release == nil {
		t.Fatal("first caller should be elected")
	}
	// The second caller blocks inside Do until the elected loader releases.
	unblocked := make(chan struct{})
	go func() {
		if second := sf.Do("k"); second != nil {
			t.Error("second caller should not be elected")
		}
		close(unblocked)
	}()
	time.Sleep(20 * time.Millisecond)
	release()
	select {
	case <-unblocked:
	case <-time.After(time.Second):
		t.Fatal("second caller not released")
	}
}

func TestURLCreateValidationErrors(t *testing.T) {
	s := &URLService{cfg: testConfig()}
	_, err := s.Create(context.Background(), "", "", "", nil)
	var ve *ValidationError
	if err == nil {
		t.Fatal("expected validation error for empty input")
	}
	if !asVE(err, &ve) || !ve.Has() {
		t.Fatal("expected field errors")
	}
	_, err = s.Create(context.Background(), "", "https://ok.example", "myalias", nil)
	if err != ErrAliasRequiresLogin {
		t.Fatalf("want ErrAliasRequiresLogin, got %v", err)
	}
}

func asVE(err error, target **ValidationError) bool {
	ve, ok := err.(*ValidationError)
	if ok {
		*target = ve
	}
	return ok
}
