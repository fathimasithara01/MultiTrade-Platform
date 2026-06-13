package enums

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleBroker  Role = "broker"
	RoleTrader  Role = "trader"
	RoleSupport Role = "support"
)

type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "PENDING"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCancelled       OrderStatus = "CANCELLED"
)

type TxType string

const (
	TxTypeDeposit    TxType = "DEPOSIT"
	TxTypeWithdrawal TxType = "WITHDRAWAL"
)
