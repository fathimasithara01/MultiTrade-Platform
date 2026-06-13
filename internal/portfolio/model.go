package portfolio

type PortfolioHolding struct {
	UserID          int64  `db:"user_id"`
	AssetID         int64  `db:"asset_id"`
	Quantity        string `db:"quantity"`
	AverageBuyPrice string `db:"average_buy_price"`
}
