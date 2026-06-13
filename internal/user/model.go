package user

import "time"

const (
	RoleAdmin   = "admin"
	RoleBroker  = "broker"
	RoleTrader  = "trader"
	RoleSupport = "support"

	UserStatusActive    = "ACTIVE"
	UserStatusSuspended = "SUSPENDED"
)

type User struct {
	ID           int64      `db:"id"           json:"id"`
	Email        string     `db:"email"        json:"email"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Role         string     `db:"role"         json:"role"`
	Status       string     `db:"status"       json:"status"`
	CreatedAt    time.Time  `db:"created_at"   json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"   json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"   json:"deleted_at,omitempty"`
}
