package token

import (
	"fmt"
	"strconv"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

const (
	// issuer and accessAudience are checked on every verify.
	issuer         = "exec-api"
	accessAudience = "access"
)

// Compile-time check that PASETOMaker satisfies Maker.
var _ Maker = (*PASETOMaker)(nil)

type PASETOMaker struct {
	symmetricKey paseto.V4SymmetricKey
	parser       paseto.Parser
}

// NewPASETOMaker takes the v4.local key as 64 hex characters (32 random
// bytes), e.g. generated with `openssl rand -hex 32`.
func NewPASETOMaker(symmetricKeyHex string) (*PASETOMaker, error) {
	key, err := paseto.V4SymmetricKeyFromHex(symmetricKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid token symmetric key (want 64 hex chars): %w", err)
	}

	// NewParser preloads the NotExpired rule. Rules read time.Now() at check
	// time, so one parser can be reused across requests.
	parser := paseto.NewParser()
	parser.AddRule(
		paseto.NotBeforeNbf(),
		paseto.IssuedBy(issuer),
		paseto.ForAudience(accessAudience),
	)

	return &PASETOMaker{symmetricKey: key, parser: parser}, nil
}

func (maker *PASETOMaker) CreateToken(userID int64, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(userID, duration)
	if err != nil {
		return "", nil, err
	}

	t := paseto.NewToken()
	t.SetJti(payload.ID.String())
	t.SetIssuer(issuer)
	t.SetAudience(accessAudience)
	t.SetSubject(strconv.FormatInt(payload.UserID, 10))
	t.SetIssuedAt(payload.IssuedAt)
	t.SetNotBefore(payload.IssuedAt)
	t.SetExpiration(payload.ExpiredAt)

	return t.V4Encrypt(maker.symmetricKey, nil), payload, nil
}

func (maker *PASETOMaker) VerifyToken(token string) (*Payload, error) {
	t, err := maker.parser.ParseV4Local(maker.symmetricKey, token, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	payload, err := payloadFromToken(t)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return payload, nil
}

func payloadFromToken(t *paseto.Token) (*Payload, error) {
	jti, err := t.GetJti()
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(jti)
	if err != nil {
		return nil, err
	}
	sub, err := t.GetSubject()
	if err != nil {
		return nil, err
	}
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return nil, err
	}
	issuedAt, err := t.GetIssuedAt()
	if err != nil {
		return nil, err
	}
	expiredAt, err := t.GetExpiration()
	if err != nil {
		return nil, err
	}

	return &Payload{
		ID:        id,
		UserID:    userID,
		IssuedAt:  issuedAt,
		ExpiredAt: expiredAt,
	}, nil
}
