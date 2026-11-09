package token

import (
	"errors"
	"strings"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
)

const (
	testKeyHex  = "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"
	otherKeyHex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
)

func newTestMaker(t *testing.T, keyHex string) *PASETOMaker {
	t.Helper()
	maker, err := NewPASETOMaker(keyHex)
	if err != nil {
		t.Fatalf("NewPASETOMaker: %v", err)
	}
	return maker
}

func TestNewPASETOMaker_InvalidKey(t *testing.T) {
	for name, key := range map[string]string{
		"too short":       "abcd",
		"raw 32 chars":    "12345678901234567890123456789012",
		"not hex, 64 len": strings.Repeat("z", 64),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewPASETOMaker(key); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestPASETOMaker_CreateAndVerify(t *testing.T) {
	maker := newTestMaker(t, testKeyHex)
	duration := time.Minute

	tok, created, err := maker.CreateToken(42, duration)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if !strings.HasPrefix(tok, "v4.local.") {
		t.Fatalf("expected v4.local token, got %q", tok)
	}

	got, err := maker.VerifyToken(tok)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if got.ID != created.ID || got.UserID != 42 {
		t.Errorf("verified payload %+v does not match created %+v", got, created)
	}
	if !got.ExpiredAt.Equal(created.ExpiredAt.Truncate(time.Second)) {
		t.Errorf("expired_at = %v, want %v", got.ExpiredAt, created.ExpiredAt)
	}
}

func TestPASETOMaker_InvalidToken(t *testing.T) {
	maker := newTestMaker(t, testKeyHex)

	wrongKeyTok, _, err := newTestMaker(t, otherKeyHex).CreateToken(42, time.Minute)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	expiredTok, _, err := maker.CreateToken(42, -time.Minute)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	// Correctly encrypted with our key, but minted for a different audience
	// -- must not pass as an access token.
	key, err := paseto.V4SymmetricKeyFromHex(testKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	foreign := paseto.NewToken()
	foreign.SetJti("6f1c1d3e-8f0a-4b2c-9d6e-1a2b3c4d5e6f")
	foreign.SetIssuer(issuer)
	foreign.SetAudience("other")
	foreign.SetSubject("42")
	foreign.SetIssuedAt(time.Now())
	foreign.SetNotBefore(time.Now())
	foreign.SetExpiration(time.Now().Add(time.Minute))
	wrongAudienceTok := foreign.V4Encrypt(key, nil)

	for name, input := range map[string]string{
		"wrong key":      wrongKeyTok,
		"expired":        expiredTok,
		"wrong audience": wrongAudienceTok,
		"garbage":        "not-a-token",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := maker.VerifyToken(input); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}
