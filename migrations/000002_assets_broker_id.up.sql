-- Add broker_id to assets so each asset is owned by a specific broker user.
ALTER TABLE assets ADD COLUMN IF NOT EXISTS broker_id BIGINT REFERENCES users(id) ON DELETE RESTRICT;

-- Back-fill existing rows with NULL is acceptable; new rows will always supply broker_id.
CREATE INDEX IF NOT EXISTS idx_assets_broker_id ON assets(broker_id);
CREATE INDEX IF NOT EXISTS idx_assets_status    ON assets(status);
