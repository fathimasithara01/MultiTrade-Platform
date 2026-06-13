-- Rollback Phase 10 admin additions
DROP INDEX IF EXISTS idx_trades_seller_created;
DROP INDEX IF EXISTS idx_trades_buyer_created;
DROP INDEX IF EXISTS idx_trades_created_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_status;
ALTER TABLE users DROP COLUMN IF EXISTS status;
