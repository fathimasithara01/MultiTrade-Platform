package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/fathimasithara01/multitrade-platform/internal/trade"
)

type TradeRepository interface {
	Insert(ctx context.Context, tx *sqlx.Tx, t *trade.Trade) (*trade.Trade, error)
	GetByUserID(ctx context.Context, userID int64) ([]trade.Trade, error)
}

type tradeRepository struct {
	db *sqlx.DB
}

func NewTradeRepository(db *sqlx.DB) TradeRepository {
	return &tradeRepository{db: db}
}

func (r *tradeRepository) Insert(ctx context.Context, tx *sqlx.Tx, t *trade.Trade) (*trade.Trade, error) {
	query := `
		INSERT INTO trades
			(buy_order_id, sell_order_id, asset_id, price, quantity, buyer_id, seller_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, buy_order_id, sell_order_id, asset_id, price, quantity, buyer_id, seller_id, created_at`

	created := &trade.Trade{}
	err := tx.QueryRowxContext(ctx, query,
		t.BuyOrderID, t.SellOrderID, t.AssetID,
		t.Price, t.Quantity,
		t.BuyerID, t.SellerID,
	).StructScan(created)
	if err != nil {
		return nil, fmt.Errorf("insert trade: %w", err)
	}
	return created, nil
}

func (r *tradeRepository) GetByUserID(ctx context.Context, userID int64) ([]trade.Trade, error) {
	query := `
		SELECT id, buy_order_id, sell_order_id, asset_id, price, quantity, buyer_id, seller_id, created_at
		FROM trades
		WHERE buyer_id = $1 OR seller_id = $1
		ORDER BY created_at DESC`

	var trades []trade.Trade
	err := r.db.SelectContext(ctx, &trades, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get trades by user id: %w", err)
	}
	return trades, nil
}
