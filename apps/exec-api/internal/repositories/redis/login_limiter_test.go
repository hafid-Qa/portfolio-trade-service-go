package redis

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Uses the real broker via testClient (see token_denylist_test.go's TestMain).
func testLimiterKey(t *testing.T) string {
	t.Helper()
	key := "test-" + uuid.NewString()
	t.Cleanup(func() { testClient.Del(context.Background(), "rate:"+loginAttemptsKeyPrefix+key) })
	return key
}

func TestLoginLimiter_AllowsUpToMaxThenRefuses(t *testing.T) {
	ctx := context.Background()
	l := NewLoginLimiter(testClient, 3, time.Minute)
	key := testLimiterKey(t)

	for i := range 3 {
		ok, _, err := l.Allow(ctx, key)
		if err != nil || !ok {
			t.Fatalf("attempt %d = %v, %v; want allowed", i+1, ok, err)
		}
	}
	ok, retry, err := l.Allow(ctx, key)
	if err != nil || ok {
		t.Fatalf("attempt 4 = %v, %v; want refused", ok, err)
	}
	if retry <= 0 || retry > time.Minute {
		t.Errorf("retryAfter = %v, want in (0, 1m]", retry)
	}
}

func TestLoginLimiter_ResetAndIndependentKeys(t *testing.T) {
	ctx := context.Background()
	l := NewLoginLimiter(testClient, 1, time.Minute)
	alice, bob := testLimiterKey(t), testLimiterKey(t)

	if ok, _, _ := l.Allow(ctx, alice); !ok {
		t.Fatal("first attempt refused")
	}
	if ok, _, _ := l.Allow(ctx, bob); !ok {
		t.Error("bob refused because of alice's attempts")
	}
	if ok, _, _ := l.Allow(ctx, alice); ok {
		t.Fatal("alice allowed over limit")
	}

	if err := l.Reset(ctx, alice); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if ok, _, err := l.Allow(ctx, alice); err != nil || !ok {
		t.Errorf("after Reset = %v, %v; want allowed", ok, err)
	}
}
