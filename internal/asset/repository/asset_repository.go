package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/fathimasithara01/multitrade-platform/internal/asset"
)

var ErrAssetNotFound = errors.New("asset not found")

type AssetRepository interface {
	Create(ctx context.Context, a *asset.Asset) (*asset.Asset, error)
	GetByID(ctx context.Context, id int64) (*asset.Asset, error)
	Update(ctx context.Context, a *asset.Asset) (*asset.Asset, error)
	ListActive(ctx context.Context) ([]asset.Asset, error)
}

type assetRepository struct {
	db *sqlx.DB
}

func NewAssetRepository(db *sqlx.DB) AssetRepository {
	return &assetRepository{db: db}
}

func (r *assetRepository) Create(ctx context.Context, a *asset.Asset) (*asset.Asset, error) {
	query := `
		INSERT INTO assets (broker_id, symbol, name, description, price, quantity, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, broker_id, symbol, name, description, price, quantity, status, created_at, updated_at`

	created := &asset.Asset{}
	err := r.db.QueryRowxContext(ctx, query,
		a.BrokerID,
		a.Symbol,
		a.Name,
		a.Description,
		a.Price,
		a.Quantity,
		a.Status,
	).StructScan(created)
	if err != nil {
		return nil, fmt.Errorf("asset create: %w", err)
	}
	return created, nil
}

func (r *assetRepository) GetByID(ctx context.Context, id int64) (*asset.Asset, error) {
	query := `
		SELECT id, broker_id, symbol, name, description, price, quantity, status, created_at, updated_at
		FROM assets WHERE id = $1`

	a := &asset.Asset{}
	err := r.db.QueryRowxContext(ctx, query, id).StructScan(a)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAssetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("asset get by id: %w", err)
	}
	return a, nil
}

func (r *assetRepository) Update(ctx context.Context, a *asset.Asset) (*asset.Asset, error) {
	query := `
		UPDATE assets
		SET name = $1, description = $2, price = $3, quantity = $4, status = $5, updated_at = NOW()
		WHERE id = $6 AND broker_id = $7
		RETURNING id, broker_id, symbol, name, description, price, quantity, status, created_at, updated_at`

	updated := &asset.Asset{}
	err := r.db.QueryRowxContext(ctx, query,
		a.Name,
		a.Description,
		a.Price,
		a.Quantity,
		a.Status,
		a.ID,
		a.BrokerID,
	).StructScan(updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAssetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("asset update: %w", err)
	}
	return updated, nil
}

func (r *assetRepository) ListActive(ctx context.Context) ([]asset.Asset, error) {
	query := `
		SELECT id, broker_id, symbol, name, description, price, quantity, status, created_at, updated_at
		FROM assets
		WHERE status = 'ACTIVE'
		ORDER BY symbol ASC`

	var assets []asset.Asset
	if err := r.db.SelectContext(ctx, &assets, query); err != nil {
		return nil, fmt.Errorf("asset list active: %w", err)
	}
	return assets, nil
}
