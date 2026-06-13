package cronjobs

import (
	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/redis"
)

type Scheduler struct {
	cron        *cron.Cron
	db          *sqlx.DB
	redisClient *redis.Client
	brokers     []string
}

func NewScheduler(db *sqlx.DB, redisClient *redis.Client, brokers []string) *Scheduler {
	return &Scheduler{
		cron:        cron.New(),
		db:          db,
		redisClient: redisClient,
		brokers:     brokers,
	}
}

func (s *Scheduler) Start() {
	_, _ = s.cron.AddFunc("0 * * * *", func() {
		ExpireStaleOrders(s.db)
	})

	_, _ = s.cron.AddFunc("0 0 * * *", func() {
		ReconcileWallets(s.db)
	})

	_, _ = s.cron.AddFunc("0 1 * * *", func() {
		VerifyTradeSettlements(s.db)
	})

	_, _ = s.cron.AddFunc("*/5 * * * *", func() {
		SystemHealthCheck(s.db, s.redisClient, s.brokers)
	})

	s.cron.Start()
	log.Info().Msg("Cron Scheduler started background maintenance jobs")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Info().Msg("Cron Scheduler stopped")
}
