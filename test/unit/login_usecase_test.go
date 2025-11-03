package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/vobe/auth-service/application/port/inbound"
	"github.com/vobe/auth-service/application/port/outbound"
	"github.com/vobe/auth-service/application/usecase"
	"github.com/vobe/auth-service/domain/entity"
)

// Mock implementations
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

// Implement missing method to satisfy outbound.UserRepository
func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

type MockRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockRefreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) RevokeByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateAccessToken(claims outbound.TokenClaims) (string, error) {
	args := m.Called(claims)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) GenerateRefreshToken() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidateAccessToken(token string) (*outbound.TokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.TokenClaims), args.Error(1)
}

type MockPasswordService struct {
	mock.Mock
}

func (m *MockPasswordService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordService) ComparePassword(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

func TestLoginUseCase_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	mockTokenService := new(MockTokenService)
	mockPasswordService := new(MockPasswordService)
	
	user := &entity.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Password: "hashed-password",
	}
	
	req := inbound.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	
	mockUserRepo.On("FindByEmail", ctx, "test@example.com").Return(user, nil)
	mockPasswordService.On("ComparePassword", "hashed-password", "password123").Return(nil)
	mockTokenService.On("GenerateAccessToken", outbound.TokenClaims{
		UserID: "user-123",
		Email:  "test@example.com",
	}).Return("access-token", nil)
	mockTokenService.On("GenerateRefreshToken").Return("refresh-token", nil)
	mockRefreshTokenRepo.On("Create", ctx, mock.AnythingOfType("*entity.RefreshToken")).Return(nil)
	
	useCase := usecase.NewLoginUseCase(
		mockUserRepo,
		mockRefreshTokenRepo,
		mockTokenService,
		mockPasswordService,
		1*time.Hour,
		7*24*time.Hour,
	)
	
	// Act
	resp, err := useCase.Login(ctx, req)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
	assert.Equal(t, 3600, resp.ExpiresIn) // 1 hour in seconds
	
	mockUserRepo.AssertExpectations(t)
	mockRefreshTokenRepo.AssertExpectations(t)
	mockTokenService.AssertExpectations(t)
	mockPasswordService.AssertExpectations(t)
}

func TestLoginUseCase_InvalidEmailFormat(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	mockTokenService := new(MockTokenService)
	mockPasswordService := new(MockPasswordService)
	
	req := inbound.LoginRequest{
		Email:    "invalid-email",
		Password: "password123",
	}
	
	useCase := usecase.NewLoginUseCase(
		mockUserRepo,
		mockRefreshTokenRepo,
		mockTokenService,
		mockPasswordService,
		1*time.Hour,
		7*24*time.Hour,
	)
	
	// Act
	resp, err := useCase.Login(ctx, req)
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid email format")
}

func TestLoginUseCase_PasswordTooShort(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	mockTokenService := new(MockTokenService)
	mockPasswordService := new(MockPasswordService)
	
	req := inbound.LoginRequest{
		Email:    "test@example.com",
		Password: "short",
	}
	
	useCase := usecase.NewLoginUseCase(
		mockUserRepo,
		mockRefreshTokenRepo,
		mockTokenService,
		mockPasswordService,
		1*time.Hour,
		7*24*time.Hour,
	)
	
	// Act
	resp, err := useCase.Login(ctx, req)
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "password must be at least 8 characters")
}

func TestLoginUseCase_UserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	mockTokenService := new(MockTokenService)
	mockPasswordService := new(MockPasswordService)
	
	req := inbound.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	
	mockUserRepo.On("FindByEmail", ctx, "test@example.com").Return(nil, nil)
	
	useCase := usecase.NewLoginUseCase(
		mockUserRepo,
		mockRefreshTokenRepo,
		mockTokenService,
		mockPasswordService,
		1*time.Hour,
		7*24*time.Hour,
	)
	
	// Act
	resp, err := useCase.Login(ctx, req)
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, usecase.ErrInvalidCredentials, err)
	
	mockUserRepo.AssertExpectations(t)
}

func TestLoginUseCase_InvalidPassword(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	mockTokenService := new(MockTokenService)
	mockPasswordService := new(MockPasswordService)
	
	user := &entity.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Password: "hashed-password",
	}
	
	req := inbound.LoginRequest{
		Email:    "test@example.com",
		Password: "wrong-password",
	}
	
	mockUserRepo.On("FindByEmail", ctx, "test@example.com").Return(user, nil)
	mockPasswordService.On("ComparePassword", "hashed-password", "wrong-password").Return(errors.New("password mismatch"))
	
	useCase := usecase.NewLoginUseCase(
		mockUserRepo,
		mockRefreshTokenRepo,
		mockTokenService,
		mockPasswordService,
		1*time.Hour,
		7*24*time.Hour,
	)
	
	// Act
	resp, err := useCase.Login(ctx, req)
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, usecase.ErrInvalidCredentials, err)
	
	mockUserRepo.AssertExpectations(t)
	mockPasswordService.AssertExpectations(t)
}

func TestLoginUseCase_TokenGenerationFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	mockTokenService := new(MockTokenService)
	mockPasswordService := new(MockPasswordService)
	
	user := &entity.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Password: "hashed-password",
	}
	
	req := inbound.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	
	mockUserRepo.On("FindByEmail", ctx, "test@example.com").Return(user, nil)
	mockPasswordService.On("ComparePassword", "hashed-password", "password123").Return(nil)
	mockTokenService.On("GenerateAccessToken", outbound.TokenClaims{
		UserID: "user-123",
		Email:  "test@example.com",
	}).Return("", errors.New("token generation failed"))
	
	useCase := usecase.NewLoginUseCase(
		mockUserRepo,
		mockRefreshTokenRepo,
		mockTokenService,
		mockPasswordService,
		1*time.Hour,
		7*24*time.Hour,
	)
	
	// Act
	resp, err := useCase.Login(ctx, req)
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "token generation failed")
	
	mockUserRepo.AssertExpectations(t)
	mockPasswordService.AssertExpectations(t)
	mockTokenService.AssertExpectations(t)
}