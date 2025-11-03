package outbound

import (
	"context"
	"errors"

	"github.com/vobe/auth-service/domain/entity"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenAlreadyExists = errors.New("refresh token already exists")
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error)
	Revoke(ctx context.Context, token string) error
	RevokeByUserID(ctx context.Context, userID string) error
}