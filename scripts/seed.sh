#!/bin/bash
# seed.go equivalent script for initial data setup

set -e

# Database connection parameters
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-multitrade}

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== MultiTrade Platform - Database Seeding ===${NC}\n"

# Connect to database and seed data
export PGPASSWORD=$DB_PASSWORD

# Insert sample assets
echo -e "${BLUE}Seeding assets...${NC}"
psql -h $DB_HOST -U $DB_USER -d $DB_NAME << EOF
INSERT INTO assets (symbol, name, type, current_price, previous_close, high, low, volume, created_at, updated_at) VALUES
('AAPL', 'Apple Inc.', 'STOCK', 150.25, 149.80, 151.50, 149.00, 50000000, NOW(), NOW()),
('GOOGL', 'Alphabet Inc.', 'STOCK', 140.65, 139.50, 141.80, 138.90, 30000000, NOW(), NOW()),
('BTC', 'Bitcoin', 'CRYPTO', 45000.00, 44500.00, 46000.00, 44000.00, 100000, NOW(), NOW()),
('ETH', 'Ethereum', 'CRYPTO', 2500.00, 2450.00, 2600.00, 2400.00, 500000, NOW(), NOW()),
('GOLD', 'Gold Futures', 'COMMODITY', 1950.00, 1945.00, 1960.00, 1940.00, 100000, NOW(), NOW()),
('EURUSD', 'EUR/USD', 'FX_PAIR', 1.0950, 1.0900, 1.1000, 1.0900, 5000000, NOW(), NOW())
ON CONFLICT DO NOTHING;
EOF

echo -e "${GREEN}✓ Assets seeded${NC}\n"

# Insert sample users (admin account)
echo -e "${BLUE}Seeding admin user...${NC}"
psql -h $DB_HOST -U $DB_USER -d $DB_NAME << EOF
-- Password: Admin@123456 (bcrypt hash)
INSERT INTO users (email, username, password, first_name, last_name, phone_number, role, status, kyc_status, created_at, updated_at) VALUES
('admin@multitrade.com', 'admin', '\$2a\$12\$ViMEX6x5pSJoV.0N/gGFyeqMX3fhdKCTFQfF.6YfmHrVGjC5H8P1O', 'Admin', 'User', '+1234567890', 'ADMIN', 'ACTIVE', 'APPROVED', NOW(), NOW())
ON CONFLICT DO NOTHING;
EOF

echo -e "${GREEN}✓ Admin user seeded${NC}\n"

echo -e "${GREEN}=== Database Seeding Completed Successfully ===${NC}\n"
echo "You can now login with:"
echo "  Email: admin@multitrade.com"
echo "  Password: Admin@123456"
