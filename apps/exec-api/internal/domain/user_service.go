package domain

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type UserService struct {
	userRepo UserRepository
	hasher   PasswordHasher
	// dummyHash is verified against when the username doesn't exist, so an
	// unknown user takes as long to reject as a wrong password and response
	// timing can't be used to discover which usernames exist.
	dummyHash func() (string, error)
}

func NewUserService(userRepo UserRepository, hasher PasswordHasher) *UserService {
	return &UserService{
		userRepo: userRepo,
		hasher:   hasher,
		dummyHash: sync.OnceValues(func() (string, error) {
			return hasher.Hash("dummy-password-for-timing")
		}),
	}
}

// Register hashes plainPassword and creates the user. The plaintext never
// reaches the repository.
func (s *UserService) Register(ctx context.Context, fullName, username, email, plainPassword string) (UserResult, error) {
	hash, err := s.hasher.Hash(plainPassword)
	if err != nil {
		return UserResult{}, fmt.Errorf("hashing password: %w", err)
	}
	return s.userRepo.Create(ctx, fullName, username, email, hash)
}

// Login checks the credentials and records the login. Every credential
// failure (unknown user, wrong password, corrupt stored hash) returns an
// error matching ErrInvalidCredentials; the wrapped cause is for server-side
// logs only and must not be sent to the client. Infrastructure failures
// (e.g. the DB is down) are returned as-is so they surface as 500, not 401.
func (s *UserService) Login(ctx context.Context, username, plainPassword string) (UserResult, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if errors.Is(err, ErrUserNotFound) {
		if dummy, hErr := s.dummyHash(); hErr == nil {
			_ = s.hasher.Verify(dummy, plainPassword) // result ignored: only the elapsed time matters
		}
		return UserResult{}, fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
	}
	if err != nil {
		return UserResult{}, err
	}

	if err := s.hasher.Verify(user.PasswordHash(), plainPassword); err != nil {
		return UserResult{}, fmt.Errorf("%w: user %d: %v", ErrInvalidCredentials, user.ID(), err)
	}

	return s.userRepo.UpdateLastLogin(ctx, user.ID())
}

// GetUser returns the user with this ID, or an error matching ErrUserNotFound.
func (s *UserService) GetUser(ctx context.Context, id int64) (UserResult, error) {
	return s.userRepo.Get(ctx, id)
}

// UpdateProfile changes the given fields; a nil pointer leaves that field as
// it is. A username/email already used by someone else matches
// ErrUserAlreadyExists.
func (s *UserService) UpdateProfile(ctx context.Context, id int64, fullName, username, email *string) (UserResult, error) {
	return s.userRepo.UpdateProfile(ctx, id, fullName, username, email)
}

// ChangePassword replaces the user's password after checking the current
// one. A wrong current password matches ErrInvalidCredentials (cause wrapped
// for logs only, like Login).
func (s *UserService) ChangePassword(ctx context.Context, id int64, currentPassword, newPassword string) error {
	user, err := s.userRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.hasher.Verify(user.PasswordHash(), currentPassword); err != nil {
		return fmt.Errorf("%w: user %d: %v", ErrInvalidCredentials, id, err)
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	_, err = s.userRepo.UpdatePassword(ctx, id, hash)
	return err
}
