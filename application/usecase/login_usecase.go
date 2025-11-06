package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/fixora/fixora/application/port/inbound"
	"github.com/fixora/fixora/application/port/outbound"
	"github.com/fixora/fixora/domain/entity"
	"github.com/fixora/fixora/domain/valueobject"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

type LoginUseCase struct {
	userRepo         outbound.UserRepository
	refreshTokenRepo outbound.RefreshTokenRepository
	tokenService     outbound.TokenService
	passwordService  outbound.PasswordService
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

func NewLoginUseCase(
	userRepo outbound.UserRepository,
	refreshTokenRepo outbound.RefreshTokenRepository,
	tokenService outbound.TokenService,
	passwordService outbound.PasswordService,
	accessTokenTTL, refreshTokenTTL time.Duration,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		tokenService:     tokenService,
		passwordService:  passwordService,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

func (uc *LoginUseCase) Login(ctx context.Context, req inbound.LoginRequest) (*inbound.LoginResponse, error) {
	// Validate credentials format
	credentials, err := valueobject.NewCredentials(req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	// Find user by email
	user, err := uc.userRepo.FindByEmail(ctx, credentials.Email())
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	err = uc.passwordService.ComparePassword(user.Password, credentials.Password())
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := uc.tokenService.GenerateAccessToken(outbound.TokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		return nil, err
	}

	refreshToken, err := uc.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Determine refresh TTL dynamically based on RememberMe
	refreshTTL := uc.refreshTokenTTL
	if !req.RememberMe {
		if refreshTTL >= (14 * 24 * time.Hour) {
			refreshTTL = 7 * 24 * time.Hour
		} else {
			refreshTTL = refreshTTL / 2
		}
	}

	expiresAt := time.Now().Add(refreshTTL)

	// Create refresh token entity
	refreshTokenEntity := entity.NewRefreshToken(
		generateID(),
		user.ID,
		refreshToken,
		expiresAt,
	)

	// Store refresh token
	err = uc.refreshTokenRepo.Create(ctx, refreshTokenEntity)
	if err != nil {
		return nil, err
	}

	return &inbound.LoginResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int(uc.accessTokenTTL.Seconds()),
		RefreshExpiresIn: int(refreshTTL.Seconds()),
		User: inbound.MeResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + generateRandomString(8)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
