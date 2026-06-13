package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

func (c *Client) SubscribePattern(ctx context.Context, pattern string) *redis.PubSub {
	return c.rdb.PSubscribe(ctx, pattern)
}
