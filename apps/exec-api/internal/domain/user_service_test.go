package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// fakeUserRepo stores users by username; only the methods UserService uses
// do real work.
type fakeUserRepo struct {
	users        map[string]UserResult
	getErr       error // returned by GetByUsername instead of a lookup
	createdHash  string
	lastLoginFor int64
}

func newFakeUserRepo(users ...UserResult) *fakeUserRepo {
	r := &fakeUserRepo{users: map[string]UserResult{}}
	for _, u := range users {
		r.users[u.UserName()] = u
	}
	return r
}

func (r *fakeUserRepo) Get(ctx context.Context, id int64) (UserResult, error) {
	return UserResult{}, errors.New("not used")
}

func (r *fakeUserRepo) GetByUsername(ctx context.Context, username string) (UserResult, error) {
	if r.getErr != nil {
		return UserResult{}, r.getErr
	}
	u, ok := r.users[username]
	if !ok {
		return UserResult{}, fmt.Errorf("%w: username %q", ErrUserNotFound, username)
	}
	return u, nil
}

func (r *fakeUserRepo) GetByEmail(ctx context.Context, email string) (UserResult, error) {
	return UserResult{}, errors.New("not used")
}

func (r *fakeUserRepo) Create(ctx context.Context, fullName, userName, email, passwordHash string) (UserResult, error) {
	r.createdHash = passwordHash
	return NewUserResult(1, fullName, userName, email, passwordHash, nil, time.Now(), time.Now()), nil
}

func (r *fakeUserRepo) UpdateProfile(ctx context.Context, id int64, fullName, userName, email *string) (UserResult, error) {
	return UserResult{}, errors.New("not used")
}

func (r *fakeUserRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) (UserResult, error) {
	return UserResult{}, errors.New("not used")
}

func (r *fakeUserRepo) UpdateLastLogin(ctx context.Context, id int64) (UserResult, error) {
	r.lastLoginFor = id
	for _, u := range r.users {
		if u.ID() == id {
			now := time.Now()
			return NewUserResult(u.ID(), u.FullName(), u.UserName(), u.Email(), u.PasswordHash(), &now, u.CreatedAt(), now), nil
		}
	}
	return UserResult{}, fmt.Errorf("%w: id %d", ErrUserNotFound, id)
}

// fakeHasher "hashes" by prefixing, and records Verify calls so tests can
// assert the dummy-hash timing defence actually runs.
type fakeHasher struct {
	hashErr     error
	verifyCalls []string // hashes Verify was called with
}

func (h *fakeHasher) Hash(plain string) (string, error) {
	if h.hashErr != nil {
		return "", h.hashErr
	}
	return "hashed:" + plain, nil
}

func (h *fakeHasher) Verify(hash, plain string) error {
	h.verifyCalls = append(h.verifyCalls, hash)
	if !strings.HasPrefix(hash, "hashed:") {
		return errors.New("corrupt hash")
	}
	if hash != "hashed:"+plain {
		return errors.New("mismatch")
	}
	return nil
}

func alice() UserResult {
	return NewUserResult(7, "Alice", "alice", "alice@example.com", "hashed:s3cret!", nil, time.Now(), time.Now())
}

func TestUserService_Register_StoresHashNotPlaintext(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewUserService(repo, &fakeHasher{})

	user, err := svc.Register(context.Background(), "Alice", "alice", "alice@example.com", "s3cret!")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if repo.createdHash != "hashed:s3cret!" {
		t.Errorf("repo got password %q, want the hash", repo.createdHash)
	}
	if user.PasswordHash() == "s3cret!" {
		t.Error("stored password is the plaintext")
	}
}

func TestUserService_Register_HashError(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewUserService(repo, &fakeHasher{hashErr: errors.New("rng failure")})

	if _, err := svc.Register(context.Background(), "Alice", "alice", "alice@example.com", "s3cret!"); err == nil {
		t.Fatal("Register() error = nil, want hashing error")
	}
	if repo.createdHash != "" {
		t.Error("user was created despite hashing failure")
	}
}

func TestUserService_Login_Success(t *testing.T) {
	repo := newFakeUserRepo(alice())
	svc := NewUserService(repo, &fakeHasher{})

	user, err := svc.Login(context.Background(), "alice", "s3cret!")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if user.ID() != 7 {
		t.Errorf("ID() = %d, want 7", user.ID())
	}
	if repo.lastLoginFor != 7 {
		t.Errorf("UpdateLastLogin called for %d, want 7", repo.lastLoginFor)
	}
	if user.LastLogin() == nil {
		t.Error("LastLogin() = nil, want the updated login time")
	}
}

func TestUserService_Login_InvalidCredentials(t *testing.T) {
	corrupt := NewUserResult(8, "Bob", "bob", "bob@example.com", "not-a-hash", nil, time.Now(), time.Now())

	cases := map[string]struct {
		username, password string
	}{
		"wrong password": {"alice", "wrong"},
		"unknown user":   {"mallory", "s3cret!"},
		"corrupt hash":   {"bob", "anything"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			repo := newFakeUserRepo(alice(), corrupt)
			svc := NewUserService(repo, &fakeHasher{})

			_, err := svc.Login(context.Background(), c.username, c.password)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
			}
			if repo.lastLoginFor != 0 {
				t.Error("UpdateLastLogin called on a failed login")
			}
		})
	}
}

// An unknown username must still run a password verification, otherwise it
// returns much faster than a wrong password and leaks which usernames exist.
func TestUserService_Login_UnknownUserStillVerifies(t *testing.T) {
	hasher := &fakeHasher{}
	svc := NewUserService(newFakeUserRepo(), hasher)

	if _, err := svc.Login(context.Background(), "mallory", "whatever"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if len(hasher.verifyCalls) != 1 {
		t.Fatalf("Verify called %d times, want 1 (dummy hash)", len(hasher.verifyCalls))
	}
}

// A DB outage is not a credentials problem: it must not be reported as
// ErrInvalidCredentials (401) but passed through (500).
func TestUserService_Login_RepoErrorIsNotInvalidCredentials(t *testing.T) {
	repo := newFakeUserRepo(alice())
	repo.getErr = errors.New("connection refused")
	svc := NewUserService(repo, &fakeHasher{})

	_, err := svc.Login(context.Background(), "alice", "s3cret!")
	if err == nil {
		t.Fatal("Login() error = nil, want repo error")
	}
	if errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Login() error = %v, must not be ErrInvalidCredentials", err)
	}
}
