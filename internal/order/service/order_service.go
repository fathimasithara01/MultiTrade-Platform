package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	asset "github.com/fathimasithara01/multitrade-platform/internal/asset"
	assetrepo "github.com/fathimasithara01/multitrade-platform/internal/asset/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/kafka"
	"github.com/fathimasithara01/multitrade-platform/internal/order"
	"github.com/fathimasithara01/multitrade-platform/internal/order/dto"
	orderrepo "github.com/fathimasithara01/multitrade-platform/internal/order/repository"
	portfoliorepo "github.com/fathimasithara01/multitrade-platform/internal/portfolio/repository"
	walletrepo "github.com/fathimasithara01/multitrade-platform/internal/wallet/repository"
)

var (
	ErrOrderInsufficientFunds   = errors.New("insufficient wallet balance to place buy order")
	ErrOrderInsufficientHolding = errors.New("insufficient asset holding to place sell order")
	ErrOrderNotOwned            = errors.New("order does not belong to this user")
	ErrOrderNotCancellable      = errors.New("only PENDING or PARTIALLY_FILLED orders can be cancelled")
	ErrOrderAssetNotActive      = errors.New("asset is not active")
)

type OrderService struct {
	db            *sqlx.DB
	orderRepo     orderrepo.OrderRepository
	walletRepo    walletrepo.WalletRepository
	portfolioRepo portfoliorepo.PortfolioRepository
	assetRepo     assetrepo.AssetRepository
	matchQueue    chan<- *order.Order
	kafkaProducer *kafka.Producer
}

func NewOrderService(
	db *sqlx.DB,
	orderRepo orderrepo.OrderRepository,
	walletRepo walletrepo.WalletRepository,
	portfolioRepo portfoliorepo.PortfolioRepository,
	assetRepo assetrepo.AssetRepository,
	matchQueue chan<- *order.Order,
	kafkaProducer *kafka.Producer,
) *OrderService {
	return &OrderService{
		db:            db,
		orderRepo:     orderRepo,
		walletRepo:    walletRepo,
		portfolioRepo: portfolioRepo,
		assetRepo:     assetRepo,
		matchQueue:    matchQueue,
		kafkaProducer: kafkaProducer,
	}
}

func (s *OrderService) PlaceOrder(ctx context.Context, userID int64, input dto.PlaceOrderInput) (*order.Order, error) {
	if input.IdempotencyKey != nil && *input.IdempotencyKey != "" {
		existing, err := s.orderRepo.GetByIdempotencyKey(ctx, userID, *input.IdempotencyKey)
		if err == nil {
			log.Info().Int64("user_id", userID).Str("idem_key", *input.IdempotencyKey).
				Msg("Duplicate order request — returning existing order")
			return existing, nil
		}
		if !errors.Is(err, orderrepo.ErrOrderNotFound) {
			return nil, fmt.Errorf("place order: idempotency lookup: %w", err)
		}
	}

	price, err := parsePosDecimal(input.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price %s", input.Price)
	}
	qty, err := parsePosDecimal(input.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity %s", input.Quantity)
	}

	a, err := s.assetRepo.GetByID(ctx, input.AssetID)
	if errors.Is(err, assetrepo.ErrAssetNotFound) {
		return nil, assetrepo.ErrAssetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("place order: get asset: %w", err)
	}
	if a.Status != asset.AssetStatusActive {
		return nil, ErrOrderAssetNotActive
	}

	switch input.Side {
	case order.OrderSideBuy:
		required := new(big.Float).Mul(price, qty)
		if err := s.checkWalletBalance(ctx, userID, required); err != nil {
			return nil, err
		}

	case order.OrderSideSell:
		if err := s.checkAssetHolding(ctx, userID, input.AssetID, qty); err != nil {
			return nil, err
		}
	}

	o := &order.Order{
		UserID:         userID,
		AssetID:        input.AssetID,
		Side:           input.Side,
		Type:           order.OrderTypeLimit,
		Price:          fmtDecimal(price),
		Quantity:       fmtDecimal(qty),
		Status:         order.OrderStatusPending,
		IdempotencyKey: input.IdempotencyKey,
	}

	var placed *order.Order
	err = s.runTx(ctx, func(tx *sqlx.Tx) error {
		var createErr error
		placed, createErr = s.orderRepo.Create(ctx, tx, o)
		if createErr != nil {
			return createErr
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, orderrepo.ErrDuplicateIdempotencyKey) {
			existing, lookupErr := s.orderRepo.GetByIdempotencyKey(ctx, userID, *input.IdempotencyKey)
			if lookupErr == nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("place order: insert: %w", err)
	}

	if s.matchQueue != nil {
		select {
		case s.matchQueue <- placed:
		default:
			log.Warn().Int64("order_id", placed.ID).Msg("Matching engine queue full, order will be picked up on next scan")
		}
	}

	if s.kafkaProducer != nil {
		eventPayload := kafka.OrderEventPayload{
			OrderID:   placed.ID,
			UserID:    placed.UserID,
			AssetID:   placed.AssetID,
			Side:      placed.Side,
			Type:      placed.Type,
			Price:     placed.Price,
			Quantity:  placed.Quantity,
			Status:    placed.Status,
			Timestamp: time.Now().UTC(),
		}
		_ = s.kafkaProducer.PublishEvent(ctx, "order-events", "order.created", eventPayload)
	}

	log.Info().
		Int64("user_id", userID).
		Int64("order_id", placed.ID).
		Str("side", placed.Side).
		Str("asset_id", fmt.Sprintf("%d", placed.AssetID)).
		Msg("Order placed")

	return placed, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, userID, orderID int64) (*order.Order, error) {
	var cancelled *order.Order

	err := s.runTx(ctx, func(tx *sqlx.Tx) error {
		o, err := s.orderRepo.GetByIDForUpdate(ctx, tx, orderID)
		if errors.Is(err, orderrepo.ErrOrderNotFound) {
			return orderrepo.ErrOrderNotFound
		}
		if err != nil {
			return fmt.Errorf("cancel order: get: %w", err)
		}

		if o.UserID != userID {
			return ErrOrderNotOwned
		}

		if o.Status != order.OrderStatusPending && o.Status != order.OrderStatusPartiallyFilled {
			return ErrOrderNotCancellable
		}

		if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, order.OrderStatusCancelled); err != nil {
			return err
		}

		cancelled = o
		cancelled.Status = order.OrderStatusCancelled
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.kafkaProducer != nil {
		eventPayload := kafka.OrderEventPayload{
			OrderID:   cancelled.ID,
			UserID:    cancelled.UserID,
			AssetID:   cancelled.AssetID,
			Side:      cancelled.Side,
			Type:      cancelled.Type,
			Price:     cancelled.Price,
			Quantity:  cancelled.Quantity,
			Status:    cancelled.Status,
			Timestamp: time.Now().UTC(),
		}
		_ = s.kafkaProducer.PublishEvent(ctx, "order-events", "order.cancelled", eventPayload)
	}

	log.Info().Int64("user_id", userID).Int64("order_id", orderID).Msg("Order cancelled")
	return cancelled, nil
}

func (s *OrderService) GetOrder(ctx context.Context, userID, orderID int64) (*order.Order, error) {
	o, err := s.orderRepo.GetByID(ctx, orderID)
	if errors.Is(err, orderrepo.ErrOrderNotFound) {
		return nil, orderrepo.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.UserID != userID {
		return nil, ErrOrderNotOwned
	}
	return o, nil
}

func (s *OrderService) ListOrders(ctx context.Context, userID int64, page, pageSize int) (*dto.OrderPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	orders, err := s.orderRepo.ListByUser(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.orderRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert orders to DTOs
	orderDTOs := make([]dto.OrderDTO, len(orders))
	for i, o := range orders {
		orderDTOs[i] = dto.OrderDTO{
			ID:                o.ID,
			UserID:            o.UserID,
			AssetID:           o.AssetID,
			Side:              o.Side,
			Type:              o.Type,
			Price:             o.Price,
			Quantity:          o.Quantity,
			FilledQuantity:    o.FilledQuantity,
			RemainingQuantity: o.RemainingQuantity,
			Status:            o.Status,
			IdempotencyKey:    o.IdempotencyKey,
			CreatedAt:         o.CreatedAt,
			UpdatedAt:         o.UpdatedAt,
		}
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return &dto.OrderPage{
		Data:       orderDTOs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *OrderService) checkWalletBalance(ctx context.Context, userID int64, required *big.Float) error {
	w, err := s.walletRepo.GetByUserID(ctx, userID)
	if errors.Is(err, walletrepo.ErrWalletNotFound) {
		return ErrOrderInsufficientFunds
	}
	if err != nil {
		return fmt.Errorf("check wallet: %w", err)
	}
	balance := mustParseBigFloat(w.Balance)
	if balance.Cmp(required) < 0 {
		return ErrOrderInsufficientFunds
	}
	return nil
}

func (s *OrderService) checkAssetHolding(ctx context.Context, userID, assetID int64, qty *big.Float) error {
	holding, err := s.portfolioRepo.GetHolding(ctx, userID, assetID)
	if errors.Is(err, portfoliorepo.ErrPortfolioNotFound) {
		return ErrOrderInsufficientHolding
	}
	if err != nil {
		return fmt.Errorf("check holding: %w", err)
	}
	held := mustParseBigFloat(holding.Quantity)
	if held.Cmp(qty) < 0 {
		return ErrOrderInsufficientHolding
	}
	return nil
}

func (s *OrderService) runTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func parsePosDecimal(s string) (*big.Float, error) {
	f, ok := new(big.Float).SetPrec(64).SetString(s)
	if !ok || f.Cmp(new(big.Float)) <= 0 {
		return nil, errors.New("invalid positive decimal")
	}
	return f, nil
}

func mustParseBigFloat(s string) *big.Float {
	f, _ := new(big.Float).SetPrec(64).SetString(s)
	return f
}

func fmtDecimal(f *big.Float) string {
	return fmt.Sprintf("%.8f", f)
}
