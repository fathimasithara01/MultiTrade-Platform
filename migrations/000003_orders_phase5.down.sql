-- Rollback Phase 5 order additions
DROP INDEX IF EXISTS idx_orders_asset_side_status;
DROP INDEX IF EXISTS idx_orders_idempotency_key;
ALTER TABLE orders DROP COLUMN IF EXISTS idempotency_key;
ALTER TABLE orders DROP COLUMN IF EXISTS remaining_quantity;
