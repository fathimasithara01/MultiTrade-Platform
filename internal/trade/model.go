package trade

import "time"

type Trade struct {
	ID          int64     `db:"id"            json:"id"`
	BuyOrderID  int64     `db:"buy_order_id"  json:"buy_order_id"`
	SellOrderID int64     `db:"sell_order_id" json:"sell_order_id"`
	AssetID     int64     `db:"asset_id"      json:"asset_id"`
	Price       string    `db:"price"         json:"price"`
	Quantity    string    `db:"quantity"      json:"quantity"`
	BuyerID     int64     `db:"buyer_id"      json:"buyer_id"`
	SellerID    int64     `db:"seller_id"     json:"seller_id"`
	CreatedAt   time.Time `db:"created_at"    json:"created_at"`
}
