package cronjobs

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/order"
)

func ExpireStaleOrders(db *sqlx.DB) int64 {
	log.Info().Msg("Running ExpireStaleOrders job...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE status IN ('PENDING', 'PARTIALLY_FILLED')
		  AND created_at < NOW() - INTERVAL '24 hours'`

	res, err := db.ExecContext(ctx, query, order.OrderStatusCancelled)
	if err != nil {
		log.Error().Err(err).Msg("ExpireStaleOrders job failed")
		return 0
	}

	rows, _ := res.RowsAffected()
	log.Info().Int64("expired_orders_count", rows).Msg("ExpireStaleOrders job completed successfully")
	return rows
}
