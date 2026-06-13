package dto

type PortfolioResponse struct {
	UserID          int64  `json:"user_id"`
	AssetID         int64  `json:"asset_id"`
	Quantity        string `json:"quantity"`
	AverageBuyPrice string `json:"average_buy_price"`
}
