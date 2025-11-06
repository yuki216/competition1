package user_management

import (
	"context"
	"fmt"

	"github.com/fixora/fixora/application/port/inbound"
	"github.com/fixora/fixora/application/port/outbound"
)

type ListUsersUseCase struct {
	userRepo outbound.UserRepository
}

func NewListUsersUseCase(userRepo outbound.UserRepository) *ListUsersUseCase {
	return &ListUsersUseCase{
		userRepo: userRepo,
	}
}

func (uc *ListUsersUseCase) Execute(ctx context.Context, req inbound.ListUsersRequest) (*inbound.ListUsersResponse, error) {
	// Set default pagination
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Calculate offset
	offset := (req.Page - 1) * req.Limit

	// Build filters
	filters := outbound.UserFilters{
		Name:   req.Filter.Name,
		Role:   req.Filter.Role,
		Status: req.Filter.Status,
	}

	// Get users from repository
	users, total, err := uc.userRepo.FindAll(ctx, offset, req.Limit, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert to response DTOs
	userItems := make([]inbound.UserListItem, len(users))
	for i, user := range users {
		userItems[i] = inbound.UserListItem{
			ID:     user.ID,
			Name:   user.Name,
			Email:  user.Email,
			Role:   user.Role,
			Status: user.Status,
		}
	}

	response := &inbound.ListUsersResponse{
		Users: userItems,
		Pagination: inbound.PaginationInfo{
			Page:  req.Page,
			Limit: req.Limit,
			Total: total,
		},
	}

	return response, nil
}
