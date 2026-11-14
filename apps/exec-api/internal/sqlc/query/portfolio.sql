-- name: GetPortfolioByUserID :one
SELECT * FROM portfolios WHERE user_id = $1;

-- name: ListPortfolios :many
SELECT * FROM portfolios;

-- name: CreatePortfolio :one
INSERT INTO portfolios (user_id, target_portfolio)
VALUES ($1, $2)
RETURNING *;

-- name: DeletePortfolio :exec
DELETE FROM portfolios WHERE user_id = $1;

-- name: UpsertPortfolio :one
INSERT INTO portfolios (user_id, target_portfolio)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET target_portfolio = EXCLUDED.target_portfolio, updated_at = NOW()
RETURNING *;
