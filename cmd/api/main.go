package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/tradeverse/internal/config"
	"github.com/fathimasithara01/tradeverse/internal/database"
)

func main() {
	// Initialize structured logging (Zerolog)
	// For local development, console writer provides pretty logging.
	// In production, standard JSON logging is used.
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("APP_ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	log.Info().Msg("Starting Multi-Role Trading Platform API Server...")

	// 1. Load Configurations
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	log.Info().Str("env", cfg.App.Env).Str("port", cfg.App.Port).Msg("Configuration loaded successfully")

	// Set Gin mode
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 2. Initialize Database & Run Migrations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.ConnectDB(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database connection")
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing database connection")
		}
	}()

	// Execute migrations
	migrationsPath := "./migrations"
	if err := database.RunMigrations(db, migrationsPath); err != nil {
		log.Fatal().Err(err).Msg("Database migration failed")
	}

	// 3. Initialize HTTP Server
	r := gin.New()
	r.Use(gin.Recovery())

	// Structured logging middleware for Gin
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		param := gin.LogFormatterParams{
			Request: c.Request,
			Keys:    c.Keys,
		}

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg(param.ErrorMessage)
	})

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Define Health Check Endpoint
	r.GET("/health", func(c *gin.Context) {
		// Ping database inside the health check to verify connection viability
		dbStatus := "UP"
		ctxPing, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer pingCancel()

		if err := db.PingContext(ctxPing); err != nil {
			log.Error().Err(err).Msg("Database health check ping failed")
			dbStatus = "DOWN"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"env":       cfg.App.Env,
			"database":  dbStatus,
		})
	})

	// Setup server
	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: r,
	}

	// Initializing the server in a goroutine so that it won't block the graceful shutdown handling
	go func() {
		log.Info().Msgf("Server listening on port %s", cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed to start")
		}
	}()

	// 4. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	// Kill (no param) default send syscall.SIGTERM
	// Kill -2 is syscall.SIGINT
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server gracefully...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctxShutdown, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exiting cleanly")
}
