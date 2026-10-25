package api

import "time"

type UserResponse struct {
	ID        int64      `json:"id"`
	FullName  string     `json:"full_name"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
