package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TokenDenylist is the in-process counterpart of redis.TokenDenylist, used
// when config.DataSource == "memory" and in tests. Revocations are lost on
// restart and not shared between instances.
type TokenDenylist struct {
	mu      sync.Mutex
	revoked map[uuid.UUID]time.Time // token ID -> when the entry can be dropped
}

func NewTokenDenylist() *TokenDenylist {
	return &TokenDenylist{revoked: map[uuid.UUID]time.Time{}}
}

func (d *TokenDenylist) Revoke(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	// Drop entries for tokens that have expired anyway, so the map can't grow forever.
	for k, exp := range d.revoked {
		if !now.Before(exp) {
			delete(d.revoked, k)
		}
	}
	if now.Before(expiresAt) {
		d.revoked[id] = expiresAt
	}
	return nil
}

func (d *TokenDenylist) IsRevoked(ctx context.Context, id uuid.UUID) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	exp, ok := d.revoked[id]
	return ok && time.Now().Before(exp), nil
}
