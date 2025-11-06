package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vobe/auth-service/application/port/outbound"
	"github.com/vobe/auth-service/domain/entity"
)

type UserRepositoryAdapter struct {
	db *sql.DB
}

func NewUserRepositoryAdapter(db *sql.DB) outbound.UserRepository {
	return &UserRepositoryAdapter{
		db: db,
	}
}

func (r *UserRepositoryAdapter) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	query := `
		SELECT id, email, password, role, created_at, updated_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	var user entity.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepositoryAdapter) FindByID(ctx context.Context, id string) (*entity.User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	query := `
		SELECT id, email, password, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user entity.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, outbound.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}

	return &user, nil
}

func (r *UserRepositoryAdapter) Create(ctx context.Context, user *entity.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	if user.ID == "" || user.Email == "" || user.Password == "" {
		return fmt.Errorf("user ID, email, and password are required")
	}

	query := `
		INSERT INTO users (id, email, password, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Password,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}