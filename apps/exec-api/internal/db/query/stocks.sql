-- name: ListStocks :many
SELECT * FROM stocks;

-- name: GetStocksByTickers :many
SELECT * FROM stocks WHERE ticker = ANY($1::varchar[]);

-- name: CreateStock :one
INSERT INTO stocks (ticker, price, tradable)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteStock :exec
DELETE FROM stocks WHERE ticker = $1;
    