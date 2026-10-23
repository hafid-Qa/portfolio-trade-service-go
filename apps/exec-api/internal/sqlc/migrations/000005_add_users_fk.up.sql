ALTER TABLE portfolios ADD CONSTRAINT fk_portfolios_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE trades ADD CONSTRAINT fk_trades_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;
