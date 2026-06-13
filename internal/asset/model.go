package asset

import "time"

const (
	AssetStatusActive   = "ACTIVE"
	AssetStatusDisabled = "DISABLED"
)

type Asset struct {
	ID          int64     `db:"id"          json:"id"`
	BrokerID    int64     `db:"broker_id"   json:"broker_id"`
	Symbol      string    `db:"symbol"      json:"symbol"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	Price       string    `db:"price"       json:"price"`
	Quantity    string    `db:"quantity"    json:"quantity"`
	Status      string    `db:"status"      json:"status"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"  json:"updated_at"`
}
