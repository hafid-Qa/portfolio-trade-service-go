CREATE INDEX IF NOT EXISTS idx_portfolios_user_id ON portfolios(user_id);
ALTER TABLE portfolios DROP CONSTRAINT IF EXISTS uq_portfolios_user_id;
