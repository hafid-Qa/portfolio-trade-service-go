ALTER TABLE portfolios DROP CONSTRAINT IF EXISTS fk_portfolios_user_id;
ALTER TABLE trades DROP CONSTRAINT IF EXISTS fk_trades_user_id;
