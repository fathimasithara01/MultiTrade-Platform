package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authdto "github.com/fathimasithara01/multitrade-platform/internal/auth/dto"
	"github.com/fathimasithara01/multitrade-platform/internal/auth/service"
	"github.com/fathimasithara01/multitrade-platform/internal/user"
	"github.com/fathimasithara01/multitrade-platform/internal/user/repository"
	"github.com/fathimasithara01/multitrade-platform/pkg/jwt"
)

type stubUserRepo struct {
	users  map[string]*user.User
	byID   map[int64]*user.User
	nextID int64
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{
		users: make(map[string]*user.User),
		byID:  make(map[int64]*user.User),
	}
}

func (r *stubUserRepo) CreateUser(_ context.Context, u *user.User) (*user.User, error) {
	r.nextID++
	u.ID = r.nextID
	u.Status = user.UserStatusActive
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	r.users[u.Email] = u
	r.byID[u.ID] = u
	return u, nil
}

func (r *stubUserRepo) GetUserByEmail(_ context.Context, email string) (*user.User, error) {
	u, ok := r.users[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return u, nil
}

func (r *stubUserRepo) GetUserByID(_ context.Context, id int64) (*user.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return u, nil
}

func (r *stubUserRepo) ListUsers(_ context.Context, _, _ string, _, _ int) ([]user.User, error) {
	return nil, nil
}

func (r *stubUserRepo) CountUsers(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}

func (r *stubUserRepo) UpdateUserStatus(_ context.Context, id int64, status string) (*user.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	u.Status = status
	return u, nil
}

type stubWalletCreator struct {
	created []int64
}

func (w *stubWalletCreator) CreateWalletForUser(_ context.Context, userID int64) error {
	w.created = append(w.created, userID)
	return nil
}

func newAuthService(repo repository.UserRepository) *service.AuthService {
	jwtSvc := jwt.NewJWTService("test-secret-key-min32chars!!!!", 15*time.Minute, 168*time.Hour)
	svc := service.NewAuthService(repo, jwtSvc)
	return svc
}

func TestRegister_Success(t *testing.T) {
	repo := newStubUserRepo()
	wc := &stubWalletCreator{}
	svc := newAuthService(repo)
	svc.SetWalletCreator(wc)

	u, tokens, err := svc.Register(context.Background(), authdto.RegisterRequest{
		Email:    "alice@test.com",
		Password: "password123",
		Role:     user.RoleTrader,
	})

	require.NoError(t, err)
	assert.Equal(t, "alice@test.com", u.Email)
	assert.Equal(t, user.RoleTrader, u.Role)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, []int64{u.ID}, wc.created)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newStubUserRepo()
	svc := newAuthService(repo)

	input := authdto.RegisterRequest{
		Email:    "bob@test.com",
		Password: "password123",
		Role:     user.RoleTrader,
	}
	_, _, err := svc.Register(context.Background(), input)
	require.NoError(t, err)

	_, _, err = svc.Register(context.Background(), input)
	assert.True(t, errors.Is(err, service.ErrEmailAlreadyExists))
}

func TestRegister_InvalidRole(t *testing.T) {
	repo := newStubUserRepo()
	svc := newAuthService(repo)

	_, _, err := svc.Register(context.Background(), authdto.RegisterRequest{
		Email:    "bad@test.com",
		Password: "password123",
		Role:     "superuser",
	})
	assert.True(t, errors.Is(err, service.ErrInvalidRole))
}

func TestLogin_Success(t *testing.T) {
	repo := newStubUserRepo()
	svc := newAuthService(repo)

	_, _, err := svc.Register(context.Background(), authdto.RegisterRequest{
		Email:    "carol@test.com",
		Password: "password123",
		Role:     user.RoleBroker,
	})
	require.NoError(t, err)

	u, tokens, err := svc.Login(context.Background(), authdto.LoginRequest{
		Email:    "carol@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "carol@test.com", u.Email)
	assert.NotEmpty(t, tokens.AccessToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newStubUserRepo()
	svc := newAuthService(repo)

	_, _, _ = svc.Register(context.Background(), authdto.RegisterRequest{
		Email:    "dave@test.com",
		Password: "correctpassword",
		Role:     user.RoleTrader,
	})

	_, _, err := svc.Login(context.Background(), authdto.LoginRequest{
		Email:    "dave@test.com",
		Password: "wrongpassword",
	})
	assert.True(t, errors.Is(err, service.ErrInvalidCredentials))
}

func TestLogin_SuspendedUser(t *testing.T) {
	repo := newStubUserRepo()
	svc := newAuthService(repo)

	created, _, _ := svc.Register(context.Background(), authdto.RegisterRequest{
		Email:    "eve@test.com",
		Password: "password123",
		Role:     user.RoleTrader,
	})

	repo.UpdateUserStatus(context.Background(), created.ID, user.UserStatusSuspended)

	_, _, err := svc.Login(context.Background(), authdto.LoginRequest{
		Email:    "eve@test.com",
		Password: "password123",
	})
	assert.True(t, errors.Is(err, service.ErrAccountSuspended))
}

func TestLogin_UnknownEmail(t *testing.T) {
	repo := newStubUserRepo()
	svc := newAuthService(repo)

	_, _, err := svc.Login(context.Background(), authdto.LoginRequest{
		Email:    "nobody@test.com",
		Password: "password123",
	})
	assert.True(t, errors.Is(err, service.ErrInvalidCredentials))
}

func TestJWT_ParseAccessToken(t *testing.T) {
	jwtSvc := jwt.NewJWTService("test-secret-key-min32chars!!!!", 15*time.Minute, 168*time.Hour)

	pair, err := jwtSvc.GenerateTokenPair(42, user.RoleAdmin)
	require.NoError(t, err)

	claims, err := jwtSvc.ParseToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, user.RoleAdmin, claims.Role)
	assert.Equal(t, jwt.TokenTypeAccess, claims.TokenType)
}

func TestJWT_RefreshTokenRejectedAsAccess(t *testing.T) {
	jwtSvc := jwt.NewJWTService("test-secret-key-min32chars!!!!", 15*time.Minute, 168*time.Hour)

	pair, _ := jwtSvc.GenerateTokenPair(1, user.RoleTrader)
	claims, err := jwtSvc.ParseToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, jwt.TokenTypeRefresh, claims.TokenType)
}

func TestRefreshTokens_Success(t *testing.T) {
	repo := newStubUserRepo()
	svc := newAuthService(repo)

	_, tokens, _ := svc.Register(context.Background(), authdto.RegisterRequest{
		Email:    "frank@test.com",
		Password: "password123",
		Role:     user.RoleTrader,
	})

	time.Sleep(1100 * time.Millisecond)

	newTokens, err := svc.RefreshTokens(context.Background(), tokens.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	assert.NotEqual(t, tokens.AccessToken, newTokens.AccessToken)
}
