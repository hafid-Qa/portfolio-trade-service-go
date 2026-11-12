package password

import (
	"errors"
	"strings"
	"testing"
)

func mustHash(t *testing.T, plain string) string {
	t.Helper()
	h, err := Hash(plain)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	return h
}

func TestHash_Format(t *testing.T) {
	h := mustHash(t, "correct horse battery staple")

	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("hash = %q, want argon2id PHC string with default params", h)
	}
	if len(h) > 255 {
		t.Errorf("len(hash) = %d, must fit users.password_hash VARCHAR(255)", len(h))
	}
	if strings.Contains(h, "correct horse") {
		t.Error("hash contains the plaintext password")
	}
}

func TestHash_UniqueSalt(t *testing.T) {
	if mustHash(t, "same-password") == mustHash(t, "same-password") {
		t.Error("two hashes of the same password are identical, want a random salt per hash")
	}
}

func TestVerify_Correct(t *testing.T) {
	h := mustHash(t, "s3cret!")
	if err := Verify(h, "s3cret!"); err != nil {
		t.Errorf("Verify(correct) = %v, want nil", err)
	}
}

func TestVerify_Wrong(t *testing.T) {
	h := mustHash(t, "s3cret!")
	for _, wrong := range []string{"s3cret", "S3cret!", "", "s3cret! "} {
		if err := Verify(h, wrong); !errors.Is(err, ErrMismatch) {
			t.Errorf("Verify(%q) = %v, want ErrMismatch", wrong, err)
		}
	}
}

// Hashes made with older/different params must keep verifying after the
// defaults change, because Verify reads params from the hash itself.
func TestVerify_UsesParamsFromHash(t *testing.T) {
	cheap := params{memory: 8 * 1024, time: 1, threads: 1, saltLen: 8, keyLen: 16}
	h, err := hashWith("s3cret!", cheap)
	if err != nil {
		t.Fatalf("hashWith: %v", err)
	}
	if !strings.Contains(h, "$m=8192,t=1,p=1$") {
		t.Fatalf("hash = %q, want custom params encoded", h)
	}
	if err := Verify(h, "s3cret!"); err != nil {
		t.Errorf("Verify = %v, want nil", err)
	}
	if err := Verify(h, "nope"); !errors.Is(err, ErrMismatch) {
		t.Errorf("Verify(wrong) = %v, want ErrMismatch", err)
	}
}

func TestVerify_InvalidHash(t *testing.T) {
	valid := mustHash(t, "s3cret!")
	parts := strings.Split(valid, "$") // ["", "argon2id", "v=19", "m=..,t=..,p=..", salt, key]

	cases := map[string]string{
		"empty":            "",
		"plaintext":        "s3cret!",
		"bcrypt":           "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		"argon2i variant":  strings.Replace(valid, "$argon2id$", "$argon2i$", 1),
		"wrong version":    strings.Replace(valid, "$v=19$", "$v=16$", 1),
		"bad params":       strings.Join([]string{"", parts[1], parts[2], "m=x,t=2,p=1", parts[4], parts[5]}, "$"),
		"zero memory":      strings.Join([]string{"", parts[1], parts[2], "m=0,t=2,p=1", parts[4], parts[5]}, "$"),
		"bad salt base64":  strings.Join([]string{"", parts[1], parts[2], parts[3], "!!!", parts[5]}, "$"),
		"empty key":        strings.Join([]string{"", parts[1], parts[2], parts[3], parts[4], ""}, "$"),
		"too many fields":  valid + "$extra",
		"missing key part": strings.Join(parts[:5], "$"),
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			if err := Verify(h, "s3cret!"); !errors.Is(err, ErrInvalidHash) {
				t.Errorf("Verify = %v, want ErrInvalidHash", err)
			}
		})
	}
}
