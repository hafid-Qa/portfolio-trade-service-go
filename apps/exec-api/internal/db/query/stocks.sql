-- name: ListStocks :many
SELECT * FROM stocks;

-- name: GetStocksByTickers :many
SELECT * FROM stocks WHERE ticker = ANY($1::varchar[]);
    