package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/audit"
	auditrepo "github.com/fathimasithara01/multitrade-platform/internal/audit/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/redis"
)

type WrappedEvent struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func StartNotificationConsumer(ctx context.Context, brokers []string, topics []string) {
	for _, topic := range topics {
		go func(t string) {
			r := kafka.NewReader(kafka.ReaderConfig{
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
						log.Error().Err(err).Str("topic", t).Msg("Notification consumer read error")
						time.Sleep(1 * time.Second)
						continue
					}

					var we WrappedEvent
					if err := json.Unmarshal(m.Value, &we); err != nil {
						log.Warn().Err(err).Msg("Notification consumer: failed to unmarshal event wrapper")
						continue
					}

					switch we.Type {
					case "order.created":
						var p OrderEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							fmt.Printf("📢 NOTIFICATION: New order created for User #%d! Order #%d: %s %s units of Asset #%d at $%s\n",
								p.UserID, p.OrderID, p.Side, p.Quantity, p.AssetID, p.Price)
						}
					case "order.cancelled":
						var p OrderEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							fmt.Printf("📢 NOTIFICATION: Order #%d cancelled for User #%d\n", p.OrderID, p.UserID)
						}
					case "trade.executed":
						var p TradeEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							fmt.Printf("📢 NOTIFICATION: Trade executed! Buyer #%d matched Seller #%d on %s units of Asset #%d at $%s\n",
								p.BuyerID, p.SellerID, p.Quantity, p.AssetID, p.Price)
						}
					case "wallet.updated":
						var p WalletEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							fmt.Printf("📢 NOTIFICATION: Wallet #%d updated for User #%d. Type: %s, Change: $%s, New Balance: $%s\n",
								p.WalletID, p.UserID, p.TransactionType, p.ChangeAmount, p.Balance)
						}
					}
				}
			}
		}(topic)
	}
}

func StartAuditConsumer(ctx context.Context, brokers []string, topics []string, auditRepo auditrepo.AuditRepository) {
	for _, topic := range topics {
		go func(t string) {
			r := kafka.NewReader(kafka.ReaderConfig{
				Brokers:  brokers,
				Topic:    t,
				GroupID:  "audit-group",
				MaxBytes: 10e6,
			})
			defer r.Close()

			log.Info().Str("topic", t).Msg("Audit consumer started")

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
						log.Error().Err(err).Str("topic", t).Msg("Audit consumer read error")
						time.Sleep(1 * time.Second)
						continue
					}

					var we WrappedEvent
					if err := json.Unmarshal(m.Value, &we); err != nil {
						log.Warn().Err(err).Msg("Audit consumer: failed to unmarshal event wrapper")
						continue
					}

					var a audit.AuditLog
					var details string

					switch we.Type {
					case "order.created":
						var p OrderEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							userID := p.UserID
							a.UserID = &userID
							a.Action = "ORDER_CREATED"
							details = fmt.Sprintf("Order #%d created: %s %s units of asset #%d at price %s",
								p.OrderID, p.Side, p.Quantity, p.AssetID, p.Price)
							a.Details = &details
						}
					case "order.cancelled":
						var p OrderEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							userID := p.UserID
							a.UserID = &userID
							a.Action = "ORDER_CANCELLED"
							details = fmt.Sprintf("Order #%d cancelled", p.OrderID)
							a.Details = &details
						}
					case "trade.executed":
						var p TradeEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							bUserID := p.BuyerID
							buyerAudit := audit.AuditLog{
								UserID: &bUserID,
								Action: "TRADE_EXECUTED",
							}
							bDetails := fmt.Sprintf("Matched trade #%d as buyer against seller #%d for quantity %s of asset #%d at price %s",
								p.TradeID, p.SellerID, p.Quantity, p.AssetID, p.Price)
							buyerAudit.Details = &bDetails
							_, _ = auditRepo.Create(ctx, nil, &buyerAudit)

							sUserID := p.SellerID
							sellerAudit := audit.AuditLog{
								UserID: &sUserID,
								Action: "TRADE_EXECUTED",
							}
							sDetails := fmt.Sprintf("Matched trade #%d as seller against buyer #%d for quantity %s of asset #%d at price %s",
								p.TradeID, p.BuyerID, p.Quantity, p.AssetID, p.Price)
							sellerAudit.Details = &sDetails
							_, _ = auditRepo.Create(ctx, nil, &sellerAudit)
							continue
						}

					case "wallet.updated":
						var p WalletEventPayload
						if err := json.Unmarshal(we.Payload, &p); err == nil {
							userID := p.UserID
							a.UserID = &userID
							a.Action = "WALLET_UPDATED"
							details = fmt.Sprintf("Wallet #%d balance changed by %s due to %s. New balance: %s",
								p.WalletID, p.ChangeAmount, p.TransactionType, p.Balance)
							a.Details = &details
						}
					default:
						continue
					}

					if a.Action != "" {
						_, err := auditRepo.Create(ctx, nil, &a)
						if err != nil {
							log.Error().Err(err).Str("action", a.Action).Msg("Audit consumer: failed to write log to database")
						}
					}
				}
			}
		}(topic)
	}
}

func StartAnalyticsConsumer(ctx context.Context, brokers []string, topic string, redisClient *redis.Client) {
	go func() {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  "analytics-group",
			MaxBytes: 10e6,
		})
		defer r.Close()

		log.Info().Str("topic", topic).Msg("Analytics consumer started")

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
					log.Error().Err(err).Str("topic", topic).Msg("Analytics consumer read error")
					time.Sleep(1 * time.Second)
					continue
				}

				var we WrappedEvent
				if err := json.Unmarshal(m.Value, &we); err != nil {
					log.Warn().Err(err).Msg("Analytics consumer: failed to unmarshal event wrapper")
					continue
				}

				if we.Type == "trade.executed" {
					var p TradeEventPayload
					if err := json.Unmarshal(we.Payload, &p); err == nil {
						if redisClient != nil {
							redisClient.RecordAnalyticsTrade(ctx, p.Price, p.Quantity)
						}
					}
				}
			}
		}
	}()
}
