-- name: CreateTrade :one
INSERT INTO trades (user_id, amount, target_portfolio, orders, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
RETURNING *;
