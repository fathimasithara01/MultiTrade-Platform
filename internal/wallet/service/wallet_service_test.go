package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fathimasithara01/multitrade-platform/internal/config"
	"github.com/fathimasithara01/multitrade-platform/internal/database"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet/dto"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet/service"
)

func connectTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	viper.SetConfigFile("../../.env")
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Skipf("config load failed: %v", err)
	}
	db, err := database.ConnectDB(context.Background(), cfg)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	if err := database.RunMigrations(db, "../../migrations"); err != nil {
		db.Close()
		t.Skipf("migrations failed: %v", err)
	}
	return db
}

func seedWallet(t *testing.T, db *sqlx.DB, email, balance string) int64 {
	t.Helper()
	ctx := context.Background()

	var userID int64
	err := db.QueryRowxContext(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if err != nil {
		err = db.QueryRowxContext(ctx, `
			INSERT INTO users (email, password_hash, role, status, created_at, updated_at)
			VALUES ($1, 'x', 'trader', 'ACTIVE', NOW(), NOW())
			RETURNING id`, email).Scan(&userID)
		require.NoError(t, err, "create test user")
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO wallets (user_id, balance, currency, version, created_at, updated_at)
		VALUES ($1, $2, 'USD', 1, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET balance = $2, version = 1, updated_at = NOW()`,
		userID, balance)
	require.NoError(t, err, "seed wallet")
	return userID
}

func TestDeposit_IncreasesBalance(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	userID := seedWallet(t, db, "wdep@unit.com", "0")
	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)

	desc := "test deposit"
	w, tx, err := svc.Deposit(context.Background(), userID, dto.AmountInput{
		Amount:      "250.50",
		Description: &desc,
	})

	require.NoError(t, err)
	assert.Equal(t, "250.50000000", w.Balance)
	assert.Equal(t, wallet.TxTypeDeposit, tx.TransactionType)
	assert.Equal(t, wallet.TxStatusCompleted, tx.Status)
	assert.Equal(t, "250.50000000", tx.Amount)
}

func TestDeposit_MultipleDepositsAccumulate(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	userID := seedWallet(t, db, "wmulti@unit.com", "0")
	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)

	ctx := context.Background()
	svc.Deposit(ctx, userID, dto.AmountInput{Amount: "100.00"})
	svc.Deposit(ctx, userID, dto.AmountInput{Amount: "200.00"})
	w, _, err := svc.Deposit(ctx, userID, dto.AmountInput{Amount: "50.00"})

	require.NoError(t, err)
	assert.Equal(t, "350.00000000", w.Balance)
}

func TestWithdraw_DecreaseBalance(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	userID := seedWallet(t, db, "wwd@unit.com", "500.00")
	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)

	w, tx, err := svc.Withdraw(context.Background(), userID, dto.AmountInput{Amount: "200.00"})

	require.NoError(t, err)
	assert.Equal(t, "300.00000000", w.Balance)
	assert.Equal(t, wallet.TxTypeWithdrawal, tx.TransactionType)
	assert.Equal(t, wallet.TxStatusCompleted, tx.Status)
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	userID := seedWallet(t, db, "winsuf@unit.com", "10.00")
	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)

	_, _, err := svc.Withdraw(context.Background(), userID, dto.AmountInput{Amount: "9999.00"})
	assert.True(t, errors.Is(err, repository.ErrInsufficientBalance))
}

func TestWithdraw_ExactBalance(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	userID := seedWallet(t, db, "wexact@unit.com", "100.00")
	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)

	w, _, err := svc.Withdraw(context.Background(), userID, dto.AmountInput{Amount: "100.00"})
	require.NoError(t, err)
	assert.Equal(t, "0.00000000", w.Balance)
}

func TestWithdraw_ZeroAmount(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	userID := seedWallet(t, db, "wzero@unit.com", "100.00")
	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)

	_, _, err := svc.Withdraw(context.Background(), userID, dto.AmountInput{Amount: "0"})
	assert.True(t, errors.Is(err, service.ErrInvalidAmount))
}

func TestWithdraw_NegativeAmount(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	userID := seedWallet(t, db, "wneg@unit.com", "100.00")
	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)

	_, _, err := svc.Withdraw(context.Background(), userID, dto.AmountInput{Amount: "-50"})
	assert.True(t, errors.Is(err, service.ErrInvalidAmount))
}

func TestListTransactions_Paginated(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	userID := seedWallet(t, db, "wtxlist@unit.com", "0")
	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.Deposit(ctx, userID, dto.AmountInput{Amount: "10.00"})
	}

	page, err := svc.ListTransactions(ctx, userID, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, 3, len(page.Data))
	assert.GreaterOrEqual(t, page.Total, 5)
	assert.GreaterOrEqual(t, page.TotalPages, 2)
}

func TestGetWallet_NotFound(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()

	svc := service.NewWalletService(db, repository.NewWalletRepository(db), nil)

	_, err := svc.GetWallet(context.Background(), 9999999)
	assert.True(t, errors.Is(err, repository.ErrWalletNotFound))
}
