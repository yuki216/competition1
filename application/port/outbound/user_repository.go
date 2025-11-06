package outbound

import (
	"context"
	"errors"

	"github.com/vobe/auth-service/domain/entity"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	SoftDelete(ctx context.Context, id string) error
	FindAll(ctx context.Context, offset, limit int, filters UserFilters) ([]*entity.User, int, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindByRole(ctx context.Context, role string) ([]*entity.User, error)
}

type UserFilters struct {
	Name   string
	Role   string
	Status string
}