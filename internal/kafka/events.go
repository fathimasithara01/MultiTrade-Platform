package kafka

import "time"

// Event represents a generic wrapper for TradeVerse events.
type Event struct {
	Type      string      `json:"type"`      // order.created, order.cancelled, trade.executed, wallet.updated
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// OrderEventPayload contains details for order placement/cancellation events.
type OrderEventPayload struct {
	OrderID   int64     `json:"order_id"`
	UserID    int64     `json:"user_id"`
	AssetID   int64     `json:"asset_id"`
	Side      string    `json:"side"`      // BUY, SELL
	Type      string    `json:"type"`      // LIMIT
	Price     string    `json:"price"`     // decimal string
	Quantity  string    `json:"quantity"`  // decimal string
	Status    string    `json:"status"`    // PENDING, PARTIALLY_FILLED, FILLED, CANCELLED
	Timestamp time.Time `json:"timestamp"`
}

// TradeEventPayload contains details for trade execution.
type TradeEventPayload struct {
	TradeID     int64     `json:"trade_id"`
	BuyOrderID  int64     `json:"buy_order_id"`
	SellOrderID int64     `json:"sell_order_id"`
	AssetID     int64     `json:"asset_id"`
	Price       string    `json:"price"`
	Quantity    string    `json:"quantity"`
	BuyerID     int64     `json:"buyer_id"`
	SellerID    int64     `json:"seller_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// WalletEventPayload contains details for wallet updates.
type WalletEventPayload struct {
	WalletID        int64     `json:"wallet_id"`
	UserID          int64     `json:"user_id"`
	Balance         string    `json:"balance"`
	ChangeAmount    string    `json:"change_amount"`
	TransactionType string    `json:"transaction_type"` // deposit, withdrawal, trade_debit, trade_credit
	Timestamp       time.Time `json:"timestamp"`
}
