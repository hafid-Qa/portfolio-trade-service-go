package memory

import (
	"context"
	"testing"
	"time"
)

// newTestLimiter returns a limiter on a fake clock the test can advance.
func newTestLimiter(max int, period time.Duration) (*LoginLimiter, *time.Time) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewLoginLimiter(max, period)
	l.now = func() time.Time { return now }
	return l, &now
}

func TestLoginLimiter_AllowsUpToMax(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLimiter(3, time.Minute)

	for i := range 3 {
		if ok, _, _ := l.Allow(ctx, "alice"); !ok {
			t.Fatalf("attempt %d refused, want allowed", i+1)
		}
	}
	ok, retry, err := l.Allow(ctx, "alice")
	if err != nil || ok {
		t.Fatalf("attempt 4 = %v, %v; want refused", ok, err)
	}
	if retry != time.Minute {
		t.Errorf("retryAfter = %v, want 1m (whole window left)", retry)
	}
}

func TestLoginLimiter_WindowExpires(t *testing.T) {
	ctx := context.Background()
	l, now := newTestLimiter(1, time.Minute)

	l.Allow(ctx, "alice")
	if ok, retry, _ := func() (bool, time.Duration, error) {
		*now = now.Add(40 * time.Second)
		return l.Allow(ctx, "alice")
	}(); ok || retry != 20*time.Second {
		t.Errorf("mid-window = %v, retry %v; want refused, 20s", ok, retry)
	}

	*now = now.Add(20 * time.Second) // window over
	if ok, _, _ := l.Allow(ctx, "alice"); !ok {
		t.Error("after window: refused, want allowed")
	}
}

func TestLoginLimiter_ResetAndIndependentKeys(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLimiter(1, time.Minute)

	l.Allow(ctx, "alice")
	if ok, _, _ := l.Allow(ctx, "bob"); !ok {
		t.Error("bob refused because of alice's attempts")
	}
	if ok, _, _ := l.Allow(ctx, "alice"); ok {
		t.Fatal("alice allowed over limit")
	}

	if err := l.Reset(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if ok, _, _ := l.Allow(ctx, "alice"); !ok {
		t.Error("alice refused after Reset")
	}
}

func TestLoginLimiter_PrunesFinishedWindows(t *testing.T) {
	ctx := context.Background()
	l, now := newTestLimiter(1, time.Minute)

	l.Allow(ctx, "alice")
	*now = now.Add(2 * time.Minute)
	l.Allow(ctx, "bob")

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.windows["alice"]; ok {
		t.Error("finished window for alice still stored, want pruned")
	}
}
