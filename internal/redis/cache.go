package redis

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/rs/zerolog/log"
)

const DefaultTTL = 10 * time.Minute

func AssetPriceKey(assetID int64) string {
	return fmt.Sprintf("asset:%d:price", assetID)
}

func (c *Client) SetPrice(ctx context.Context, assetID int64, price string) {
	if err := c.rdb.Set(ctx, AssetPriceKey(assetID), price, DefaultTTL).Err(); err != nil {
		log.Warn().Err(err).Int64("asset_id", assetID).Msg("Redis SetPrice failed")
	}
}

func (c *Client) GetPrice(ctx context.Context, assetID int64) (string, bool) {
	val, err := c.rdb.Get(ctx, AssetPriceKey(assetID)).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

func (c *Client) InvalidatePrice(ctx context.Context, assetID int64) {
	if err := c.rdb.Del(ctx, AssetPriceKey(assetID)).Err(); err != nil {
		log.Warn().Err(err).Int64("asset_id", assetID).Msg("Redis InvalidatePrice failed")
	}
}

func (c *Client) RecordAnalyticsTrade(ctx context.Context, priceStr, quantityStr string) {
	price, _ := new(big.Float).SetString(priceStr)
	qty, _ := new(big.Float).SetString(quantityStr)
	if price == nil || qty == nil {
		return
	}

	costFloat, _ := new(big.Float).Mul(price, qty).Float64()

	if err := c.rdb.IncrByFloat(ctx, "analytics:total_volume", costFloat).Err(); err != nil {
		log.Warn().Err(err).Msg("Redis IncrByFloat total_volume failed")
	}
	if err := c.rdb.Incr(ctx, "analytics:trade_count").Err(); err != nil {
		log.Warn().Err(err).Msg("Redis Incr trade_count failed")
	}
	log.Debug().Float64("added_volume", costFloat).Msg("Redis trade analytics recorded")
}
