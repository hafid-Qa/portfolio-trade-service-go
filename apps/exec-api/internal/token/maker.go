package token

import "time"

// Maker issues and verifies access tokens. The api layer depends on this
// interface only, so the token format (PASETO today) stays swappable.
type Maker interface {
	// CreateToken returns the encoded token plus its payload, so callers
	// (e.g. login) can report expires_at without re-parsing the token.
	CreateToken(userID int64, duration time.Duration) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}
