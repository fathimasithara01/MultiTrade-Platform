-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Wallets Table
CREATE TABLE IF NOT EXISTS wallets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance NUMERIC(20, 8) DEFAULT 0.00000000 NOT NULL CHECK (balance >= 0),
    currency VARCHAR(10) DEFAULT 'USD' NOT NULL,
    version BIGINT DEFAULT 1 NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Wallet Transactions Table
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id BIGSERIAL PRIMARY KEY,
    wallet_id BIGINT NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
    amount NUMERIC(20, 8) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    reference_id VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Assets Table
CREATE TABLE IF NOT EXISTS assets (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price NUMERIC(20, 8) NOT NULL CHECK (price >= 0),
    quantity NUMERIC(20, 8) NOT NULL CHECK (quantity >= 0),
    status VARCHAR(50) DEFAULT 'ACTIVE' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Orders Table
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    side VARCHAR(10) NOT NULL, -- BUY, SELL
    type VARCHAR(20) NOT NULL, -- LIMIT
    price NUMERIC(20, 8) NOT NULL CHECK (price > 0),
    quantity NUMERIC(20, 8) NOT NULL CHECK (quantity > 0),
    filled_quantity NUMERIC(20, 8) DEFAULT 0.00000000 NOT NULL CHECK (filled_quantity <= quantity),
    status VARCHAR(50) NOT NULL, -- PENDING, PARTIALLY_FILLED, FILLED, CANCELLED
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Trades Table
CREATE TABLE IF NOT EXISTS trades (
    id BIGSERIAL PRIMARY KEY,
    buy_order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    sell_order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    price NUMERIC(20, 8) NOT NULL CHECK (price > 0),
    quantity NUMERIC(20, 8) NOT NULL CHECK (quantity > 0),
    buyer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    seller_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Portfolios Table
CREATE TABLE IF NOT EXISTS portfolios (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    quantity NUMERIC(20, 8) DEFAULT 0.00000000 NOT NULL CHECK (quantity >= 0),
    average_buy_price NUMERIC(20, 8) DEFAULT 0.00000000 NOT NULL CHECK (average_buy_price >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (user_id, asset_id)
);

-- Audit Logs Table
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    details TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create Indexes
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_asset_id ON orders(asset_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_trades_buyer_id ON trades(buyer_id);
CREATE INDEX IF NOT EXISTS idx_trades_seller_id ON trades(seller_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_user_id ON portfolios(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
