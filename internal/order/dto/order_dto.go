package dto

import "time"

type PlaceOrderInput struct {
	AssetID        int64   `json:"asset_id"         binding:"required,min=1"`
	Side           string  `json:"side"             binding:"required,oneof=BUY SELL"`
	Price          string  `json:"price"            binding:"required"`
	Quantity       string  `json:"quantity"         binding:"required"`
	IdempotencyKey *string `json:"idempotency_key"`
}

type OrderDTO struct {
	ID                int64     `json:"id"`
	UserID            int64     `json:"user_id"`
	AssetID           int64     `json:"asset_id"`
	Side              string    `json:"side"`
	Type              string    `json:"type"`
	Price             string    `json:"price"`
	Quantity          string    `json:"quantity"`
	FilledQuantity    string    `json:"filled_quantity"`
	RemainingQuantity string    `json:"remaining_quantity"`
	Status            string    `json:"status"`
	IdempotencyKey    *string   `json:"idempotency_key,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type OrderPage struct {
	Data       []OrderDTO `json:"data"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}
