package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/kafka"
	notificationsvc "github.com/fathimasithara01/multitrade-platform/internal/notification/service"
)

// StartNotificationConsumer fans out over topics and prints user-facing notifications.
func StartNotificationConsumer(ctx context.Context, brokers []string, topics []string, svc notificationsvc.NotificationService) {
	for _, topic := range topics {
		go func(t string) {
			r := kafkago.NewReader(kafkago.ReaderConfig{
				Brokers:  brokers,
				Topic:    t,
				GroupID:  "notification-group",
				MaxBytes: 10e6,
			})
			defer r.Close()
			log.Info().Str("topic", t).Msg("Notification consumer started")

			for {
				select {
				case <-ctx.Done():
					return
				default:
					m, err := r.ReadMessage(ctx)
					if err != nil {
						if ctx.Err() != nil {
							return
						}
						log.Error().Err(err).Str("topic", t).Msg("notification consumer read error")
						time.Sleep(time.Second)
						continue
					}

					var we kafka.WrappedEvent
					if err := json.Unmarshal(m.Value, &we); err != nil {
						log.Warn().Err(err).Msg("notification consumer: failed to unmarshal event")
						continue
					}

					switch we.Type {
					case "order.created":
						var p kafka.OrderEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							msg := fmt.Sprintf("New order created: %s %s units of Asset #%d at $%s",
								p.Side, p.Quantity, p.AssetID, p.Price)
							_ = svc.SendNotification(ctx, p.UserID, msg)
						}
					case "order.cancelled":
						var p kafka.OrderEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							_ = svc.SendNotification(ctx, p.UserID, fmt.Sprintf("Order #%d cancelled", p.OrderID))
						}
					case "trade.executed":
						var p kafka.TradeEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							_ = svc.SendNotification(ctx, p.BuyerID,
								fmt.Sprintf("Trade executed! You bought %s units of Asset #%d at $%s",
									p.Quantity, p.AssetID, p.Price))
							_ = svc.SendNotification(ctx, p.SellerID,
								fmt.Sprintf("Trade executed! You sold %s units of Asset #%d at $%s",
									p.Quantity, p.AssetID, p.Price))
						}
					case "wallet.updated":
						var p kafka.WalletEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							_ = svc.SendNotification(ctx, p.UserID,
								fmt.Sprintf("Wallet updated. Type: %s, Change: $%s, New Balance: $%s",
									p.TransactionType, p.ChangeAmount, p.Balance))
						}
					}
				}
			}
		}(topic)
	}
}
