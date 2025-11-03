package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vobe/auth-service/application/port/outbound"
	"github.com/vobe/auth-service/domain/entity"
)

type refreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) outbound.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, revoked_at) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	var revokedAt sql.NullTime
	if token.RevokedAt != nil {
		revokedAt.Valid = true
		revokedAt.Time = *token.RevokedAt
	}
	
	_, err := r.db.ExecContext(ctx, query, 
		token.ID, 
		token.UserID, 
		token.Token, 
		token.ExpiresAt, 
		token.CreatedAt,
		revokedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	
	return nil
}

func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, revoked_at 
		FROM refresh_tokens 
		WHERE token = $1
	`
	
	var refreshToken entity.RefreshToken
	var revokedAt sql.NullTime
	
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.Token,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&revokedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, outbound.ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	
	if revokedAt.Valid {
		refreshToken.RevokedAt = &revokedAt.Time
	}
	
	return &refreshToken, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, token string) error {
	query := `
		UPDATE refresh_tokens 
		SET revoked_at = $1 
		WHERE token = $2 AND revoked_at IS NULL
	`
	
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, token)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return outbound.ErrRefreshTokenNotFound
	}
	
	return nil
}

func (r *refreshTokenRepository) RevokeByUserID(ctx context.Context, userID string) error {
	query := `
		UPDATE refresh_tokens 
		SET revoked_at = $1 
		WHERE user_id = $2 AND revoked_at IS NULL
	`
	
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, now, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh tokens by user ID: %w", err)
	}
	
	return nil
}