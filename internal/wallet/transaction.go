package wallet

import "time"

type WalletTransaction struct {
	ID              int64     `db:"id"               json:"id"`
	WalletID        int64     `db:"wallet_id"        json:"wallet_id"`
	Amount          string    `db:"amount"           json:"amount"`
	TransactionType string    `db:"transaction_type" json:"transaction_type"`
	Status          string    `db:"status"           json:"status"`
	ReferenceID     *string   `db:"reference_id"     json:"reference_id,omitempty"`
	Description     *string   `db:"description"      json:"description,omitempty"`
	CreatedAt       time.Time `db:"created_at"       json:"created_at"`
}
