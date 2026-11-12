package memory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenDenylist_RevokeAndCheck(t *testing.T) {
	ctx := context.Background()
	d := NewTokenDenylist()
	revoked, other := uuid.New(), uuid.New()

	if err := d.Revoke(ctx, revoked, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	if got, _ := d.IsRevoked(ctx, revoked); !got {
		t.Error("IsRevoked(revoked) = false, want true")
	}
	if got, _ := d.IsRevoked(ctx, other); got {
		t.Error("IsRevoked(other) = true, want false")
	}
}

func TestTokenDenylist_ExpiredEntriesAreIgnoredAndPruned(t *testing.T) {
	ctx := context.Background()
	d := NewTokenDenylist()
	alreadyExpired := uuid.New()

	if err := d.Revoke(ctx, alreadyExpired, time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if got, _ := d.IsRevoked(ctx, alreadyExpired); got {
		t.Error("IsRevoked(already expired) = true, want false")
	}

	shortLived := uuid.New()
	if err := d.Revoke(ctx, shortLived, time.Now().Add(10*time.Millisecond)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if got, _ := d.IsRevoked(ctx, shortLived); got {
		t.Error("IsRevoked after expiry = true, want false")
	}

	// The next Revoke prunes the stale entry.
	if err := d.Revoke(ctx, uuid.New(), time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.revoked[shortLived]; ok {
		t.Error("expired entry still stored, want pruned")
	}
}
