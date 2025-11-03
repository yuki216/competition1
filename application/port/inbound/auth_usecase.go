package inbound

import (
	"context"
)

type LoginRequest struct {
	Email          string `json:"email" validate:"required,email"`
	Password       string `json:"password" validate:"required,min=8"`
	RememberMe     bool   `json:"remember_me"`
	RecaptchaToken string `json:"recaptcha_token"`
}

type LoginResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    // seconds until refresh token expiry (for cookie TTL)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"-"`
}

type MeResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type AuthUseCase interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Refresh(ctx context.Context, req RefreshRequest) (*RefreshResponse, error)
	Logout(ctx context.Context, req LogoutRequest) error
	Me(ctx context.Context, userID string) (*MeResponse, error)
}