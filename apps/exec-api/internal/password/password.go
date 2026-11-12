// Package password hashes and verifies user passwords with argon2id.
//
// Hashes are stored in the standard PHC string format:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<base64 salt>$<base64 key>
//
// The parameters travel with each hash, so they can be raised later without
// invalidating existing hashes: Verify always uses the parameters embedded in
// the stored hash, not the current defaults.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	// ErrMismatch means the password does not match the hash.
	ErrMismatch = errors.New("password does not match")
	// ErrInvalidHash means the stored hash is not a valid argon2id PHC string.
	ErrInvalidHash = errors.New("invalid password hash")
)

// params are the argon2id cost settings.
type params struct {
	memory  uint32 // KiB
	time    uint32 // iterations
	threads uint8
	saltLen uint32
	keyLen  uint32
}

// defaultParams follows the OWASP Password Storage Cheat Sheet minimum for
// argon2id (19 MiB, 2 iterations, 1 thread). Memory is per hash in flight,
// so concurrent logins multiply it -- keep that in mind before raising it.
var defaultParams = params{
	memory:  19 * 1024,
	time:    2,
	threads: 1,
	saltLen: 16,
	keyLen:  32,
}

// Hash returns the argon2id PHC string for plain, using a fresh random salt.
func Hash(plain string) (string, error) {
	return hashWith(plain, defaultParams)
}

func hashWith(plain string, p params) (string, error) {
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	key := argon2.IDKey([]byte(plain), salt, p.time, p.memory, p.threads, p.keyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.time, p.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify reports whether plain matches encodedHash. It returns ErrMismatch
// for a wrong password and ErrInvalidHash for a malformed hash; callers doing
// login should treat both (and "user not found") as the same generic failure.
func Verify(encodedHash, plain string) error {
	p, salt, key, err := decode(encodedHash)
	if err != nil {
		return err
	}

	other := argon2.IDKey([]byte(plain), salt, p.time, p.memory, p.threads, p.keyLen)

	// Constant-time compare so response timing doesn't leak how many bytes matched.
	if subtle.ConstantTimeCompare(key, other) != 1 {
		return ErrMismatch
	}
	return nil
}

func decode(encodedHash string) (params, []byte, []byte, error) {
	// "$argon2id$v=19$m=..,t=..,p=..$salt$key" splits into 6 parts, the first empty.
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return params{}, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return params{}, nil, nil, ErrInvalidHash
	}

	var p params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return params{}, nil, nil, ErrInvalidHash
	}
	if p.memory == 0 || p.time == 0 || p.threads == 0 {
		return params{}, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return params{}, nil, nil, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return params{}, nil, nil, ErrInvalidHash
	}
	p.saltLen = uint32(len(salt))
	p.keyLen = uint32(len(key))

	return p, salt, key, nil
}

// Argon2Hasher adapts Hash/Verify to the domain.PasswordHasher interface.
type Argon2Hasher struct{}

func (Argon2Hasher) Hash(plain string) (string, error) { return Hash(plain) }

func (Argon2Hasher) Verify(hash, plain string) error { return Verify(hash, plain) }
