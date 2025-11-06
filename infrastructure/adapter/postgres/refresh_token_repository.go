package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/fixora/fixora/application/port/outbound"
	"github.com/fixora/fixora/domain/entity"
)

type RefreshTokenRepositoryAdapter struct {
	db   *sql.DB
	salt string
}

func NewRefreshTokenRepositoryAdapter(db *sql.DB, salt string) outbound.RefreshTokenRepository {
	return &RefreshTokenRepositoryAdapter{
		db:   db,
		salt: salt,
	}
}

func (r *RefreshTokenRepositoryAdapter) Create(ctx context.Context, token *entity.RefreshToken) error {
	if token == nil {
		return fmt.Errorf("refresh token cannot be nil")
	}

	if token.ID == "" || token.UserID == "" || token.Token == "" {
		return fmt.Errorf("refresh token ID, user ID, and token are required")
	}

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, revoked, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Hash token before storing as BYTEA
	hashHex := hashToken(token.Token, r.salt)
	hashBytes, errHex := hex.DecodeString(hashHex)
	if errHex != nil {
		return fmt.Errorf("failed to decode token hash: %w", errHex)
	}

	// revoked flag based on RevokedAt presence
	revoked := token.RevokedAt != nil

	_, err := r.db.ExecContext(ctx, query,
		token.ID,
		token.UserID,
		hashBytes,
		token.ExpiresAt,
		token.CreatedAt,
		revoked,
		token.RevokedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepositoryAdapter) FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	query := `
		SELECT id, user_id, expires_at, created_at, revoked, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
		LIMIT 1
	`

	var refreshToken entity.RefreshToken
	var revokedAt sql.NullTime
	var revokedFlag bool

	// Hash provided token for lookup as BYTEA
	hashHex := hashToken(token, r.salt)
	hashBytes, errHex := hex.DecodeString(hashHex)
	if errHex != nil {
		return nil, fmt.Errorf("failed to decode token hash: %w", errHex)
	}

	err := r.db.QueryRowContext(ctx, query, hashBytes).Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&revokedFlag,
		&revokedAt,
	)

	// keep original token plaintext out of entity to avoid leakage
	refreshToken.Token = ""

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, outbound.ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	if revokedAt.Valid {
		refreshToken.RevokedAt = &revokedAt.Time
	} else if revokedFlag {
		// if revoked=true but no revoked_at, set to now for safety
		n := time.Now()
		refreshToken.RevokedAt = &n
	}

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

func (r *RefreshTokenRepositoryAdapter) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE, revoked_at = $1
		WHERE token_hash = $2 AND revoked = FALSE
	`

	now := time.Now()
	// Hash provided token for revocation as BYTEA
	hashHex := hashToken(token, r.salt)
	hashBytes, errHex := hex.DecodeString(hashHex)
	if errHex != nil {
		return fmt.Errorf("failed to decode token hash: %w", errHex)
	}
	result, err := r.db.ExecContext(ctx, query, now, hashBytes)
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

func (r *RefreshTokenRepositoryAdapter) RevokeByUserID(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE, revoked_at = $1
		WHERE user_id = $2 AND revoked = FALSE
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, now, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh tokens by user ID: %w", err)
	}

	return nil
}

func hashToken(raw, salt string) string {
	sum := sha256.Sum256([]byte(raw + salt))
	return hex.EncodeToString(sum[:])
}
