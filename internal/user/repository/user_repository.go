package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/fathimasithara01/multitrade-platform/internal/user"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	CreateUser(ctx context.Context, u *user.User) (*user.User, error)
	GetUserByEmail(ctx context.Context, email string) (*user.User, error)
	GetUserByID(ctx context.Context, id int64) (*user.User, error)

	// Admin operations
	ListUsers(ctx context.Context, role, status string, limit, offset int) ([]user.User, error)
	CountUsers(ctx context.Context, role, status string) (int, error)
	UpdateUserStatus(ctx context.Context, id int64, status string) (*user.User, error)
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, u *user.User) (*user.User, error) {
	query := `
		INSERT INTO users (email, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'ACTIVE', NOW(), NOW())
		RETURNING id, email, password_hash, role, status, created_at, updated_at, deleted_at`

	created := &user.User{}
	err := r.db.QueryRowxContext(ctx, query,
		u.Email,
		u.PasswordHash,
		u.Role,
	).StructScan(created)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT id, email, password_hash, role, status, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	u := &user.User{}
	err := r.db.QueryRowxContext(ctx, query, email).StructScan(u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id int64) (*user.User, error) {
	query := `
		SELECT id, email, password_hash, role, status, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	u := &user.User{}
	err := r.db.QueryRowxContext(ctx, query, id).StructScan(u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *userRepository) ListUsers(ctx context.Context, role, status string, limit, offset int) ([]user.User, error) {
	query := `
		SELECT id, email, password_hash, role, status, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR role   = $1)
		  AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	var users []user.User
	if err := r.db.SelectContext(ctx, &users, query, role, status, limit, offset); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (r *userRepository) CountUsers(ctx context.Context, role, status string) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx, `
		SELECT COUNT(*) FROM users
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR role   = $1)
		  AND ($2 = '' OR status = $2)`,
		role, status,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *userRepository) UpdateUserStatus(ctx context.Context, id int64, status string) (*user.User, error) {
	query := `
		UPDATE users SET status = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, email, password_hash, role, status, created_at, updated_at, deleted_at`

	u := &user.User{}
	err := r.db.QueryRowxContext(ctx, query, status, id).StructScan(u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update user status: %w", err)
	}
	return u, nil
}
