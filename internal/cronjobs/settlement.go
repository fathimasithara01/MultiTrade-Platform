package cronjobs

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/trade"
)

func VerifyTradeSettlements(db *sqlx.DB) int {
	log.Info().Msg("Running VerifyTradeSettlements job...")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var trades []trade.Trade
	err := db.SelectContext(ctx, &trades, "SELECT id, buy_order_id, sell_order_id, price, quantity, buyer_id, seller_id FROM trades")
	if err != nil {
		log.Error().Err(err).Msg("VerifyTradeSettlements: failed to fetch trades")
		return 0
	}

	discrepancies := 0

	for _, t := range trades {
		price, _ := new(big.Float).SetString(t.Price)
		qty, _ := new(big.Float).SetString(t.Quantity)
		cost := new(big.Float).Mul(price, qty)
		costStr := fmt.Sprintf("%.8f", cost)

		var buyerTxExists bool
		err = db.GetContext(ctx, &buyerTxExists, `
			SELECT EXISTS(
				SELECT 1 FROM wallet_transactions wt
				JOIN wallets w ON wt.wallet_id = w.id
				WHERE w.user_id = $1 
				  AND wt.transaction_type = 'WITHDRAWAL' 
				  AND wt.amount = $2 
				  AND wt.status = 'COMPLETED'
				  AND wt.description LIKE '%' || $3 || '%'
			)`, t.BuyerID, costStr, fmt.Sprintf("order #%d", t.BuyOrderID))
		if err != nil || !buyerTxExists {
			discrepancies++
			log.Error().Int64("trade_id", t.ID).Int64("buyer_id", t.BuyerID).Str("expected_cost", costStr).
				Msg("ALERT: Trade settlement: Missing/mismatched wallet transaction for buyer debit")
		}

		var sellerTxExists bool
		err = db.GetContext(ctx, &sellerTxExists, `
			SELECT EXISTS(
				SELECT 1 FROM wallet_transactions wt
				JOIN wallets w ON wt.wallet_id = w.id
				WHERE w.user_id = $1 
				  AND wt.transaction_type = 'DEPOSIT' 
				  AND wt.amount = $2 
				  AND wt.status = 'COMPLETED'
				  AND wt.description LIKE '%' || $3 || '%'
			)`, t.SellerID, costStr, fmt.Sprintf("order #%d", t.SellOrderID))
		if err != nil || !sellerTxExists {
			discrepancies++
			log.Error().Int64("trade_id", t.ID).Int64("seller_id", t.SellerID).Str("expected_credit", costStr).
				Msg("ALERT: Trade settlement: Missing/mismatched wallet transaction for seller credit")
		}
	}

	log.Info().Int("settlement_discrepancies_found", discrepancies).Msg("VerifyTradeSettlements job completed")
	return discrepancies
}
