package matchingengine

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/kafka"
	"github.com/fathimasithara01/multitrade-platform/internal/order"
	"github.com/fathimasithara01/multitrade-platform/internal/trade"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet"
)

func (e *Engine) executeTrade(ctx context.Context, incoming, candidate *order.Order) (bool, error) {
	var buyOrder, sellOrder *order.Order
	if incoming.Side == order.OrderSideBuy {
		buyOrder = incoming
		sellOrder = candidate
	} else {
		buyOrder = candidate
		sellOrder = incoming
	}

	tradePrice := mustParseBig(candidate.Price)

	ctxTx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tx, err := e.db.BeginTxx(ctxTx, nil)
	if err != nil {
		return false, fmt.Errorf("executeTrade: begin tx: %w", err)
	}

	firstID, secondID := buyOrder.ID, sellOrder.ID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	lockedFirst, err := e.orderRepo.GetByIDForUpdate(ctxTx, tx, firstID)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: lock order %d: %w", firstID, err)
	}
	lockedSecond, err := e.orderRepo.GetByIDForUpdate(ctxTx, tx, secondID)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: lock order %d: %w", secondID, err)
	}

	if lockedFirst.ID == buyOrder.ID {
		buyOrder, sellOrder = lockedFirst, lockedSecond
	} else {
		buyOrder, sellOrder = lockedSecond, lockedFirst
	}

	if !isOpen(buyOrder.Status) || !isOpen(sellOrder.Status) {
		tx.Rollback()
		return false, nil
	}

	bPrice := mustParseBig(buyOrder.Price)
	sPrice := mustParseBig(sellOrder.Price)
	if sPrice.Cmp(bPrice) > 0 {
		tx.Rollback()
		return false, nil
	}

	buyRem := mustParseBig(buyOrder.RemainingQuantity)
	sellRem := mustParseBig(sellOrder.RemainingQuantity)
	fillQty := minBig(buyRem, sellRem)
	if fillQty.Cmp(bigZero()) <= 0 {
		tx.Rollback()
		return false, nil
	}

	fillQtyStr := fmt8f(fillQty)
	tradePriceStr := fmt8f(tradePrice)

	tradeCost := new(big.Float).Mul(fillQty, tradePrice)
	tradeCostStr := fmt8f(tradeCost)

	t := &trade.Trade{
		BuyOrderID:  buyOrder.ID,
		SellOrderID: sellOrder.ID,
		AssetID:     buyOrder.AssetID,
		Price:       tradePriceStr,
		Quantity:    fillQtyStr,
		BuyerID:     buyOrder.UserID,
		SellerID:    sellOrder.UserID,
	}
	insertedTrade, err := e.tradeRepo.Insert(ctxTx, tx, t)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: insert trade: %w", err)
	}

	newBuyFilled := new(big.Float).Add(mustParseBig(buyOrder.FilledQuantity), fillQty)
	buyStatus := newStatus(newBuyFilled, mustParseBig(buyOrder.Quantity))
	if err := e.orderRepo.UpdateFill(ctxTx, tx, buyOrder.ID, fillQtyStr, buyStatus); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: update buy order: %w", err)
	}

	newSellFilled := new(big.Float).Add(mustParseBig(sellOrder.FilledQuantity), fillQty)
	sellStatus := newStatus(newSellFilled, mustParseBig(sellOrder.Quantity))
	if err := e.orderRepo.UpdateFill(ctxTx, tx, sellOrder.ID, fillQtyStr, sellStatus); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: update sell order: %w", err)
	}

	buyerWallet, err := e.walletRepo.GetByUserIDForUpdate(ctxTx, tx, buyOrder.UserID)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: lock buyer wallet: %w", err)
	}
	buyerBalance := mustParseBig(buyerWallet.Balance)
	buyerNewBal := new(big.Float).Sub(buyerBalance, tradeCost)
	if buyerNewBal.Cmp(bigZero()) < 0 {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: buyer has insufficient balance at execution time")
	}
	if err := e.walletRepo.UpdateBalance(ctxTx, tx, buyerWallet.ID, fmt8f(buyerNewBal), buyerWallet.Version); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: debit buyer: %w", err)
	}
	buyerDesc := fmt.Sprintf("Trade fill: buy order #%d", buyOrder.ID)
	if _, err := e.walletRepo.InsertTransaction(ctxTx, tx, &wallet.WalletTransaction{
		WalletID:        buyerWallet.ID,
		Amount:          tradeCostStr,
		TransactionType: wallet.TxTypeWithdrawal,
		Status:          wallet.TxStatusCompleted,
		Description:     &buyerDesc,
	}); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: buyer wallet tx: %w", err)
	}

	sellerWallet, err := e.walletRepo.GetByUserIDForUpdate(ctxTx, tx, sellOrder.UserID)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: lock seller wallet: %w", err)
	}
	sellerBalance := mustParseBig(sellerWallet.Balance)
	sellerNewBal := new(big.Float).Add(sellerBalance, tradeCost)
	if err := e.walletRepo.UpdateBalance(ctxTx, tx, sellerWallet.ID, fmt8f(sellerNewBal), sellerWallet.Version); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: credit seller: %w", err)
	}
	sellerDesc := fmt.Sprintf("Trade fill: sell order #%d", sellOrder.ID)
	if _, err := e.walletRepo.InsertTransaction(ctxTx, tx, &wallet.WalletTransaction{
		WalletID:        sellerWallet.ID,
		Amount:          tradeCostStr,
		TransactionType: wallet.TxTypeDeposit,
		Status:          wallet.TxStatusCompleted,
		Description:     &sellerDesc,
	}); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: seller wallet tx: %w", err)
	}

	if err := e.portfolioRepo.UpsertBuyerHolding(ctxTx, tx, buyOrder.UserID, buyOrder.AssetID, fillQtyStr, tradePriceStr); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: buyer portfolio: %w", err)
	}
	if err := e.portfolioRepo.ReduceSellerHolding(ctxTx, tx, sellOrder.UserID, sellOrder.AssetID, fillQtyStr); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("executeTrade: seller portfolio: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("executeTrade: commit: %w", err)
	}

	log.Info().
		Int64("buy_order_id", buyOrder.ID).
		Int64("sell_order_id", sellOrder.ID).
		Str("price", tradePriceStr).
		Str("quantity", fillQtyStr).
		Str("cost", tradeCostStr).
		Str("buy_status", buyStatus).
		Str("sell_status", sellStatus).
		Msg("Trade executed")

	if e.cacheClient != nil {
		e.cacheClient.SetPrice(context.Background(), insertedTrade.AssetID, insertedTrade.Price)

		redisMsg := map[string]interface{}{
			"asset_id":  insertedTrade.AssetID,
			"price":     insertedTrade.Price,
			"quantity":  insertedTrade.Quantity,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		redisBytes, err := json.Marshal(redisMsg)
		if err == nil {
			redisChan := fmt.Sprintf("price_updates:%d", insertedTrade.AssetID)
			_ = e.cacheClient.Publish(context.Background(), redisChan, string(redisBytes))
		}
	}

	if e.kafkaProducer != nil {
		tradePayload := kafka.TradeEventPayload{
			TradeID:     insertedTrade.ID,
			BuyOrderID:  insertedTrade.BuyOrderID,
			SellOrderID: insertedTrade.SellOrderID,
			AssetID:     insertedTrade.AssetID,
			Price:       insertedTrade.Price,
			Quantity:    insertedTrade.Quantity,
			BuyerID:     insertedTrade.BuyerID,
			SellerID:    insertedTrade.SellerID,
			Timestamp:   time.Now().UTC(),
		}
		_ = e.kafkaProducer.PublishEvent(context.Background(), "trade-events", "trade.executed", tradePayload)

		buyerWalletPayload := kafka.WalletEventPayload{
			WalletID:        buyerWallet.ID,
			UserID:          buyOrder.UserID,
			Balance:         fmt8f(buyerNewBal),
			ChangeAmount:    tradeCostStr,
			TransactionType: "trade_debit",
			Timestamp:       time.Now().UTC(),
		}
		_ = e.kafkaProducer.PublishEvent(context.Background(), "wallet-events", "wallet.updated", buyerWalletPayload)

		sellerWalletPayload := kafka.WalletEventPayload{
			WalletID:        sellerWallet.ID,
			UserID:          sellOrder.UserID,
			Balance:         fmt8f(sellerNewBal),
			ChangeAmount:    tradeCostStr,
			TransactionType: "trade_credit",
			Timestamp:       time.Now().UTC(),
		}
		_ = e.kafkaProducer.PublishEvent(context.Background(), "wallet-events", "wallet.updated", sellerWalletPayload)
	}

	return true, nil
}
