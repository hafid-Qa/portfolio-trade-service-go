-- name: ListStocks :many
SELECT * FROM stocks;

-- name: GetStocksByTickers :many
SELECT * FROM stocks WHERE ticker = ANY($1::varchar[]);

-- name: CreateStock :one
INSERT INTO stocks (ticker, price, tradable)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpsertStocks :many
INSERT INTO stocks (ticker, price, tradable)
SELECT unnest(sqlc.arg(tickers)::varchar[]),
       unnest(sqlc.arg(prices)::int[]),
       unnest(sqlc.arg(tradables)::bool[])
ON CONFLICT (ticker) DO UPDATE SET price = EXCLUDED.price, tradable = EXCLUDED.tradable, updated_at = NOW()
RETURNING *;

-- name: DeleteStock :exec
DELETE FROM stocks WHERE ticker = $1;
    