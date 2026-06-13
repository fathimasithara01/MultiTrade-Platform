package matchingengine

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/kafka"
	"github.com/fathimasithara01/multitrade-platform/internal/order"
	orderrepo "github.com/fathimasithara01/multitrade-platform/internal/order/repository"
	portfoliorepo "github.com/fathimasithara01/multitrade-platform/internal/portfolio/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/redis"
	traderepo "github.com/fathimasithara01/multitrade-platform/internal/trade/repository"
	walletrepo "github.com/fathimasithara01/multitrade-platform/internal/wallet/repository"
)

type Engine struct {
	db            *sqlx.DB
	orderRepo     orderrepo.OrderRepository
	walletRepo    walletrepo.WalletRepository
	portfolioRepo portfoliorepo.PortfolioRepository
	tradeRepo     traderepo.TradeRepository
	queue         <-chan *order.Order
	cacheClient   *redis.Client
	kafkaProducer *kafka.Producer
}

func New(
	db *sqlx.DB,
	orderRepo orderrepo.OrderRepository,
	walletRepo walletrepo.WalletRepository,
	portfolioRepo portfoliorepo.PortfolioRepository,
	tradeRepo traderepo.TradeRepository,
	queue <-chan *order.Order,
	cacheClient *redis.Client,
	kafkaProducer *kafka.Producer,
) *Engine {
	return &Engine{
		db:            db,
		orderRepo:     orderRepo,
		walletRepo:    walletRepo,
		portfolioRepo: portfolioRepo,
		tradeRepo:     tradeRepo,
		queue:         queue,
		cacheClient:   cacheClient,
		kafkaProducer: kafkaProducer,
	}
}

func (e *Engine) Run(ctx context.Context) {
	log.Info().Msg("Matching engine started")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Matching engine stopped")
			return
		case o, ok := <-e.queue:
			if !ok {
				log.Info().Msg("Matching engine queue closed")
				return
			}
			if err := e.processOrder(ctx, o); err != nil {
				log.Error().Err(err).
					Int64("order_id", o.ID).
					Msg("Matching engine: error processing order")
			}
		}
	}
}

func (e *Engine) processOrder(ctx context.Context, incoming *order.Order) error {
	oppositeSide := order.OrderSideSell
	if incoming.Side == order.OrderSideSell {
		oppositeSide = order.OrderSideBuy
	}

	for {
		current, err := e.orderRepo.GetByID(ctx, incoming.ID)
		if err != nil {
			return fmt.Errorf("processOrder: re-fetch incoming: %w", err)
		}
		if current.Status == order.OrderStatusFilled || current.Status == order.OrderStatusCancelled {
			return nil
		}

		incomingRemaining := mustParseBig(current.RemainingQuantity)
		if incomingRemaining.Cmp(bigZero()) <= 0 {
			return nil
		}

		candidate, err := e.findBestCandidate(ctx, incoming.AssetID, oppositeSide, current)
		if err != nil {
			return fmt.Errorf("processOrder: find candidate: %w", err)
		}
		if candidate == nil {
			return nil
		}

		matched, err := e.executeTrade(ctx, current, candidate)
		if err != nil {
			log.Error().Err(err).
				Int64("incoming_id", current.ID).
				Int64("candidate_id", candidate.ID).
				Msg("Matching engine: trade execution failed")
			return err
		}
		if !matched {
			freshIncoming, err := e.orderRepo.GetByID(ctx, incoming.ID)
			if err != nil {
				return fmt.Errorf("processOrder: check incoming after unmatched: %w", err)
			}
			if isOpen(freshIncoming.Status) {
				continue
			}
			return nil
		}

		current, err = e.orderRepo.GetByID(ctx, incoming.ID)
		if err != nil {
			return fmt.Errorf("processOrder: re-fetch after fill: %w", err)
		}
		if current.Status == order.OrderStatusFilled {
			return nil
		}
	}
}
