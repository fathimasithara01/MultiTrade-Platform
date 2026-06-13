package service

import (
	"context"

	"github.com/fathimasithara01/multitrade-platform/internal/portfolio"
	portfoliorepo "github.com/fathimasithara01/multitrade-platform/internal/portfolio/repository"
)

type PortfolioService interface {
	GetMyPortfolio(ctx context.Context, userID int64) ([]portfolio.PortfolioHolding, error)
}

type portfolioService struct {
	repo portfoliorepo.PortfolioRepository
}

func NewPortfolioService(repo portfoliorepo.PortfolioRepository) PortfolioService {
	return &portfolioService{repo: repo}
}

func (s *portfolioService) GetMyPortfolio(ctx context.Context, userID int64) ([]portfolio.PortfolioHolding, error) {
	return s.repo.ListHoldings(ctx, userID)
}
