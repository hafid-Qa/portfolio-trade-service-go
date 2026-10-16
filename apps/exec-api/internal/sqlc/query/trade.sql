-- name: CreateTrade :one
INSERT INTO trades (user_id, amount, target_portfolio, orders, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
RETURNING *;
-- name: GetTradeByID :one
SELECT * FROM trades WHERE id = $1;
-- name: GetTradesByUserID :many
SELECT * FROM trades WHERE user_id = $1 ORDER BY created_at DESC;
-- name: UpdateTradeStatus :one
UPDATE trades SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

