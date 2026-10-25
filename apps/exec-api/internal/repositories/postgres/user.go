package postgres

import (
	"app/internal/domain"
	"app/internal/sqlc"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type UserRepo struct {
	store sqlc.Store
}

func NewUserRepo(store sqlc.Store) *UserRepo {
	return &UserRepo{store: store}
}

func (r *UserRepo) Get(ctx context.Context, id int64) (domain.UserResult, error) {
	row, err := r.store.GetUserByID(ctx, int32(id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserResult{}, fmt.Errorf("%w: id %d", domain.ErrUserNotFound, id)
	}
	if err != nil {
		return domain.UserResult{}, fmt.Errorf("getting user %d: %w", id, err)
	}
	return rowToUser(row)
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (domain.UserResult, error) {
	row, err := r.store.GetUserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserResult{}, fmt.Errorf("%w: username %q", domain.ErrUserNotFound, username)
	}
	if err != nil {
		return domain.UserResult{}, fmt.Errorf("getting user %q: %w", username, err)
	}
	return rowToUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (domain.UserResult, error) {
	row, err := r.store.GetUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserResult{}, fmt.Errorf("%w: email %q", domain.ErrUserNotFound, email)
	}
	if err != nil {
		return domain.UserResult{}, fmt.Errorf("getting user %q: %w", email, err)
	}
	return rowToUser(row)
}

func (r *UserRepo) Create(ctx context.Context, fullName string, userName string, email string, passwordHash string) (domain.UserResult, error) {
	row, err := r.store.CreateUser(ctx, sqlc.CreateUserParams{
		FullName:     fullName,
		Username:     userName,
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return domain.UserResult{}, fmt.Errorf("creating user: %w", err)
	}
	return rowToUser(row)
}

func (r *UserRepo) UpdateProfile(ctx context.Context, id int64, fullName *string, userName *string, email *string) (domain.UserResult, error) {
	row, err := r.store.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		ID:       int32(id),
		FullName: toNullString(fullName),
		Username: toNullString(userName),
		Email:    toNullString(email),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserResult{}, fmt.Errorf("%w: id %d", domain.ErrUserNotFound, id)
	}
	if err != nil {
		return domain.UserResult{}, fmt.Errorf("updating profile for user %d: %w", id, err)
	}
	return rowToUser(row)
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) (domain.UserResult, error) {
	row, err := r.store.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           int32(id),
		PasswordHash: passwordHash,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserResult{}, fmt.Errorf("%w: id %d", domain.ErrUserNotFound, id)
	}
	if err != nil {
		return domain.UserResult{}, fmt.Errorf("updating password for user %d: %w", id, err)
	}
	return rowToUser(row)
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id int64) (domain.UserResult, error) {
	row, err := r.store.UpdateUserLastLogin(ctx, int32(id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserResult{}, fmt.Errorf("%w: id %d", domain.ErrUserNotFound, id)
	}
	if err != nil {
		return domain.UserResult{}, fmt.Errorf("updating last login for user %d: %w", id, err)
	}
	return rowToUser(row)
}

// toNullString converts an optional field (nil means "leave unchanged", per
// UpdateUserProfile's COALESCE pattern) into the sql.NullString the generated
// query expects -- Valid: false signals COALESCE to skip it.
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func rowToUser(row sqlc.User) (domain.UserResult, error) {
	var lastLogin *time.Time
	if row.LastLogin.Valid {
		lastLogin = &row.LastLogin.Time
	}
	return domain.NewUserResult(int64(row.ID), row.FullName, row.Username, row.Email, row.PasswordHash, lastLogin, row.CreatedAt, row.UpdatedAt), nil
}
