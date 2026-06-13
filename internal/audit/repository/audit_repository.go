package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/fathimasithara01/multitrade-platform/internal/audit"
)

type AuditRepository interface {
	Create(ctx context.Context, tx *sqlx.Tx, log *audit.AuditLog) (*audit.AuditLog, error)
	List(ctx context.Context, userID *int64, action string, limit, offset int) ([]audit.AuditLog, error)
	Count(ctx context.Context, userID *int64, action string) (int, error)
}

type auditRepository struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, tx *sqlx.Tx, logEntry *audit.AuditLog) (*audit.AuditLog, error) {
	query := `
		INSERT INTO audit_logs (user_id, action, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, user_id, action, details, ip_address, created_at`

	created := &audit.AuditLog{}
	var err error
	if tx != nil {
		err = tx.QueryRowxContext(ctx, query,
			logEntry.UserID, logEntry.Action, logEntry.Details, logEntry.IPAddress,
		).StructScan(created)
	} else {
		err = r.db.QueryRowxContext(ctx, query,
			logEntry.UserID, logEntry.Action, logEntry.Details, logEntry.IPAddress,
		).StructScan(created)
	}
	if err != nil {
		return nil, fmt.Errorf("create audit log: %w", err)
	}
	return created, nil
}

func (r *auditRepository) List(ctx context.Context, userID *int64, action string, limit, offset int) ([]audit.AuditLog, error) {
	query := `
		SELECT id, user_id, action, details, ip_address, created_at
		FROM audit_logs
		WHERE ($1::BIGINT IS NULL OR user_id = $1)
		  AND ($2 = '' OR action = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	var logs []audit.AuditLog
	if err := r.db.SelectContext(ctx, &logs, query, userID, action, limit, offset); err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, nil
}

func (r *auditRepository) Count(ctx context.Context, userID *int64, action string) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx, `
		SELECT COUNT(*) FROM audit_logs
		WHERE ($1::BIGINT IS NULL OR user_id = $1)
		  AND ($2 = '' OR action = $2)`,
		userID, action,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit logs: %w", err)
	}
	return count, nil
}
