package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/fathimasithara01/multitrade-platform/internal/wallet"
)

var ErrWalletNotFound = errors.New("wallet not found")
var ErrInsufficientBalance = errors.New("insufficient wallet balance")

type WalletRepository interface {
	CreateWallet(ctx context.Context, tx *sqlx.Tx, userID int64) (*wallet.Wallet, error)
	GetByUserID(ctx context.Context, userID int64) (*wallet.Wallet, error)
	GetByUserIDForUpdate(ctx context.Context, tx *sqlx.Tx, userID int64) (*wallet.Wallet, error)
	UpdateBalance(ctx context.Context, tx *sqlx.Tx, walletID int64, newBalance string, currentVersion int64) error
	InsertTransaction(ctx context.Context, tx *sqlx.Tx, wt *wallet.WalletTransaction) (*wallet.WalletTransaction, error)
	ListTransactions(ctx context.Context, walletID int64, limit, offset int) ([]wallet.WalletTransaction, error)
	CountTransactions(ctx context.Context, walletID int64) (int, error)
}

type walletRepository struct {
	db *sqlx.DB
}

func NewWalletRepository(db *sqlx.DB) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) CreateWallet(ctx context.Context, tx *sqlx.Tx, userID int64) (*wallet.Wallet, error) {
	query := `
		INSERT INTO wallets (user_id, balance, currency, version, created_at, updated_at)
		VALUES ($1, 0.00000000, 'USD', 1, NOW(), NOW())
		RETURNING id, user_id, balance, currency, version, created_at, updated_at`

	w := &wallet.Wallet{}
	var err error
	if tx != nil {
		err = tx.QueryRowxContext(ctx, query, userID).StructScan(w)
	} else {
		err = r.db.QueryRowxContext(ctx, query, userID).StructScan(w)
	}
	if err != nil {
		return nil, fmt.Errorf("create wallet: %w", err)
	}
	return w, nil
}

func (r *walletRepository) GetByUserID(ctx context.Context, userID int64) (*wallet.Wallet, error) {
	query := `
		SELECT id, user_id, balance, currency, version, created_at, updated_at
		FROM wallets WHERE user_id = $1`

	w := &wallet.Wallet{}
	err := r.db.QueryRowxContext(ctx, query, userID).StructScan(w)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWalletNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	return w, nil
}

func (r *walletRepository) GetByUserIDForUpdate(ctx context.Context, tx *sqlx.Tx, userID int64) (*wallet.Wallet, error) {
	query := `
		SELECT id, user_id, balance, currency, version, created_at, updated_at
		FROM wallets WHERE user_id = $1 FOR UPDATE`

	w := &wallet.Wallet{}
	err := tx.QueryRowxContext(ctx, query, userID).StructScan(w)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWalletNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get wallet for update: %w", err)
	}
	return w, nil
}

func (r *walletRepository) UpdateBalance(ctx context.Context, tx *sqlx.Tx, walletID int64, newBalance string, currentVersion int64) error {
	query := `
		UPDATE wallets
		SET balance = $1, version = version + 1, updated_at = NOW()
		WHERE id = $2 AND version = $3`

	res, err := tx.ExecContext(ctx, query, newBalance, walletID, currentVersion)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update balance rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update balance: no rows affected, possible version conflict")
	}
	return nil
}

func (r *walletRepository) InsertTransaction(ctx context.Context, tx *sqlx.Tx, wt *wallet.WalletTransaction) (*wallet.WalletTransaction, error) {
	query := `
		INSERT INTO wallet_transactions
			(wallet_id, amount, transaction_type, status, reference_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, wallet_id, amount, transaction_type, status, reference_id, description, created_at`

	created := &wallet.WalletTransaction{}
	err := tx.QueryRowxContext(ctx, query,
		wt.WalletID,
		wt.Amount,
		wt.TransactionType,
		wt.Status,
		wt.ReferenceID,
		wt.Description,
	).StructScan(created)
	if err != nil {
		return nil, fmt.Errorf("insert wallet transaction: %w", err)
	}
	return created, nil
}

func (r *walletRepository) ListTransactions(ctx context.Context, walletID int64, limit, offset int) ([]wallet.WalletTransaction, error) {
	query := `
		SELECT id, wallet_id, amount, transaction_type, status, reference_id, description, created_at
		FROM wallet_transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	var txs []wallet.WalletTransaction
	if err := r.db.SelectContext(ctx, &txs, query, walletID, limit, offset); err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return txs, nil
}

func (r *walletRepository) CountTransactions(ctx context.Context, walletID int64) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx, `SELECT COUNT(*) FROM wallet_transactions WHERE wallet_id = $1`, walletID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count transactions: %w", err)
	}
	return count, nil
}
