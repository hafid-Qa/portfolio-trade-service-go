-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;
-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = LOWER($1);
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = LOWER($1);
-- name: CreateUser :one
INSERT INTO users (full_name, username, email, password_hash, created_at, updated_at)
VALUES (sqlc.arg(full_name), LOWER(sqlc.arg(username)), LOWER(sqlc.arg(email)), sqlc.arg(password_hash), NOW(), NOW())
RETURNING *;
-- name: UpdateUserProfile :one
UPDATE users SET
  full_name = COALESCE(sqlc.narg(full_name), full_name),
  username  = COALESCE(LOWER(sqlc.narg(username)), username),
  email     = COALESCE(LOWER(sqlc.narg(email)), email),
  updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;
-- name: UpdateUserPassword :one
UPDATE users SET password_hash = sqlc.arg(password_hash), updated_at = NOW() WHERE id = sqlc.arg(id) RETURNING *;
-- name: UpdateUserLastLogin :one
UPDATE users SET last_login = NOW() WHERE id = sqlc.arg(id) RETURNING *;
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
