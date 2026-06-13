package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/fathimasithara01/multitrade-platform/internal/user"
	userrepo "github.com/fathimasithara01/multitrade-platform/internal/user/repository"
)

type AuthRepository interface {
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByID(ctx context.Context, id int64) (*user.User, error)
}

type authRepository struct {
	userRepo userrepo.UserRepository
}

func NewAuthRepository(db *sqlx.DB) AuthRepository {
	return &authRepository{
		userRepo: userrepo.NewUserRepository(db),
	}
}

func (r *authRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.userRepo.GetUserByEmail(ctx, email)
}

func (r *authRepository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	return r.userRepo.GetUserByID(ctx, id)
}
