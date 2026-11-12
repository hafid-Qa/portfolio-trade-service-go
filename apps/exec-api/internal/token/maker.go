package token

import "time"

// Maker issues and verifies access tokens. The api layer depends on this
// interface only, so the token format (PASETO today) stays swappable.
type Maker interface {
	CreateToken(userID int64, duration time.Duration) (string, error)
	VerifyToken(token string) (*Payload, error)
}
