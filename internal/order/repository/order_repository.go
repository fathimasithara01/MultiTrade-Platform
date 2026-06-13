package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/fathimasithara01/multitrade-platform/internal/order"
)

var ErrOrderNotFound = errors.New("order not found")
var ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key: order already exists")

type OrderRepository interface {
	Create(ctx context.Context, tx *sqlx.Tx, o *order.Order) (*order.Order, error)
	GetByID(ctx context.Context, id int64) (*order.Order, error)
	GetByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id int64) (*order.Order, error)
	UpdateStatus(ctx context.Context, tx *sqlx.Tx, id int64, status string) error
	UpdateFill(ctx context.Context, tx *sqlx.Tx, id int64, addFilled string, newStatus string) error
	GetOpenOrdersForMatching(ctx context.Context, tx *sqlx.Tx, assetID int64, side string) ([]order.Order, error)
	GetByIdempotencyKey(ctx context.Context, userID int64, key string) (*order.Order, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]order.Order, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
}

type orderRepository struct {
	db *sqlx.DB
}

func NewOrderRepository(db *sqlx.DB) OrderRepository {
	return &orderRepository{db: db}
}

const orderSelectCols = `
	id, user_id, asset_id, side, type, price, quantity,
	filled_quantity, remaining_quantity, status, idempotency_key,
	created_at, updated_at`

func (r *orderRepository) Create(ctx context.Context, tx *sqlx.Tx, o *order.Order) (*order.Order, error) {
	query := `
		INSERT INTO orders
			(user_id, asset_id, side, type, price, quantity, filled_quantity, status, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $7, $8, NOW(), NOW())
		RETURNING ` + orderSelectCols

	created := &order.Order{}
	var err error
	if tx != nil {
		err = tx.QueryRowxContext(ctx, query,
			o.UserID, o.AssetID, o.Side, o.Type,
			o.Price, o.Quantity, o.Status, o.IdempotencyKey,
		).StructScan(created)
	} else {
		err = r.db.QueryRowxContext(ctx, query,
			o.UserID, o.AssetID, o.Side, o.Type,
			o.Price, o.Quantity, o.Status, o.IdempotencyKey,
		).StructScan(created)
	}
	if err != nil {
		if isDuplicateKeyErr(err) {
			return nil, ErrDuplicateIdempotencyKey
		}
		return nil, fmt.Errorf("order create: %w", err)
	}
	return created, nil
}

func (r *orderRepository) GetByID(ctx context.Context, id int64) (*order.Order, error) {
	query := `SELECT ` + orderSelectCols + ` FROM orders WHERE id = $1`
	o := &order.Order{}
	err := r.db.QueryRowxContext(ctx, query, id).StructScan(o)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order get by id: %w", err)
	}
	return o, nil
}

func (r *orderRepository) GetByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id int64) (*order.Order, error) {
	query := `SELECT ` + orderSelectCols + ` FROM orders WHERE id = $1 FOR UPDATE`
	o := &order.Order{}
	err := tx.QueryRowxContext(ctx, query, id).StructScan(o)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order get for update: %w", err)
	}
	return o, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, tx *sqlx.Tx, id int64, status string) error {
	query := `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("order update status: %w", err)
	}
	return nil
}

func (r *orderRepository) UpdateFill(ctx context.Context, tx *sqlx.Tx, id int64, addFilled string, newStatus string) error {
	query := `
		UPDATE orders
		SET filled_quantity = filled_quantity + $1,
		    status          = $2,
		    updated_at      = NOW()
		WHERE id = $3`
	_, err := tx.ExecContext(ctx, query, addFilled, newStatus, id)
	if err != nil {
		return fmt.Errorf("order update fill: %w", err)
	}
	return nil
}

func (r *orderRepository) GetOpenOrdersForMatching(ctx context.Context, tx *sqlx.Tx, assetID int64, side string) ([]order.Order, error) {
	priceOrder := "ASC"
	if side == order.OrderSideBuy {
		priceOrder = "DESC"
	}

	query := `
		SELECT ` + orderSelectCols + `
		FROM orders
		WHERE asset_id = $1
		  AND side     = $2
		  AND status   IN ('PENDING', 'PARTIALLY_FILLED')
		ORDER BY price ` + priceOrder + `, created_at ASC`

	var orders []order.Order
	var err error
	if tx != nil {
		err = tx.SelectContext(ctx, &orders, query, assetID, side)
	} else {
		err = r.db.SelectContext(ctx, &orders, query, assetID, side)
	}
	if err != nil {
		return nil, fmt.Errorf("get open orders for matching: %w", err)
	}
	return orders, nil
}

func (r *orderRepository) GetByIdempotencyKey(ctx context.Context, userID int64, key string) (*order.Order, error) {
	query := `SELECT ` + orderSelectCols + `
		FROM orders WHERE user_id = $1 AND idempotency_key = $2`
	o := &order.Order{}
	err := r.db.QueryRowxContext(ctx, query, userID, key).StructScan(o)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order get by idempotency key: %w", err)
	}
	return o, nil
}

func (r *orderRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]order.Order, error) {
	query := `SELECT ` + orderSelectCols + `
		FROM orders WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	var orders []order.Order
	if err := r.db.SelectContext(ctx, &orders, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("order list by user: %w", err)
	}
	return orders, nil
}

func (r *orderRepository) CountByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx, `SELECT COUNT(*) FROM orders WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("order count: %w", err)
	}
	return count, nil
}

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsStr(msg, "23505") || containsStr(msg, "idx_orders_idempotency_key") || containsStr(msg, "unique constraint")
}

func containsStr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
