package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/audit"
	auditrepo "github.com/fathimasithara01/multitrade-platform/internal/audit/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/redis"
	"github.com/fathimasithara01/multitrade-platform/internal/user"
	userrepo "github.com/fathimasithara01/multitrade-platform/internal/user/repository"
)

var ErrInvalidStatus = errors.New("status must be ACTIVE or SUSPENDED")

type UserPage struct {
	Data       []user.User `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type VolumePoint struct {
	Period string `json:"period" db:"period"`
	Trades int    `json:"trades" db:"trades"`
	Volume string `json:"volume" db:"volume"`
}

type SuspiciousUser struct {
	UserID     int64  `json:"user_id"     db:"user_id"`
	Email      string `json:"email"       db:"email"`
	TradeCount int    `json:"trade_count" db:"trade_count"`
}

type SystemHealth struct {
	Database string `json:"database"`
	Redis    string `json:"redis"`
	Kafka    string `json:"kafka"`
}

type AdminService struct {
	db           *sqlx.DB
	userRepo     userrepo.UserRepository
	auditRepo    auditrepo.AuditRepository
	redisClient  *redis.Client
	kafkaBrokers []string
}

func NewAdminService(
	db *sqlx.DB,
	userRepo userrepo.UserRepository,
	auditRepo auditrepo.AuditRepository,
	redisClient *redis.Client,
	kafkaBrokers []string,
) *AdminService {
	return &AdminService{
		db:           db,
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		redisClient:  redisClient,
		kafkaBrokers: kafkaBrokers,
	}
}

func (s *AdminService) ListUsers(ctx context.Context, role, status string, page, pageSize int) (*UserPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	users, err := s.userRepo.ListUsers(ctx, role, status, pageSize, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.userRepo.CountUsers(ctx, role, status)
	if err != nil {
		return nil, err
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return &UserPage{
		Data:       users,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *AdminService) UpdateUserStatus(ctx context.Context, adminUserID, targetUserID int64, status string) (*user.User, error) {
	if status != user.UserStatusActive && status != user.UserStatusSuspended {
		return nil, ErrInvalidStatus
	}

	u, err := s.userRepo.UpdateUserStatus(ctx, targetUserID, status)
	if errors.Is(err, userrepo.ErrUserNotFound) {
		return nil, userrepo.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update user status: %w", err)
	}

	action := "USER_ACTIVATED"
	if status == user.UserStatusSuspended {
		action = "USER_SUSPENDED"
	}
	details := fmt.Sprintf("Admin #%d set user #%d status to %s", adminUserID, targetUserID, status)
	if _, auditErr := s.auditRepo.Create(ctx, nil, &audit.AuditLog{
		UserID:  &adminUserID,
		Action:  action,
		Details: &details,
	}); auditErr != nil {
		log.Error().Err(auditErr).Msg("admin: write audit log failed")
	}

	log.Info().Int64("admin_id", adminUserID).Int64("target_id", targetUserID).Str("status", status).Msg("User status updated")
	return u, nil
}

func (s *AdminService) VolumeAnalytics(ctx context.Context, hours int) ([]VolumePoint, error) {
	if hours <= 0 || hours > 720 {
		hours = 24
	}

	query := `
		SELECT
			TO_CHAR(DATE_TRUNC('hour', created_at), 'YYYY-MM-DD HH24:00') AS period,
			COUNT(*)                                                        AS trades,
			COALESCE(SUM(price * quantity), 0)::TEXT                       AS volume
		FROM trades
		WHERE created_at >= NOW() - ($1 * INTERVAL '1 hour')
		GROUP BY DATE_TRUNC('hour', created_at)
		ORDER BY DATE_TRUNC('hour', created_at) ASC`

	var points []VolumePoint
	if err := s.db.SelectContext(ctx, &points, query, hours); err != nil {
		return nil, fmt.Errorf("volume analytics: %w", err)
	}
	return points, nil
}

func (s *AdminService) SuspiciousUsers(ctx context.Context, windowHours int, threshold int) ([]SuspiciousUser, error) {
	if windowHours <= 0 {
		windowHours = 1
	}
	if threshold <= 0 {
		threshold = 10
	}

	query := `
		SELECT
			u.id   AS user_id,
			u.email,
			COUNT(t.id) AS trade_count
		FROM users u
		JOIN (
			SELECT buyer_id  AS user_id, id FROM trades WHERE created_at >= NOW() - ($1 * INTERVAL '1 hour')
			UNION ALL
			SELECT seller_id AS user_id, id FROM trades WHERE created_at >= NOW() - ($1 * INTERVAL '1 hour')
		) t ON u.id = t.user_id
		GROUP BY u.id, u.email
		HAVING COUNT(t.id) > $2
		ORDER BY trade_count DESC`

	var suspects []SuspiciousUser
	if err := s.db.SelectContext(ctx, &suspects, query, windowHours, threshold); err != nil {
		return nil, fmt.Errorf("suspicious users: %w", err)
	}
	return suspects, nil
}

func (s *AdminService) ListAuditLogs(ctx context.Context, userID *int64, action string, page, pageSize int) ([]audit.AuditLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	logs, err := s.auditRepo.List(ctx, userID, action, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.auditRepo.Count(ctx, userID, action)
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (s *AdminService) Health(ctx context.Context) *SystemHealth {
	h := &SystemHealth{}

	ctxDB, cancelDB := context.WithTimeout(ctx, 2*time.Second)
	defer cancelDB()
	if err := s.db.PingContext(ctxDB); err != nil {
		h.Database = "DOWN"
	} else {
		h.Database = "UP"
	}

	if s.redisClient != nil {
		ctxR, cancelR := context.WithTimeout(ctx, 2*time.Second)
		defer cancelR()
		if err := s.redisClient.Ping(ctxR); err != nil {
			h.Redis = "DOWN"
		} else {
			h.Redis = "UP"
		}
	} else {
		h.Redis = "DISABLED"
	}

	kafkaOK := true
	for _, b := range s.kafkaBrokers {
		conn, err := net.DialTimeout("tcp", b, 2*time.Second)
		if err != nil {
			kafkaOK = false
			break
		}
		conn.Close()
	}
	if len(s.kafkaBrokers) == 0 {
		h.Kafka = "DISABLED"
	} else if kafkaOK {
		h.Kafka = "UP"
	} else {
		h.Kafka = "DOWN"
	}

	return h
}
