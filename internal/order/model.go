package order

import "time"

const (
	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"

	OrderTypeLimit = "LIMIT"

	OrderStatusPending         = "PENDING"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled          = "FILLED"
	OrderStatusCancelled       = "CANCELLED"
)

type Order struct {
	ID                int64      `db:"id"                 json:"id"`
	UserID            int64      `db:"user_id"            json:"user_id"`
	AssetID           int64      `db:"asset_id"           json:"asset_id"`
	Side              string     `db:"side"               json:"side"`
	Type              string     `db:"type"               json:"type"`
	Price             string     `db:"price"              json:"price"`
	Quantity          string     `db:"quantity"           json:"quantity"`
	FilledQuantity    string     `db:"filled_quantity"    json:"filled_quantity"`
	RemainingQuantity string     `db:"remaining_quantity" json:"remaining_quantity"`
	Status            string     `db:"status"             json:"status"`
	IdempotencyKey    *string    `db:"idempotency_key"    json:"idempotency_key,omitempty"`
	CreatedAt         time.Time  `db:"created_at"         json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"         json:"updated_at"`
}
