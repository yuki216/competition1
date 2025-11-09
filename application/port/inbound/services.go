package inbound

import (
	"context"
	"time"
)

// TokenService defines the interface for JWT token operations
type TokenService interface {
	GenerateAccessToken(userID string) (string, error)
	GenerateRefreshToken() (string, error)
	ValidateToken(token string) (*TokenClaims, error)
}

// TokenClaims represents the claims in a JWT token
type TokenClaims struct {
	UserID string
	Email  string
}

// PasswordService defines the interface for password operations
type PasswordService interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, hash string) (bool, error)
}

// RecaptchaService defines reCAPTCHA verification behavior used by application/usecases
// Implemented by infrastructure/service/recaptcha
type RecaptchaService interface {
	VerifyToken(ctx context.Context, token string) (bool, error)
	IsEnabled() bool
}

// RateLimitService defines rate limiting behavior used by application/usecases and middleware
// Implemented by infrastructure/service/ratelimit
type RateLimitService interface {
	CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	Increment(ctx context.Context, key string, window time.Duration) error
	Block(ctx context.Context, key string, duration time.Duration, reason string) error
	IsBlocked(ctx context.Context, key string) (bool, error)
	GetAttempts(ctx context.Context, key string) (int, error)
}