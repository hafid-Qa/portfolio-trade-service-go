package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"app/config"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

var testClient *goredis.Client

// TestMain connects to the real broker, the same pattern as the queue tests
// (and db_test for Postgres). Every key a test writes is deleted in Cleanup.
func TestMain(m *testing.M) {
	cfg, err := config.LoadConfig(context.Background())
	if err != nil {
		panic(err)
	}
	testClient = goredis.NewClient(&goredis.Options{
		Addr:     cfg.BrokerAddr,
		Username: cfg.BrokerUser,
		Password: cfg.BrokerPassword,
	})
	os.Exit(m.Run())
}

func cleanupKey(t *testing.T, id uuid.UUID) {
	t.Helper()
	t.Cleanup(func() { testClient.Del(context.Background(), revokedTokenKeyPrefix+id.String()) })
}

func TestTokenDenylist_RevokeAndCheck(t *testing.T) {
	ctx := context.Background()
	d := NewTokenDenylist(testClient)
	revoked, other := uuid.New(), uuid.New()
	cleanupKey(t, revoked)

	if err := d.Revoke(ctx, revoked, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	got, err := d.IsRevoked(ctx, revoked)
	if err != nil || !got {
		t.Errorf("IsRevoked(revoked) = %v, %v; want true, nil", got, err)
	}
	got, err = d.IsRevoked(ctx, other)
	if err != nil || got {
		t.Errorf("IsRevoked(other) = %v, %v; want false, nil", got, err)
	}
}

// The key must expire with the token, so Redis never accumulates entries.
func TestTokenDenylist_TTLMatchesTokenExpiry(t *testing.T) {
	ctx := context.Background()
	d := NewTokenDenylist(testClient)
	id := uuid.New()
	cleanupKey(t, id)

	if err := d.Revoke(ctx, id, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	ttl, err := testClient.TTL(ctx, revokedTokenKeyPrefix+id.String()).Result()
	if err != nil {
		t.Fatalf("TTL() error = %v", err)
	}
	if ttl <= 0 || ttl > time.Minute {
		t.Errorf("TTL = %v, want in (0, 1m]", ttl)
	}
}

func TestTokenDenylist_AlreadyExpiredIsNotStored(t *testing.T) {
	ctx := context.Background()
	d := NewTokenDenylist(testClient)
	id := uuid.New()
	cleanupKey(t, id)

	if err := d.Revoke(ctx, id, time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	n, err := testClient.Exists(ctx, revokedTokenKeyPrefix+id.String()).Result()
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if n != 0 {
		t.Error("key stored for an already-expired token, want none")
	}
}
