package constants

const (
	// Pagination constants
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100

	RoleAdmin   = "admin"
	RoleBroker  = "broker"
	RoleTrader  = "trader"
	RoleSupport = "support"

	UserStatusActive    = "ACTIVE"
	UserStatusSuspended = "SUSPENDED"

	TxTypeDeposit     = "DEPOSIT"
	TxTypeWithdrawal  = "WITHDRAWAL"
	TxStatusCompleted = "COMPLETED"
	TxStatusFailed    = "FAILED"

	OrderSideBuy   = "BUY"
	OrderSideSell  = "SELL"
	OrderTypeLimit = "LIMIT"

	OrderStatusPending         = "PENDING"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled          = "FILLED"
	OrderStatusCancelled       = "CANCELLED"
)
