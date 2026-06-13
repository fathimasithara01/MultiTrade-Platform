package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/fathimasithara01/multitrade-platform/internal/portfolio"
)

var ErrPortfolioNotFound = errors.New("portfolio holding not found")

type PortfolioRepository interface {
	GetHolding(ctx context.Context, userID, assetID int64) (*portfolio.PortfolioHolding, error)
	UpsertBuyerHolding(ctx context.Context, tx *sqlx.Tx, userID, assetID int64, qty, tradePrice string) error
	ReduceSellerHolding(ctx context.Context, tx *sqlx.Tx, userID, assetID int64, qty string) error
	ListHoldings(ctx context.Context, userID int64) ([]portfolio.PortfolioHolding, error)
}

type portfolioRepository struct {
	db *sqlx.DB
}

func NewPortfolioRepository(db *sqlx.DB) PortfolioRepository {
	return &portfolioRepository{db: db}
}

func (r *portfolioRepository) GetHolding(ctx context.Context, userID, assetID int64) (*portfolio.PortfolioHolding, error) {
	query := `SELECT user_id, asset_id, quantity, average_buy_price
	          FROM portfolios WHERE user_id = $1 AND asset_id = $2`
	h := &portfolio.PortfolioHolding{}
	err := r.db.QueryRowxContext(ctx, query, userID, assetID).StructScan(h)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPortfolioNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get portfolio holding: %w", err)
	}
	return h, nil
}

func (r *portfolioRepository) UpsertBuyerHolding(ctx context.Context, tx *sqlx.Tx, userID, assetID int64, qty, tradePrice string) error {
	query := `
		INSERT INTO portfolios (user_id, asset_id, quantity, average_buy_price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id, asset_id) DO UPDATE
		  SET quantity         = portfolios.quantity + EXCLUDED.quantity,
		      average_buy_price = (
		          portfolios.quantity        * portfolios.average_buy_price +
		          EXCLUDED.quantity          * EXCLUDED.average_buy_price
		      ) / (portfolios.quantity + EXCLUDED.quantity),
		      updated_at       = NOW()`

	_, err := tx.ExecContext(ctx, query, userID, assetID, qty, tradePrice)
	if err != nil {
		return fmt.Errorf("upsert buyer holding: %w", err)
	}
	return nil
}

func (r *portfolioRepository) ReduceSellerHolding(ctx context.Context, tx *sqlx.Tx, userID, assetID int64, qty string) error {
	query := `
		UPDATE portfolios
		SET quantity   = quantity - $1,
		    updated_at = NOW()
		WHERE user_id = $2 AND asset_id = $3`

	_, err := tx.ExecContext(ctx, query, qty, userID, assetID)
	if err != nil {
		return fmt.Errorf("reduce seller holding: %w", err)
	}
	return nil
}

func (r *portfolioRepository) ListHoldings(ctx context.Context, userID int64) ([]portfolio.PortfolioHolding, error) {
	query := `SELECT user_id, asset_id, quantity, average_buy_price
	          FROM portfolios WHERE user_id = $1`
	var holdings []portfolio.PortfolioHolding
	err := r.db.SelectContext(ctx, &holdings, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list portfolio holdings: %w", err)
	}
	return holdings, nil
}
