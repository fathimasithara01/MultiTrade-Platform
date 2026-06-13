-- Add remaining_quantity to track how much of an order is still unfilled.
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS remaining_quantity NUMERIC(20, 8)
        GENERATED ALWAYS AS (quantity - filled_quantity) STORED;

-- Add idempotency_key for duplicate-request protection on order placement.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_idempotency_key
    ON orders(user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Index to speed up order-book queries (pending/partial orders for an asset).
CREATE INDEX IF NOT EXISTS idx_orders_asset_side_status
    ON orders(asset_id, side, status)
    WHERE status IN ('PENDING', 'PARTIALLY_FILLED');
