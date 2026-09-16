-- name: GetPortfolioByUserID :one
SELECT * FROM portfolios WHERE user_id = $1;
