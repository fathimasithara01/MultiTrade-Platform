package audit

import "time"

type AuditLog struct {
	ID        int64     `db:"id" json:"id"`
	UserID    *int64    `db:"user_id" json:"user_id"`
	Action    string    `db:"action" json:"action"`
	Details   *string   `db:"details" json:"details"`
	IPAddress *string   `db:"ip_address" json:"ip_address"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
