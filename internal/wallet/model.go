package wallet

import "time"

const (
	TxTypeDeposit    = "DEPOSIT"
	TxTypeWithdrawal = "WITHDRAWAL"

	TxStatusCompleted = "COMPLETED"
	TxStatusFailed    = "FAILED"
)

type Wallet struct {
	ID        int64     `db:"id"         json:"id"`
	UserID    int64     `db:"user_id"    json:"user_id"`
	Balance   string    `db:"balance"    json:"balance"`
	Currency  string    `db:"currency"   json:"currency"`
	Version   int64     `db:"version"    json:"version"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}


