package api

import (
	"time"

	"app/internal/domain"
)

// UserRequest's max lengths mirror the users table columns, so oversized
// input is a 400 here rather than a DB error (500). The password cap bounds
// the work argon2 does per request.
type UserRequest struct {
	FullName string `json:"full_name" binding:"required,max=100"`
	Username string `json:"username" binding:"required,alphanum,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

type UserResponse struct {
	ID        int64      `json:"id"`
	FullName  string     `json:"full_name"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// loginUserRequest binds from JSON or a form: Swagger's OAuth2 password flow
// posts application/x-www-form-urlencoded (plus a grant_type field, ignored).
type loginUserRequest struct {
	Username string `json:"username" form:"username" binding:"required,alphanum"`
	Password string `json:"password" form:"password" binding:"required,min=6"`
}

// loginResponse uses the OAuth2 token response field names (RFC 6749 5.1),
// which Swagger UI's password flow reads.
type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// toUserResponse is the only way a user leaves the API: it copies the public
// fields and never the password hash.
func toUserResponse(u domain.UserResult) UserResponse {
	return UserResponse{
		ID:        u.ID(),
		FullName:  u.FullName(),
		Username:  u.UserName(),
		Email:     u.Email(),
		LastLogin: u.LastLogin(),
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
	}
}
