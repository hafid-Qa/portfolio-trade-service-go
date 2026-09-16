CREATE TABLE IF NOT EXISTS portfolios(
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    target_portfolio JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_portfolios_user_id ON portfolios(user_id);
