package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const revokedTokenKeyPrefix = "revoked_token:"

// TokenDenylist stores revoked token IDs as Redis keys whose TTL is the
// token's remaining lifetime, so entries delete themselves once the token
// would have expired anyway -- no cleanup job needed.
type TokenDenylist struct {
	client *goredis.Client
}

func NewTokenDenylist(client *goredis.Client) *TokenDenylist {
	return &TokenDenylist{client: client}
}

func (d *TokenDenylist) Revoke(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // already expired: the token is rejected without an entry
	}
	if err := d.client.Set(ctx, revokedTokenKeyPrefix+id.String(), 1, ttl).Err(); err != nil {
		return fmt.Errorf("revoking token %s: %w", id, err)
	}
	return nil
}

func (d *TokenDenylist) IsRevoked(ctx context.Context, id uuid.UUID) (bool, error) {
	n, err := d.client.Exists(ctx, revokedTokenKeyPrefix+id.String()).Result()
	if err != nil {
		return false, fmt.Errorf("checking token %s: %w", id, err)
	}
	return n > 0, nil
}
