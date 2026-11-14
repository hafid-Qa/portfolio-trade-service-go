-- One target portfolio per user: PUT /api/portfolios upserts on user_id, and
-- GetPortfolioByUserID (:one) assumes at most one row. The UNIQUE constraint
-- brings its own index, so the plain one from 000002 is redundant.
ALTER TABLE portfolios ADD CONSTRAINT uq_portfolios_user_id UNIQUE (user_id);
DROP INDEX IF EXISTS idx_portfolios_user_id;
