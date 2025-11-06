package user_management

import (
	"context"
	"fmt"

	"github.com/fixora/fixora/application/port/outbound"
)

type DeleteUserUseCase struct {
	userRepo outbound.UserRepository
}

func NewDeleteUserUseCase(userRepo outbound.UserRepository) *DeleteUserUseCase {
	return &DeleteUserUseCase{
		userRepo: userRepo,
	}
}

func (uc *DeleteUserUseCase) Execute(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Perform soft delete
	err := uc.userRepo.SoftDelete(ctx, userID)
	if err != nil {
		if err == outbound.ErrUserNotFound {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
