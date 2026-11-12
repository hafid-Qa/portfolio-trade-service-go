package token

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Denylist records tokens revoked before their natural expiry (logout).
// Tokens are stateless, so this is the only server-side state auth needs: an
// entry only has to live until the token would have expired anyway.
type Denylist interface {
	// Revoke marks the token with this ID as revoked until expiresAt.
	Revoke(ctx context.Context, id uuid.UUID, expiresAt time.Time) error
	IsRevoked(ctx context.Context, id uuid.UUID) (bool, error)
}
