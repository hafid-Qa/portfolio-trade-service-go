# Portfolio Trade Service — Go

Given a trade amount, the service apportions it across the symbols in a customer's
target portfolio by weight, computes the resulting per-symbol orders, and persists
the trade. A Go implementation of [portfolio-trade-service](https://github.com/hafid-Qa/portfolio-trade-service)
(the original Python version), grown past a straight port into its own two-service,
gRPC-based architecture with Postgres persistence.

## Architecture

Two services, one shared gRPC contract, plus an async worker for trade execution:

- **`calc`** — pure calculation, owns no data. Takes a trade amount, target portfolio
  weights, and stock prices/tradability as request parameters, and returns the
  apportioned orders. No database, no YAML, no business-rule constants baked in —
  everything it needs comes from the caller.
- **`exec-api`** — owns the stock catalogue, portfolios, and trade history. Calls
  `calc` over gRPC to do the actual math, persists the resulting trade, then enqueues
  it for execution on the `broker` queue.
- **`proto/`** — the shared `calc.v1` gRPC contract, its own Go module. Both services
  import the same generated client/server code, so there's no risk of two
  independently-generated stubs drifting apart.
- **worker (`cmd/worker`)** — a second entrypoint in the `exec-api` module (same
  `Trade`/`TradeStatus` domain, not a separate service deployment) that consumes
  `trade:execute` tasks from the `broker` queue and drives each trade's status
  from `pending` → `executing` → `completed`/`failed`.

## Stack

| Concern | Choice |
| --- | --- |
| HTTP (`exec-api`) | `gin-gonic/gin` v1.12 |
| RPC (`calc` ↔ `exec-api`) | `google.golang.org/grpc` + Protocol Buffers |
| Request validation | `go-playground/validator` (via Gin binding tags) |
| Config | `sethvargo/go-envconfig` — env vars into a typed struct |
| Persistence | PostgreSQL, via `sqlc`-generated queries + `lib/pq` |
| Migrations | `golang-migrate`, run via its CLI (not a Go library dependency in prod code) |
| Async task queue | `hibiken/asynq`, backed by Redis (`broker` service) |
| YAML | `goccy/go-yaml` — `memory` backend fallback only (see below) |
| API docs | `swaggo/swag` (generated), served via `swaggo/gin-swagger` |
| Reload during development | `air`, running inside the dev container |

## Running it

```bash
docker compose up
```

The API serves on `http://localhost:${API_EXT_PORT}` (`8000` by default, see `.env`).
`calc` isn't meant to be called directly by a client — `exec-api` is the only consumer.

```bash
curl -s -X POST localhost:8000/api/users/1/trade \
  -H 'Content-Type: application/json' \
  -d '{"amount":10000}'
```

```json
{
  "id": 1,
  "amount": 10000,
  "target_portfolio": { "A": 40, "B": 60 },
  "orders": [
    { "symbol": "A", "amount": 4000, "quantity": 4 },
    { "symbol": "B", "amount": 6000, "quantity": 38.709 }
  ],
  "status": "pending",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

Interactive docs at `http://localhost:8000/swagger/index.html`, regenerated
automatically on every `air` rebuild.

`docker compose up` also starts `broker` (Redis) and `consumer` (the worker,
`go run ./cmd/worker`). Every created trade is enqueued onto `broker` right after
it's persisted; `consumer` picks it up and advances its status from `pending` to
`executing` to `completed`/`failed` a second later — poll
`GET /api/users/{user_id}/trades/{id}` to watch it transition.

### Database setup

Migrations run automatically on every `api` container start (`scripts/entrypoint.sh`
runs `migrate ... up` before the app starts, skipped when `DATA_SOURCE=memory`) — no
manual step needed there, and it's idempotent (reports `no change` if nothing's
pending).

Stocks are seeded automatically too, right after migrations (`cmd/seed`, also skipped
in `memory` mode): it reads `data/stocks.yml` and upserts every entry into `stocks` in
a single batched query, keyed on `ticker` — safe to run on every container start,
since re-seeding an already-seeded DB just reaffirms the same rows rather than erroring
or duplicating them.

Portfolios are still manual, though: nothing seeds `data/portfolio.yml` yet. Insert via
`psql` or `sqlc`'s generated `CreatePortfolio` — see `data/portfolio.yml` for the
values.

### `memory` fallback

`exec-api`'s stock/portfolio backend is chosen by `DATA_SOURCE` (`postgres` by
default). Setting `DATA_SOURCE=memory` switches to the original YAML-backed
in-memory repositories (`data/stocks.yml`, `data/portfolio.yml`) — useful for local
dev without a live Postgres connection, kept as a reference/fallback rather than a
second production path. Trade persistence in that mode uses an in-memory store too
(data lost on restart, not written to Postgres), and `TradeProducer` is a no-op
in that mode (logs and returns, never touches `broker`) — no reason for a local
`memory` run or a test to reach out to Redis.

## Business rules

- All dollar calculations floor — integer division throughout, never float division on
  the way to a dollar amount. Quantities are stored as an integer count of
  0.001-share units; the division into a decimal `quantity` happens only once, at the
  API response boundary.
- Minimum trade amount: $1,000.
- The trade amount is apportioned across the target portfolio's weights: a $10,000
  trade against `A: 40%, B: 60%` produces `A: $4,000`, `B: $6,000`.
- A halted (non-tradable) or unknown stock is excluded, and its weight is
  redistributed across the remaining eligible symbols.
- Minimum order amount: $200. A symbol whose apportioned share falls below that is
  excluded and its weight redistributed the same way as a halted stock.
- A symbol too expensive to buy even 0.001 of a share at its apportioned amount is
  excluded and its weight redistributed, same as the two rules above.
- Exclusion and redistribution run until the eligible set stabilizes — one round isn't
  always enough, since a symbol excluded on affordability can itself unlock room for
  a previously-marginal survivor.
- Flooring can leave a small remainder unallocated (e.g. a $10,000 trade producing
  orders totaling $9,999) — this is expected, not a bug.
- A portfolio referencing a ticker absent from the stock catalogue is a data-integrity
  problem, not a client error — caught at `exec-api` startup, not request time.
- Every trade is persisted (`id`, `status`, `created_at`, `updated_at` returned in the
  response) and enqueued for execution only after that persist succeeds — nothing is
  executed against a real market, so the worker simulates it: `status` moves
  `pending` → `executing` → `completed`/`failed` (a trade ID divisible by 10 fails,
  deterministic so it's easy to test; everything else completes).

## Folder structure

```
proto/                            shared gRPC contract, its own Go module
  calc/v1/calc.proto               source contract
  gen/                             generated calcv1 package (committed)
apps/
  calc/                            module calc — pure calculation, zero owned data
    cmd/main.go                    gRPC server
    config/config.go               GRPCPort (env CALC_GRPC_PORT)
    internal/domain/
      calculator.go                 Calculate() — the allocation rules, pure, no I/O
      stock.go, order.go            plain value types, no repositories
  exec-api/                        module app — owns data, orchestrates
    cmd/
      server/main.go                HTTP API entrypoint
      worker/main.go                asynq worker entrypoint — consumes the broker queue
      seed/main.go                  batch-upserts data/stocks.yml into Postgres, run by
                                     entrypoint.sh right after migrations
    config/config.go               ServerPort, CalcAddr, DataSource, DB_*, Broker* config
    internal/
      domain/
        const.go                    MinTradeAmount, MinOrderAmount, QuantityPrecision
        errors.go                   sentinel errors + typed errors
        stock.go, portfolio.go      Symbol, Stock, Portfolio (validating constructors)
        order.go                    Order — int-only quantityUnits
        trade_result.go             TradeResult — the persisted trade (id, status,
                                     amount, target portfolio, orders, timestamps)
        trade_status.go             TradeStatus enum + Valid()
        producer.go                 TradeProducer (outbound port: Enqueue(ctx, tradeID))
        repository.go               StockRepository, PortfolioRepository,
                                     TradeRepository (interfaces; TradeRepository also
                                     has UpdateStatus, used by the worker)
        service.go                  TradeService.CreateTrade — orchestration:
                                     resolve portfolio/stocks -> call calc -> persist
                                     -> enqueue (only after persist succeeds)
      queue/
        trade_producer.go           TradeProducer impl — asynq.Client-backed, defines
                                     the trade:execute task type/payload
      worker/
        trade_hanlder.go            TradeHandler.ProcessTask — asynq.Handler consuming
                                     trade:execute: pending -> executing -> completed/
                                     failed (trade ID divisible by 10 fails, deterministic).
                                     A not-found trade or malformed payload is wrapped
                                     with asynq.SkipRetry -- terminal, not worth retrying.
      repositories/
        memory/                     YAML-backed stock/portfolio repos, in-memory
                                     trade store, no-op TradeProducer —
                                     DATA_SOURCE=memory fallback
        postgres/                   sqlc-backed stock/portfolio/trade repos (default)
      sqlc/                         sqlc config target: migrations/, query/, and the
                                     generated Go code, all together
      api/
        router.go, server.go        Gin engine, Server/NewServer, startup
                                     referential-integrity check
        trade.go                    TradeURI, TradeRequest, TradeResponse, OrderResponse
        errors.go                   errorResponse helper
    docs/                          generated by swag — gitignored, rebuilt on every start
data/
  stocks.yml, portfolio.yml        reference data (memory backend + Postgres seed source)
compose.yml, compose.test.yml       dev stack + test-only db_test override
Makefile                            migrate/sqlc/proto codegen, make test
.github/workflows/ci.yml
```

## Testing

```bash
make test
```

Runs `go vet`/`go test` for both `exec-api` and `calc` inside Docker. Self-sufficient —
`compose.test.yml` brings up a health-checked `db_test` service, and the Postgres-backed
repository tests' `TestMain` runs migrations against it itself before each run, so a
completely fresh checkout needs no manual setup. `internal/queue`'s tests connect to
the same live `broker` service the app itself uses (already a dependency of `api`, so
it's started automatically) rather than a mock — asynq has no official in-memory
broker — and clean up whatever they enqueue. `internal/worker`'s tests need no live
infrastructure at all: they drive `TradeHandler.ProcessTask` directly against
`memory.TradeRepo`.

Expected values in the calculator/service tests are hand-computed, not derived from
the implementation — the point is catching a shared misunderstanding, which computing
them from the same code under test wouldn't do.

CI (`.github/workflows/ci.yml`) runs `make test`, a `gofmt -l` check across both
services, and a build check for the `proto` module.

## Makefile commands

Everything assumes `docker compose up -d` is already running.

| Command | Does |
| --- | --- |
| `make test` | `go vet` + `go test` for `exec-api` and `calc`, brings up `db_test` itself |
| `make migrateup` | Apply all pending migrations to `db` (also runs automatically on `api` startup) |
| `make migrateup1` | Apply just the next pending migration |
| `make migratedown` | Roll back all migrations on `db` |
| `make migratedown1` | Roll back the last applied migration only |
| `make new_migration name=add_x` | Scaffold a new `.up.sql`/`.down.sql` pair in `internal/sqlc/migrations/` |
| `make sqlc` | Regenerate Go code from `internal/sqlc/query/*.sql` (see `sqlc.yaml`) |
| `make buf` | Regenerate `proto/gen` from `proto/calc/v1/calc.proto` |

`db_docs`/`db_schema` exist in the `Makefile` but aren't wired up yet (`doc/db.dbml`
doesn't exist).
