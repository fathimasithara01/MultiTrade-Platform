package matchingengine_test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"

	"github.com/fathimasithara01/multitrade-platform/internal/config"
	"github.com/fathimasithara01/multitrade-platform/internal/database"
	"github.com/fathimasithara01/multitrade-platform/internal/matchingengine"
	"github.com/fathimasithara01/multitrade-platform/internal/order"
	orderrepo "github.com/fathimasithara01/multitrade-platform/internal/order/repository"
	portfoliorepo "github.com/fathimasithara01/multitrade-platform/internal/portfolio/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/redis"
	traderepo "github.com/fathimasithara01/multitrade-platform/internal/trade/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/user"
	walletrepo "github.com/fathimasithara01/multitrade-platform/internal/wallet/repository"
)

func setupTestDB(t *testing.T) (*sqlx.DB, *redis.Client) {
	t.Helper()

	viper.SetConfigFile("../../.env")
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	db, err := database.ConnectDB(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test Postgres: %v", err)
	}

	if err := database.RunMigrations(db, "../../migrations"); err != nil {
		db.Close()
		t.Fatalf("Failed to run test database migrations: %v", err)
	}

	redisClient := redis.NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, "TRUNCATE audit_logs, wallet_transactions, trades, portfolios, orders, wallets, users, assets CASCADE")
	if err != nil {
		db.Close()
		t.Fatalf("Failed to clean test tables: %v", err)
	}

	return db, redisClient
}

func seedTestData(t *testing.T, db *sqlx.DB) (int64, int64, int64) {
	t.Helper()

	ctx := context.Background()

	var buyerID int64
	err := db.QueryRowContext(ctx, "INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id",
		"buyer@test.com", "hash", user.RoleTrader).Scan(&buyerID)
	if err != nil {
		t.Fatalf("Failed to seed buyer: %v", err)
	}

	var sellerID int64
	err = db.QueryRowContext(ctx, "INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id",
		"seller@test.com", "hash", user.RoleTrader).Scan(&sellerID)
	if err != nil {
		t.Fatalf("Failed to seed seller: %v", err)
	}

	_, err = db.ExecContext(ctx, "INSERT INTO wallets (user_id, balance, currency) VALUES ($1, 10000.00, 'USD'), ($2, 10000.00, 'USD')",
		buyerID, sellerID)
	if err != nil {
		t.Fatalf("Failed to seed wallets: %v", err)
	}

	var assetID int64
	err = db.QueryRowContext(ctx, "INSERT INTO assets (symbol, name, price, quantity, status) VALUES ($1, $2, 100.00, 10000, 'ACTIVE') RETURNING id",
		"TEST", "Test Asset").Scan(&assetID)
	if err != nil {
		t.Fatalf("Failed to seed asset: %v", err)
	}

	_, err = db.ExecContext(ctx, "INSERT INTO portfolios (user_id, asset_id, quantity, average_buy_price) VALUES ($1, $2, 1000.00, 100.00)",
		sellerID, assetID)
	if err != nil {
		t.Fatalf("Failed to seed seller portfolio: %v", err)
	}

	return buyerID, sellerID, assetID
}

func TestEngine_FullAndPartialMatch(t *testing.T) {
	db, redisClient := setupTestDB(t)
	defer db.Close()
	defer redisClient.Close()

	buyerID, sellerID, assetID := seedTestData(t, db)

	orderRepo := orderrepo.NewOrderRepository(db)
	walletRepo := walletrepo.NewWalletRepository(db)
	portfolioRepo := portfoliorepo.NewPortfolioRepository(db)
	tradeRepo := traderepo.NewTradeRepository(db)

	matchQueue := make(chan *order.Order, 100)
	engine := matchingengine.New(db, orderRepo, walletRepo, portfolioRepo, tradeRepo, matchQueue, redisClient, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go engine.Run(ctx)

	sellOrder1 := &order.Order{
		UserID:   sellerID,
		AssetID:  assetID,
		Side:     order.OrderSideSell,
		Type:     order.OrderTypeLimit,
		Price:    "100.00000000",
		Quantity: "10.00000000",
		Status:   order.OrderStatusPending,
	}
	sell1, err := orderRepo.Create(ctx, nil, sellOrder1)
	if err != nil {
		t.Fatalf("Failed to create sell order: %v", err)
	}

	buyOrder1 := &order.Order{
		UserID:   buyerID,
		AssetID:  assetID,
		Side:     order.OrderSideBuy,
		Type:     order.OrderTypeLimit,
		Price:    "100.00000000",
		Quantity: "10.00000000",
		Status:   order.OrderStatusPending,
	}
	buy1, err := orderRepo.Create(ctx, nil, buyOrder1)
	if err != nil {
		t.Fatalf("Failed to create buy order: %v", err)
	}

	matchQueue <- sell1
	matchQueue <- buy1

	time.Sleep(1 * time.Second)

	fSell1, _ := orderRepo.GetByID(ctx, sell1.ID)
	fBuy1, _ := orderRepo.GetByID(ctx, buy1.ID)

	if fSell1.Status != order.OrderStatusFilled {
		t.Errorf("Expected sell order 1 to be FILLED, got %s", fSell1.Status)
	}
	if fBuy1.Status != order.OrderStatusFilled {
		t.Errorf("Expected buy order 1 to be FILLED, got %s", fBuy1.Status)
	}

	buyerWallet, _ := walletRepo.GetByUserID(ctx, buyerID)
	sellerWallet, _ := walletRepo.GetByUserID(ctx, sellerID)

	if buyerWallet.Balance != "9000.00000000" {
		t.Errorf("Expected buyer balance to be 9000.00000000, got %s", buyerWallet.Balance)
	}
	if sellerWallet.Balance != "11000.00000000" {
		t.Errorf("Expected seller balance to be 11000.00000000, got %s", sellerWallet.Balance)
	}

	buyerHolding, _ := portfolioRepo.GetHolding(ctx, buyerID, assetID)
	sellerHolding, _ := portfolioRepo.GetHolding(ctx, sellerID, assetID)

	if buyerHolding.Quantity != "10.00000000" {
		t.Errorf("Expected buyer portfolio holding to be 10.00000000, got %s", buyerHolding.Quantity)
	}
	if sellerHolding.Quantity != "990.00000000" {
		t.Errorf("Expected seller portfolio holding to be 990.00000000, got %s", sellerHolding.Quantity)
	}

	sellOrder2 := &order.Order{
		UserID:   sellerID,
		AssetID:  assetID,
		Side:     order.OrderSideSell,
		Type:     order.OrderTypeLimit,
		Price:    "100.00000000",
		Quantity: "10.00000000",
		Status:   order.OrderStatusPending,
	}
	sell2, _ := orderRepo.Create(ctx, nil, sellOrder2)

	buyOrder2 := &order.Order{
		UserID:   buyerID,
		AssetID:  assetID,
		Side:     order.OrderSideBuy,
		Type:     order.OrderTypeLimit,
		Price:    "100.00000000",
		Quantity: "4.00000000",
		Status:   order.OrderStatusPending,
	}
	buy2, _ := orderRepo.Create(ctx, nil, buyOrder2)

	matchQueue <- sell2
	matchQueue <- buy2

	time.Sleep(1 * time.Second)

	fSell2, _ := orderRepo.GetByID(ctx, sell2.ID)
	fBuy2, _ := orderRepo.GetByID(ctx, buy2.ID)

	if fSell2.Status != order.OrderStatusPartiallyFilled {
		t.Errorf("Expected sell order 2 to be PARTIALLY_FILLED, got %s", fSell2.Status)
	}
	if fSell2.RemainingQuantity != "6.00000000" {
		t.Errorf("Expected sell order 2 remaining quantity to be 6.00000000, got %s", fSell2.RemainingQuantity)
	}
	if fBuy2.Status != order.OrderStatusFilled {
		t.Errorf("Expected buy order 2 to be FILLED, got %s", fBuy2.Status)
	}
}

func TestEngine_ConcurrencySafety(t *testing.T) {
	db, redisClient := setupTestDB(t)
	defer db.Close()
	defer redisClient.Close()

	buyerID, sellerID, assetID := seedTestData(t, db)

	orderRepo := orderrepo.NewOrderRepository(db)
	walletRepo := walletrepo.NewWalletRepository(db)
	portfolioRepo := portfoliorepo.NewPortfolioRepository(db)
	tradeRepo := traderepo.NewTradeRepository(db)

	matchQueue := make(chan *order.Order, 200)
	engine := matchingengine.New(db, orderRepo, walletRepo, portfolioRepo, tradeRepo, matchQueue, redisClient, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go engine.Run(ctx)

	var wg sync.WaitGroup
	ordersCount := 25

	for i := 0; i < ordersCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			buyOrder := &order.Order{
				UserID:         buyerID,
				AssetID:        assetID,
				Side:           order.OrderSideBuy,
				Type:           order.OrderTypeLimit,
				Price:          "100.00000000",
				Quantity:       "1.00000000",
				Status:         order.OrderStatusPending,
				IdempotencyKey: ptr(fmt.Sprintf("buyer-idem-%d", index)),
			}
			placed, err := orderRepo.Create(ctx, nil, buyOrder)
			if err == nil {
				matchQueue <- placed
			}
		}(i)
	}

	for i := 0; i < ordersCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sellOrder := &order.Order{
				UserID:         sellerID,
				AssetID:        assetID,
				Side:           order.OrderSideSell,
				Type:           order.OrderTypeLimit,
				Price:          "100.00000000",
				Quantity:       "1.00000000",
				Status:         order.OrderStatusPending,
				IdempotencyKey: ptr(fmt.Sprintf("seller-idem-%d", index)),
			}
			placed, err := orderRepo.Create(ctx, nil, sellOrder)
			if err == nil {
				matchQueue <- placed
			}
		}(i)
	}

	wg.Wait()

	time.Sleep(3 * time.Second)

	buyerWallet, _ := walletRepo.GetByUserID(ctx, buyerID)
	sellerWallet, _ := walletRepo.GetByUserID(ctx, sellerID)

	bBal, _ := strconv.ParseFloat(buyerWallet.Balance, 64)
	sBal, _ := strconv.ParseFloat(sellerWallet.Balance, 64)

	if bBal != 7500.0 {
		t.Errorf("Expected buyer final balance to be 7500.00, got %f", bBal)
	}
	if sBal != 12500.0 {
		t.Errorf("Expected seller final balance to be 12500.00, got %f", sBal)
	}

	buyerHolding, _ := portfolioRepo.GetHolding(ctx, buyerID, assetID)
	sellerHolding, _ := portfolioRepo.GetHolding(ctx, sellerID, assetID)

	bQty, _ := strconv.ParseFloat(buyerHolding.Quantity, 64)
	sQty, _ := strconv.ParseFloat(sellerHolding.Quantity, 64)

	if bQty != 25.0 {
		t.Errorf("Expected buyer final quantity to be 25.0, got %f", bQty)
	}
	if sQty != 975.0 {
		t.Errorf("Expected seller final quantity to be 975.0, got %f", sQty)
	}
}

func ptr(s string) *string {
	return &s
}
