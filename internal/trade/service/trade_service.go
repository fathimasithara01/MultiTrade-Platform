package service

import (
	"context"

	"github.com/fathimasithara01/multitrade-platform/internal/trade"
	traderepo "github.com/fathimasithara01/multitrade-platform/internal/trade/repository"
)

type TradeService interface {
	GetByUserID(ctx context.Context, userID int64) ([]trade.Trade, error)
}

type tradeService struct {
	repo traderepo.TradeRepository
}

func NewTradeService(repo traderepo.TradeRepository) TradeService {
	return &tradeService{repo: repo}
}

func (s *tradeService) GetByUserID(ctx context.Context, userID int64) ([]trade.Trade, error) {
	return s.repo.GetByUserID(ctx, userID)
}
