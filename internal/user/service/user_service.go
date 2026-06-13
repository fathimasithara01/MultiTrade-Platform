package service

import (
	"context"

	"github.com/fathimasithara01/multitrade-platform/internal/user"
	userrepo "github.com/fathimasithara01/multitrade-platform/internal/user/repository"
)

type UserService interface {
	GetUserByID(ctx context.Context, id int64) (*user.User, error)
}

type userService struct {
	repo userrepo.UserRepository
}

func NewUserService(repo userrepo.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) GetUserByID(ctx context.Context, id int64) (*user.User, error) {
	return s.repo.GetUserByID(ctx, id)
}
