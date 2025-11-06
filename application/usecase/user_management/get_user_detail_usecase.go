package user_management

import (
	"context"
	"fmt"

	"github.com/fixora/fixora/application/port/inbound"
	"github.com/fixora/fixora/application/port/outbound"
)

type GetUserDetailUseCase struct {
	userRepo outbound.UserRepository
}

func NewGetUserDetailUseCase(userRepo outbound.UserRepository) *GetUserDetailUseCase {
	return &GetUserDetailUseCase{
		userRepo: userRepo,
	}
}

func (uc *GetUserDetailUseCase) Execute(ctx context.Context, userID string) (*inbound.GetUserDetailResponse, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Find user by ID
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == outbound.ErrUserNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Convert to response DTO
	response := &inbound.GetUserDetailResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return response, nil
}
