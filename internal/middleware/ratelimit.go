package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RateLimiter implements a Redis-backed sliding window rate limiter.
// Each unique key (IP or user) is limited to maxRequests per window.
type RateLimiter struct {
	rdb         *redis.Client
	maxRequests int64
	window      time.Duration
}

// NewRateLimiter constructs a RateLimiter backed by the supplied Redis client.
func NewRateLimiter(rdb *redis.Client, maxRequests int64, window time.Duration) *RateLimiter {
	return &RateLimiter{rdb: rdb, maxRequests: maxRequests, window: window}
}

// Middleware returns a Gin handler that enforces the rate limit.
// keyFn derives the bucket key from the request (e.g. by IP or user_id).
func (rl *RateLimiter) Middleware(keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s", keyFn(c))
		ctx, cancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
		defer cancel()

		pipe := rl.rdb.Pipeline()
		now := time.Now().UnixNano()
		windowStart := now - rl.window.Nanoseconds()

		// Remove old entries outside the window
		pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
		// Count remaining entries
		countCmd := pipe.ZCard(ctx, key)
		// Add current request
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)})
		// Set expiry on the whole key
		pipe.Expire(ctx, key, rl.window)

		if _, err := pipe.Exec(ctx); err != nil {
			// Redis unavailable — fail open (allow request)
			log.Warn().Err(err).Str("key", key).Msg("Rate limiter Redis error, allowing request")
			c.Next()
			return
		}

		count := countCmd.Val()
		remaining := rl.maxRequests - count
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(rl.window).Unix()))

		if count >= rl.maxRequests {
			// Record the blocked request in Prometheus
			RateLimitHits.WithLabelValues(c.FullPath()).Inc()

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": fmt.Sprintf("%.0fs", rl.window.Seconds()),
			})
			return
		}

		c.Next()
	}
}

// ByIP is the standard key function — rate limits per client IP.
func ByIP(c *gin.Context) string {
	return c.ClientIP()
}

// ByUserID rate limits per authenticated user (falls back to IP if unauthenticated).
func ByUserID(c *gin.Context) string {
	if uid, exists := c.Get(ContextKeyUserID); exists {
		return fmt.Sprintf("user:%v", uid)
	}
	return c.ClientIP()
}
