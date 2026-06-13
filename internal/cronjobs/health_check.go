package cronjobs

import (
	"context"
	"net"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/redis"
)

func SystemHealthCheck(db *sqlx.DB, redisClient *redis.Client, brokers []string) {
	log.Info().Msg("Running SystemHealthCheck job...")

	dbStatus := "UP"
	ctxDB, cancelDB := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelDB()
	if err := db.PingContext(ctxDB); err != nil {
		dbStatus = "DOWN"
		log.Error().Err(err).Msg("Health Check: Postgres database ping failed")
	}

	redisStatus := "UP"
	if redisClient != nil {
		ctxRedis, cancelRedis := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelRedis()
		if err := redisClient.Ping(ctxRedis); err != nil {
			redisStatus = "DOWN"
			log.Error().Err(err).Msg("Health Check: Redis cache ping failed")
		}
	} else {
		redisStatus = "DISABLED"
	}

	kafkaStatus := "UP"
	for _, b := range brokers {
		conn, err := net.DialTimeout("tcp", b, 2*time.Second)
		if err != nil {
			kafkaStatus = "DOWN"
			log.Error().Err(err).Str("broker", b).Msg("Health Check: Kafka broker TCP dial failed")
			break
		}
		conn.Close()
	}

	log.Info().
		Str("database", dbStatus).
		Str("redis", redisStatus).
		Str("kafka", kafkaStatus).
		Msg("Health check completed")
}
