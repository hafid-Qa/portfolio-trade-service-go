package token

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidToken covers every verification failure (bad key, tampering,
// expiry, wrong issuer/audience). Deliberately a single error: callers map
// it to a generic 401 without telling the client why.
var ErrInvalidToken = errors.New("invalid token")

// Payload is the library-independent view of a verified token's claims.
type Payload struct {
	ID        uuid.UUID `json:"id"`
	UserID    int64     `json:"user_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

func NewPayload(userID int64, duration time.Duration) (*Payload, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	now := time.Now()

	return &Payload{
		ID:        id,
		UserID:    userID,
		IssuedAt:  now,
		ExpiredAt: now.Add(duration),
	}, nil
}
