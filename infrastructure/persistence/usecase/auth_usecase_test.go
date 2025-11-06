package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/vobe/auth-service/application/port/inbound"
	"github.com/vobe/auth-service/application/port/outbound"
	"github.com/vobe/auth-service/domain/entity"
	"github.com/vobe/auth-service/infrastructure/service/logger"
)

// Mock implementations
type mockUserRepository struct {
	users map[string]*entity.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*entity.User),
	}
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, outbound.ErrUserNotFound
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, outbound.ErrUserNotFound
}

func (m *mockUserRepository) Create(ctx context.Context, user *entity.User) error {
	if _, exists := m.users[user.ID]; exists {
		return outbound.ErrUserAlreadyExists
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *entity.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return outbound.ErrUserNotFound
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) SoftDelete(ctx context.Context, id string) error {
	if _, exists := m.users[id]; !exists {
		return outbound.ErrUserNotFound
	}
	// For mock, just remove from the map to simulate soft delete
	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) FindAll(ctx context.Context, offset, limit int, filters outbound.UserFilters) ([]*entity.User, int, error) {
	var users []*entity.User
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, len(users), nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	for _, user := range m.users {
		if user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockUserRepository) FindByRole(ctx context.Context, role string) ([]*entity.User, error) {
	var users []*entity.User
	for _, user := range m.users {
		if user.Role == role {
			users = append(users, user)
		}
	}
	return users, nil
}

type mockRefreshTokenRepository struct {
	tokens map[string]*entity.RefreshToken
}

func newMockRefreshTokenRepository() *mockRefreshTokenRepository {
	return &mockRefreshTokenRepository{
		tokens: make(map[string]*entity.RefreshToken),
	}
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	if token == nil {
		return errors.New("token cannot be nil")
	}
	if _, exists := m.tokens[token.Token]; exists {
		return outbound.ErrRefreshTokenAlreadyExists
	}
	m.tokens[token.Token] = token
	return nil
}

func (m *mockRefreshTokenRepository) FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}
	if rt, exists := m.tokens[token]; exists {
		return rt, nil
	}
	return nil, outbound.ErrRefreshTokenNotFound
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	if rt, exists := m.tokens[token]; exists {
		now := time.Now()
		rt.RevokedAt = &now
		return nil
	}
	return outbound.ErrRefreshTokenNotFound
}

func (m *mockRefreshTokenRepository) RevokeByUserID(ctx context.Context, userID string) error {
	for _, rt := range m.tokens {
		if rt.UserID == userID {
			now := time.Now()
			rt.RevokedAt = &now
		}
	}
	return nil
}

type mockTokenService struct {
	accessTokenCounter  int
	refreshTokenCounter int
}

func (m *mockTokenService) GenerateAccessToken(claims outbound.TokenClaims) (string, error) {
	m.accessTokenCounter++
	return fmt.Sprintf("mock-access-token-%d", m.accessTokenCounter), nil
}

func (m *mockTokenService) GenerateRefreshToken() (string, error) {
	m.refreshTokenCounter++
	return fmt.Sprintf("mock-refresh-token-%d", m.refreshTokenCounter), nil
}

func (m *mockTokenService) ValidateAccessToken(token string) (*outbound.TokenClaims, error) {
	if token == "valid-token" {
		return &outbound.TokenClaims{UserID: "user123", Email: "test@example.com"}, nil
	}
	return nil, errors.New("invalid token")
}

type mockPasswordService struct{}

func (m *mockPasswordService) HashPassword(password string) (string, error) {
	return "hashed-" + password, nil
}

func (m *mockPasswordService) VerifyPassword(password, hash string) (bool, error) {
	return hash == "hashed-"+password, nil
}

// Additional mocks for new dependencies

type mockRecaptchaService struct{ enabled bool }

func (m *mockRecaptchaService) VerifyToken(ctx context.Context, token string) (bool, error) {
	return true, nil
}
func (m *mockRecaptchaService) IsEnabled() bool { return m.enabled }

type mockRateLimitService struct{}

func (m *mockRateLimitService) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	return true, nil
}
func (m *mockRateLimitService) Increment(ctx context.Context, key string, window time.Duration) error {
	return nil
}
func (m *mockRateLimitService) Block(ctx context.Context, key string, duration time.Duration, reason string) error {
	return nil
}
func (m *mockRateLimitService) IsBlocked(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (m *mockRateLimitService) GetAttempts(ctx context.Context, key string) (int, error) {
	return 0, nil
}

// Minimal no-op logger

type testLogger struct{}

func (l *testLogger) Info(ctx context.Context, message string, fields map[string]interface{}) {}
func (l *testLogger) Error(ctx context.Context, message string, err error, fields map[string]interface{}) {
}
func (l *testLogger) Warn(ctx context.Context, message string, fields map[string]interface{})  {}
func (l *testLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) {}
func (l *testLogger) WithFields(fields map[string]interface{}) logger.Logger                   { return l }

func TestAuthUseCase(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	tokenService := &mockTokenService{}
	passwordService := &mockPasswordService{}
	recaptchaService := &mockRecaptchaService{enabled: false}
	rateLimitService := &mockRateLimitService{}
	log := &testLogger{}

	authUseCase := NewAuthUseCase(
		userRepo,
		refreshTokenRepo,
		tokenService,
		passwordService,
		recaptchaService,
		rateLimitService,
		log,
		15*time.Minute,
		30*24*time.Hour,
	)

	// Create test user
	testUser := entity.NewUser("user123", "Test User", "test@example.com", "hashed-password123", "user", "active")
	if err := userRepo.Create(ctx, testUser); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("LoginSuccess", func(t *testing.T) {
		req := inbound.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		resp, err := authUseCase.Login(ctx, req)
		if err != nil {
			t.Errorf("Login should succeed: %v", err)
		}
		if resp.AccessToken == "" {
			t.Error("Access token should not be empty")
		}
		if resp.RefreshToken == "" {
			t.Error("Refresh token should not be empty")
		}
	})

	t.Run("LoginInvalidCredentials", func(t *testing.T) {
		req := inbound.LoginRequest{
			Email:    "test@example.com",
			Password: "wrong-password",
		}

		_, err := authUseCase.Login(ctx, req)
		if err == nil {
			t.Error("Login should fail with wrong password")
		}
	})

	t.Run("LoginNonExistentUser", func(t *testing.T) {
		req := inbound.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		_, err := authUseCase.Login(ctx, req)
		if err == nil {
			t.Error("Login should fail with non-existent user")
		}
	})

	t.Run("RefreshSuccess", func(t *testing.T) {
		// Create refresh token
		refreshToken := entity.NewRefreshToken("refresh123", "user123", "refresh-token123", time.Now().Add(24*time.Hour))
		if err := refreshTokenRepo.Create(ctx, refreshToken); err != nil {
			t.Fatalf("Failed to create refresh token: %v", err)
		}

		req := inbound.RefreshRequest{
			RefreshToken: "refresh-token123",
		}

		resp, err := authUseCase.Refresh(ctx, req)
		if err != nil {
			t.Errorf("Refresh should succeed: %v", err)
		}
		if resp.AccessToken == "" {
			t.Error("Access token should not be empty")
		}
	})

	t.Run("LogoutSuccess", func(t *testing.T) {
		// Create refresh token
		refreshToken := entity.NewRefreshToken("refresh456", "user123", "refresh-token456", time.Now().Add(24*time.Hour))
		if err := refreshTokenRepo.Create(ctx, refreshToken); err != nil {
			t.Fatalf("Failed to create refresh token: %v", err)
		}

		req := inbound.LogoutRequest{
			RefreshToken: "refresh-token456",
		}

		err := authUseCase.Logout(ctx, req)
		if err != nil {
			t.Errorf("Logout should succeed: %v", err)
		}
	})

	t.Run("MeSuccess", func(t *testing.T) {
		resp, err := authUseCase.Me(ctx, "user123")
		if err != nil {
			t.Errorf("Me should succeed: %v", err)
		}
		if resp.ID != "user123" {
			t.Errorf("Expected user ID 'user123', got '%s'", resp.ID)
		}
		if resp.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", resp.Email)
		}
	})
}
