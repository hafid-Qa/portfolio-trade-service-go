package domain

import "time"

type UserResult struct {
	id           int64
	fullName     string
	userName     string
	email        string
	passwordHash string
	lastLogin    *time.Time
	createdAt    time.Time
	updatedAt    time.Time
}

func (u UserResult) ID() int64             { return u.id }
func (u UserResult) FullName() string      { return u.fullName }
func (u UserResult) UserName() string      { return u.userName }
func (u UserResult) Email() string         { return u.email }
func (u UserResult) PasswordHash() string  { return u.passwordHash }
func (u UserResult) LastLogin() *time.Time { return u.lastLogin }
func (u UserResult) CreatedAt() time.Time  { return u.createdAt }
func (u UserResult) UpdatedAt() time.Time  { return u.updatedAt }

func NewUserResult(id int64, fullName, userName, email, passwordHash string, lastLogin *time.Time, createdAt, updatedAt time.Time) UserResult {
	return UserResult{
		id:           id,
		fullName:     fullName,
		userName:     userName,
		email:        email,
		passwordHash: passwordHash,
		lastLogin:    lastLogin,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}
