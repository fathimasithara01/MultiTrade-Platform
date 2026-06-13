package dto

import "time"

type AmountInput struct {
	Amount      string  `json:"amount"      binding:"required"`
	Description *string `json:"description"`
}

type WalletTransactionDTO struct {
	ID              int64     `json:"id"`
	WalletID        int64     `json:"wallet_id"`
	Amount          string    `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	Status          string    `json:"status"`
	ReferenceID     *string   `json:"reference_id,omitempty"`
	Description     *string   `json:"description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type TransactionPage struct {
	Data       []WalletTransactionDTO `json:"data"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}
