-- Add status column to users for suspend/activate functionality.
ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE';
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_role   ON users(role);

-- Index for analytics volume queries.
CREATE INDEX IF NOT EXISTS idx_trades_created_at ON trades(created_at);

-- Index to speed up suspicious-user queries.
CREATE INDEX IF NOT EXISTS idx_trades_buyer_created  ON trades(buyer_id, created_at);
CREATE INDEX IF NOT EXISTS idx_trades_seller_created ON trades(seller_id, created_at);
