
CREATE TABLE IF NOT EXISTS stocks(
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL UNIQUE,
    price INTEGER NOT NULL,
    tradable BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stocks_ticker ON stocks(ticker);
CREATE INDEX IF NOT EXISTS idx_stocks_tradable ON stocks(tradable);
CREATE INDEX IF NOT EXISTS idx_stocks_created_at ON stocks(created_at);
