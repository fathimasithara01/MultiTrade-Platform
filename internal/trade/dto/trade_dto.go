package dto

import "time"

type TradeResponse struct {
	ID          int64     `json:"id"`
	BuyOrderID  int64     `json:"buy_order_id"`
	SellOrderID int64     `json:"sell_order_id"`
	AssetID     int64     `json:"asset_id"`
	Price       string    `json:"price"`
	Quantity    string    `json:"quantity"`
	BuyerID     int64     `json:"buyer_id"`
	SellerID    int64     `json:"seller_id"`
	CreatedAt   time.Time `json:"created_at"`
}
