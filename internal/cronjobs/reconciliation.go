package cronjobs

import (
	"context"
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/wallet"
)

func ReconcileWallets(db *sqlx.DB) int {
	log.Info().Msg("Running ReconcileWallets job...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wallets []wallet.Wallet
	err := db.SelectContext(ctx, &wallets, "SELECT id, user_id, balance FROM wallets")
	if err != nil {
		log.Error().Err(err).Msg("ReconcileWallets: failed to fetch wallets")
		return 0
	}

	discrepanciesCount := 0

	for _, w := range wallets {
		var sumStr *string
		err := db.GetContext(ctx, &sumStr, `
			SELECT COALESCE(SUM(
				CASE WHEN transaction_type = 'DEPOSIT' THEN amount
				     WHEN transaction_type = 'WITHDRAWAL' THEN -amount
				     ELSE 0.00000000 END
			), 0.00000000)
			FROM wallet_transactions
			WHERE wallet_id = $1 AND status = 'COMPLETED'`, w.ID)
		if err != nil {
			log.Error().Err(err).Int64("wallet_id", w.ID).Msg("ReconcileWallets: failed to sum transactions")
			continue
		}

		walletBal, _ := new(big.Float).SetString(w.Balance)
		var txSum *big.Float
		if sumStr != nil {
			txSum, _ = new(big.Float).SetString(*sumStr)
		} else {
			txSum = new(big.Float)
		}

		if walletBal == nil || txSum == nil {
			log.Error().Int64("wallet_id", w.ID).Msg("ReconcileWallets: failed to parse numeric balance or transaction sum")
			continue
		}

		diff := new(big.Float).Sub(walletBal, txSum)
		diff = new(big.Float).Abs(diff)
		limit, _ := new(big.Float).SetString("0.00000001")

		if diff.Cmp(limit) > 0 {
			discrepanciesCount++
			log.Error().
				Int64("wallet_id", w.ID).
				Int64("user_id", w.UserID).
				Str("wallet_balance", w.Balance).
				Str("transaction_sum", txSum.Text('f', 8)).
				Msg("ALERT: Wallet balance reconciliation mismatch found!")
		}
	}

	log.Info().Int("discrepancies_found", discrepanciesCount).Msg("ReconcileWallets job completed")
	return discrepanciesCount
}
