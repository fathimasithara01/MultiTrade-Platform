package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus counters, histograms, and gauges for the platform.
var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tradeverse",
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tradeverse",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	OrdersPlacedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tradeverse",
		Name:      "orders_placed_total",
		Help:      "Total orders placed, partitioned by side (BUY/SELL).",
	}, []string{"side"})

	TradesExecutedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tradeverse",
		Name:      "trades_executed_total",
		Help:      "Total number of trades executed by the matching engine.",
	})

	MatchingEngineLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tradeverse",
		Name:      "matching_engine_duration_seconds",
		Help:      "Time taken to execute one trade in the matching engine.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})

	WebSocketConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tradeverse",
		Name:      "websocket_connections_active",
		Help:      "Number of currently active WebSocket connections.",
	})

	// RateLimitHits counts requests blocked by the rate limiter, by path.
	RateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tradeverse",
		Name:      "rate_limit_hits_total",
		Help:      "Number of requests blocked by the rate limiter.",
	}, []string{"path"})

	WalletTransactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tradeverse",
		Name:      "wallet_transactions_total",
		Help:      "Total wallet transactions by type.",
	}, []string{"type"})
)

// PrometheusMiddleware records HTTP request count and latency per route.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// Use the templated path to avoid cardinality explosion from path params.
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
