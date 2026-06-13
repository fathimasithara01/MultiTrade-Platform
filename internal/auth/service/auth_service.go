package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	authdto "github.com/fathimasithara01/multitrade-platform/internal/auth/dto"
	"github.com/fathimasithara01/multitrade-platform/internal/user"
	userrepo "github.com/fathimasithara01/multitrade-platform/internal/user/repository"
	"github.com/fathimasithara01/multitrade-platform/pkg/jwt"
	"github.com/fathimasithara01/multitrade-platform/pkg/password"
)

var (
	ErrInvalidRole        = errors.New("invalid role")
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountSuspended   = errors.New("account is suspended")
	ErrTokenInvalid       = errors.New("token is invalid")
)

type WalletCreator interface {
	CreateWalletForUser(ctx context.Context, userID int64) error
}

var validRoles = map[string]struct{}{
	user.RoleAdmin:   {},
	user.RoleBroker:  {},
	user.RoleTrader:  {},
	user.RoleSupport: {},
}

type AuthService struct {
	userRepo      userrepo.UserRepository
	jwtService    *jwt.JWTService
	walletCreator WalletCreator
}

func NewAuthService(userRepo userrepo.UserRepository, jwtService *jwt.JWTService) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (s *AuthService) SetWalletCreator(wc WalletCreator) {
	s.walletCreator = wc
}

func (s *AuthService) Register(ctx context.Context, input authdto.RegisterRequest) (*user.User, *jwt.TokenPair, error) {
	if _, ok := validRoles[input.Role]; !ok {
		return nil, nil, ErrInvalidRole
	}

	_, err := s.userRepo.GetUserByEmail(ctx, input.Email)
	if err == nil {
		return nil, nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, userrepo.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("register: lookup user: %w", err)
	}

	hash, err := password.Hash(input.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("register: hash password: %w", err)
	}

	u := &user.User{
		Email:        input.Email,
		PasswordHash: hash,
		Role:         input.Role,
	}

	created, err := s.userRepo.CreateUser(ctx, u)
	if err != nil {
		return nil, nil, fmt.Errorf("register: create user: %w", err)
	}

	tokens, err := s.jwtService.GenerateTokenPair(created.ID, created.Role)
	if err != nil {
		return nil, nil, fmt.Errorf("register: generate tokens: %w", err)
	}

	if s.walletCreator != nil {
		if wErr := s.walletCreator.CreateWalletForUser(ctx, created.ID); wErr != nil {
			log.Error().Err(wErr).Int64("user_id", created.ID).Msg("auto-create wallet failed")
		}
	}

	log.Info().Int64("user_id", created.ID).Str("role", created.Role).Msg("New user registered")
	return created, tokens, nil
}

func (s *AuthService) Login(ctx context.Context, input authdto.LoginRequest) (*user.User, *jwt.TokenPair, error) {
	u, err := s.userRepo.GetUserByEmail(ctx, input.Email)
	if errors.Is(err, userrepo.ErrUserNotFound) {
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, fmt.Errorf("login: lookup user: %w", err)
	}

	if err := password.Verify(u.PasswordHash, input.Password); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	if u.Status == user.UserStatusSuspended {
		return nil, nil, ErrAccountSuspended
	}

	tokens, err := s.jwtService.GenerateTokenPair(u.ID, u.Role)
	if err != nil {
		return nil, nil, fmt.Errorf("login: generate tokens: %w", err)
	}

	log.Info().Int64("user_id", u.ID).Str("role", u.Role).Msg("User logged in")
	return u, tokens, nil
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshTokenStr string) (*jwt.TokenPair, error) {
	claims, err := s.jwtService.ParseToken(refreshTokenStr)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != jwt.TokenTypeRefresh {
		return nil, ErrTokenInvalid
	}

	u, err := s.userRepo.GetUserByID(ctx, claims.UserID)
	if errors.Is(err, userrepo.ErrUserNotFound) {
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("refresh: lookup user: %w", err)
	}

	tokens, err := s.jwtService.GenerateTokenPair(u.ID, u.Role)
	if err != nil {
		return nil, fmt.Errorf("refresh: generate tokens: %w", err)
	}

	return tokens, nil
}
