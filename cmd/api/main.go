// @title           MultiTrade Platform API
// @version         1.0
// @description     A multi-role trading platform with order matching, wallets, and real-time WebSocket price feeds.
// @termsOfService  http://localhost:8080/terms/

// @contact.name   TradeVerse Engineering
// @contact.email  engineering@tradeverse.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description "Type: Bearer <token>"

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Domain: Admin
	adminhandler "github.com/fathimasithara01/multitrade-platform/internal/admin/handler"
	adminservice "github.com/fathimasithara01/multitrade-platform/internal/admin/service"
	// Domain: Audit
	auditrepo "github.com/fathimasithara01/multitrade-platform/internal/audit/repository"
	// Domain: Asset
	assethandler "github.com/fathimasithara01/multitrade-platform/internal/asset/handler"
	assetrepo "github.com/fathimasithara01/multitrade-platform/internal/asset/repository"
	assetservice "github.com/fathimasithara01/multitrade-platform/internal/asset/service"
	// Domain: Auth
	authhandler "github.com/fathimasithara01/multitrade-platform/internal/auth/handler"
	authservice "github.com/fathimasithara01/multitrade-platform/internal/auth/service"
	// Domain: Order
	orderhandler "github.com/fathimasithara01/multitrade-platform/internal/order/handler"
	orderrepo "github.com/fathimasithara01/multitrade-platform/internal/order/repository"
	orderservice "github.com/fathimasithara01/multitrade-platform/internal/order/service"
	// Domain: Portfolio
	portfoliorepo "github.com/fathimasithara01/multitrade-platform/internal/portfolio/repository"
	// Domain: Trade
	traderepo "github.com/fathimasithara01/multitrade-platform/internal/trade/repository"
	// Domain: User
	userrepo "github.com/fathimasithara01/multitrade-platform/internal/user/repository"
	// Domain: Wallet
	wallethandler "github.com/fathimasithara01/multitrade-platform/internal/wallet/handler"
	walletrepo "github.com/fathimasithara01/multitrade-platform/internal/wallet/repository"
	walletservice "github.com/fathimasithara01/multitrade-platform/internal/wallet/service"
	// Domain: Order model (for matchQueue chan type)
	ordermodel "github.com/fathimasithara01/multitrade-platform/internal/order"

	// Infrastructure
	"github.com/fathimasithara01/multitrade-platform/internal/config"
	"github.com/fathimasithara01/multitrade-platform/internal/cronjobs"
	"github.com/fathimasithara01/multitrade-platform/internal/database"
	_ "github.com/fathimasithara01/multitrade-platform/internal/docs" // Swagger generated docs
	"github.com/fathimasithara01/multitrade-platform/internal/kafka"
	"github.com/fathimasithara01/multitrade-platform/internal/matchingengine"
	"github.com/fathimasithara01/multitrade-platform/internal/middleware"
	"github.com/fathimasithara01/multitrade-platform/internal/redis"
	"github.com/fathimasithara01/multitrade-platform/internal/shared/constants"
	"github.com/fathimasithara01/multitrade-platform/internal/websocket"

	// Shared / pkg
	"github.com/fathimasithara01/multitrade-platform/pkg/jwt"
)

func main() {
	// ── Logging ────────────────────────────────────────────────────────────────
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("APP_ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
	log.Info().Msg("Starting MultiTrade Platform API Server...")

	// ── Config ─────────────────────────────────────────────────────────────────
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	log.Info().Str("env", cfg.App.Env).Str("port", cfg.App.Port).Msg("Configuration loaded")

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// ── Database ───────────────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.ConnectDB(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	if err := database.RunMigrations(db, "./migrations"); err != nil {
		log.Fatal().Err(err).Msg("Database migration failed")
	}

	// ── Infrastructure ─────────────────────────────────────────────────────────
	brokers := strings.Split(cfg.Kafka.Brokers, ",")
	kafkaProducer := kafka.NewProducer(brokers)
	defer kafkaProducer.Close()

	redisClient := redis.NewClient(cfg)
	defer redisClient.Close()

	jwtSvc := jwt.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)

	// ── Repositories ───────────────────────────────────────────────────────────
	userRepo      := userrepo.NewUserRepository(db)
	walletRepo    := walletrepo.NewWalletRepository(db)
	assetRepo     := assetrepo.NewAssetRepository(db)
	orderRepo     := orderrepo.NewOrderRepository(db)
	portfolioRepo := portfoliorepo.NewPortfolioRepository(db)
	tradeRepo     := traderepo.NewTradeRepository(db)
	auditRepo     := auditrepo.NewAuditRepository(db)

	// ── Services ───────────────────────────────────────────────────────────────
	matchQueue   := make(chan *ordermodel.Order, 1024)

	walletSvc := walletservice.NewWalletService(db, walletRepo, kafkaProducer)

	authSvc := authservice.NewAuthService(userRepo, jwtSvc)
	authSvc.SetWalletCreator(walletSvc)

	assetSvc := assetservice.NewAssetService(assetRepo, redisClient)

	orderSvc := orderservice.NewOrderService(
		db, orderRepo, walletRepo, portfolioRepo, assetRepo, matchQueue, kafkaProducer,
	)

	adminSvc := adminservice.NewAdminService(db, userRepo, auditRepo, redisClient, brokers)

	// ── Handlers ───────────────────────────────────────────────────────────────
	authH   := authhandler.NewAuthHandler(authSvc)
	walletH := wallethandler.NewWalletHandler(walletSvc)
	assetH  := assethandler.NewAssetHandler(assetSvc)
	orderH  := orderhandler.NewOrderHandler(orderSvc)
	adminH  := adminhandler.NewAdminHandler(adminSvc)

	// ── Rate limiters (Redis sliding window) ──────────────────────────────────
	loginRL  := middleware.NewRateLimiter(redisClient.Raw(), 10, time.Minute)  // 10/min per IP
	orderRL  := middleware.NewRateLimiter(redisClient.Raw(), 60, time.Minute)  // 60/min per user
	walletRL := middleware.NewRateLimiter(redisClient.Raw(), 30, time.Minute)  // 30/min per user

	// ── WebSocket hub ──────────────────────────────────────────────────────────
	wsHub := websocket.NewHub()
	go wsHub.Run()
	wsHandler := websocket.NewHandler(wsHub)

	// Redis Pub/Sub → WebSocket broadcast
	go func() {
		log.Info().Msg("Starting Redis Pub/Sub → WebSocket forward...")
		pubsub := redisClient.SubscribePattern(context.Background(), "price_updates:*")
		defer pubsub.Close()
		for msg := range pubsub.Channel() {
			var assetID int64
			fmt.Sscanf(msg.Channel, "price_updates:%d", &assetID)
			if assetID > 0 {
				wsHub.BroadcastChan <- websocket.BroadcastMessage{
					Topic:   fmt.Sprintf("asset:%d", assetID),
					Message: []byte(msg.Payload),
				}
			}
		}
	}()

	// ── Kafka consumers ────────────────────────────────────────────────────────
	kafkaTopics := []string{"order-events", "trade-events", "wallet-events"}
	kafka.StartNotificationConsumer(context.Background(), brokers, kafkaTopics)
	kafka.StartAuditConsumer(context.Background(), brokers, kafkaTopics, auditRepo)
	kafka.StartAnalyticsConsumer(context.Background(), brokers, "trade-events", redisClient)

	// ── Cron scheduler ─────────────────────────────────────────────────────────
	scheduler := cronjobs.NewScheduler(db, redisClient, brokers)
	scheduler.Start()
	defer scheduler.Stop()

	// ── Matching engine ────────────────────────────────────────────────────────
	engineCtx, engineCancel := context.WithCancel(context.Background())
	defer engineCancel()
	engine := matchingengine.New(db, orderRepo, walletRepo, portfolioRepo, tradeRepo, matchQueue, redisClient, kafkaProducer)
	go engine.Run(engineCtx)

	// ── Router ─────────────────────────────────────────────────────────────────
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.PrometheusMiddleware())
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}
		c.Next()
		log.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Str("ip", c.ClientIP()).
			Str("request_id", c.GetString("request_id")).
			Msg("request")
	})
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID", "Idempotency-Key"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Public endpoints ───────────────────────────────────────────────────────
	r.GET("/metrics",      gin.WrapH(promhttp.Handler()))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/ws",           wsHandler.Connect)
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "UP"
		ctxPing, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer pingCancel()
		if err := db.PingContext(ctxPing); err != nil {
			dbStatus = "DOWN"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"env":       cfg.App.Env,
			"database":  dbStatus,
		})
	})

	// ── /auth ─────────────────────────────────────────────────────────────────
	authGroup := r.Group("/auth")
	authGroup.POST("/register", authH.Register)
	authGroup.POST("/login",    loginRL.Middleware(middleware.ByIP), authH.Login)
	authGroup.POST("/refresh",  authH.Refresh)
	authGroup.GET("/me",        middleware.AuthMiddleware(jwtSvc), authH.Me)

	// ── /admin ────────────────────────────────────────────────────────────────
	adminGroup := r.Group("/admin",
		middleware.AuthMiddleware(jwtSvc),
		middleware.RBACMiddleware(constants.RoleAdmin),
	)
	adminGroup.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong", "role": "admin"})
	})
	adminGroup.GET("/users",                adminH.ListUsers)
	adminGroup.PATCH("/users/:id/status",   adminH.UpdateUserStatus)
	adminGroup.GET("/analytics/volume",     adminH.VolumeAnalytics)
	adminGroup.GET("/analytics/suspicious", adminH.SuspiciousUsers)
	adminGroup.GET("/health",               adminH.AdminHealth)
	adminGroup.GET("/audit",                adminH.AuditLogs)
	adminGroup.POST("/jobs/expire", func(c *gin.Context) {
		rows := cronjobs.ExpireStaleOrders(db)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "expired_orders": rows})
	})
	adminGroup.POST("/jobs/reconcile", func(c *gin.Context) {
		d := cronjobs.ReconcileWallets(db)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "discrepancies_found": d})
	})
	adminGroup.POST("/jobs/verify-settlement", func(c *gin.Context) {
		d := cronjobs.VerifyTradeSettlements(db)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "settlement_discrepancies_found": d})
	})

	// ── /wallet ───────────────────────────────────────────────────────────────
	walletGroup := r.Group("/wallet",
		middleware.AuthMiddleware(jwtSvc),
		walletRL.Middleware(middleware.ByUserID),
	)
	walletGroup.GET("",              walletH.GetWallet)
	walletGroup.POST("/deposit",     walletH.Deposit)
	walletGroup.POST("/withdraw",    walletH.Withdraw)
	walletGroup.GET("/transactions", walletH.ListTransactions)

	// ── /assets ───────────────────────────────────────────────────────────────
	assetReadGroup := r.Group("/assets", middleware.AuthMiddleware(jwtSvc))
	assetReadGroup.GET("",     assetH.ListAssets)
	assetReadGroup.GET("/:id", assetH.GetAsset)

	assetWriteGroup := r.Group("/assets",
		middleware.AuthMiddleware(jwtSvc),
		middleware.RBACMiddleware(constants.RoleBroker),
	)
	assetWriteGroup.POST("",              assetH.CreateAsset)
	assetWriteGroup.PATCH("/:id",         assetH.UpdateAsset)
	assetWriteGroup.PATCH("/:id/disable", assetH.DisableAsset)

	// ── /orders ───────────────────────────────────────────────────────────────
	orderGroup := r.Group("/orders",
		middleware.AuthMiddleware(jwtSvc),
		middleware.RBACMiddleware(constants.RoleTrader),
		orderRL.Middleware(middleware.ByUserID),
	)
	orderGroup.POST("",       orderH.PlaceOrder)
	orderGroup.GET("",        orderH.ListOrders)
	orderGroup.GET("/:id",    orderH.GetOrder)
	orderGroup.DELETE("/:id", orderH.CancelOrder)

	// ── HTTP server ────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Msgf("Server listening on port %s", cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// ── Graceful shutdown ──────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server gracefully...")

	ctxShutdown, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}
	log.Info().Msg("Server exiting cleanly")
}
