package repository

import (
	"context"
	"strconv"

	"github.com/fathimasithara01/multitrade-platform/internal/redis"
)

// AnalyticsRepository provides access to trading analytics metrics stored in Redis.
type AnalyticsRepository interface {
	RecordTrade(ctx context.Context, priceStr, quantityStr string) error
	GetTotalVolume(ctx context.Context) (float64, error)
	GetTradeCount(ctx context.Context) (int64, error)
}

type analyticsRepository struct {
	redisClient *redis.Client
}

// NewAnalyticsRepository constructs an analytics repository backed by Redis.
func NewAnalyticsRepository(redisClient *redis.Client) AnalyticsRepository {
	return &analyticsRepository{redisClient: redisClient}
}

func (r *analyticsRepository) RecordTrade(ctx context.Context, priceStr, quantityStr string) error {
	r.redisClient.RecordAnalyticsTrade(ctx, priceStr, quantityStr)
	return nil
}

func (r *analyticsRepository) GetTotalVolume(ctx context.Context) (float64, error) {
	val, err := r.redisClient.Raw().Get(ctx, "analytics:total_volume").Result()
	if err != nil {
		return 0, nil
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}

func (r *analyticsRepository) GetTradeCount(ctx context.Context) (int64, error) {
	val, err := r.redisClient.Raw().Get(ctx, "analytics:trade_count").Result()
	if err != nil {
		return 0, nil
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}
