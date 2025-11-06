package user_management

import (
	"context"
	"errors"
	"fmt"

	"github.com/vobe/auth-service/application/port/inbound"
	"github.com/vobe/auth-service/application/port/outbound"
)

type UpdateUserUseCase struct {
	userRepo outbound.UserRepository
}

func NewUpdateUserUseCase(userRepo outbound.UserRepository) *UpdateUserUseCase {
	return &UpdateUserUseCase{
		userRepo: userRepo,
	}
}

func (uc *UpdateUserUseCase) Execute(ctx context.Context, userID string, req inbound.UpdateUserRequest) error {
	// Validate input
	if err := uc.validateUpdateUserRequest(req); err != nil {
		return err
	}

	// Find existing user
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == outbound.ErrUserNotFound {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Update user fields
	if req.Name != "" {
		user.UpdateName(req.Name)
	}
	if req.Role != "" {
		user.UpdateRole(req.Role)
	}
	if req.Status != "" {
		user.UpdateStatus(req.Status)
	}

	// Save to repository
	err = uc.userRepo.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (uc *UpdateUserUseCase) validateUpdateUserRequest(req inbound.UpdateUserRequest) error {
	// Validate name if provided
	if req.Name != "" {
		if len(req.Name) < 2 || len(req.Name) > 255 {
			return errors.New("invalid name format")
		}
	}

	// Validate role if provided
	if req.Role != "" {
		validRoles := []string{"admin", "user", "superadmin", "employee"}
		isValidRole := false
		for _, role := range validRoles {
			if req.Role == role {
				isValidRole = true
				break
			}
		}
		if !isValidRole {
			return errors.New("invalid role")
		}
	}

	// Validate status if provided
	if req.Status != "" {
		validStatuses := []string{"active", "inactive"}
		isValidStatus := false
		for _, status := range validStatuses {
			if req.Status == status {
				isValidStatus = true
				break
			}
		}
		if !isValidStatus {
			return errors.New("invalid status")
		}
	}

	return nil
}