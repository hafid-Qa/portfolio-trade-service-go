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
