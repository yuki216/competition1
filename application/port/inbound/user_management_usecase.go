package inbound

import (
	"context"
)

// Create User
type CreateUserRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name" validate:"required,min=2,max=255"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role" validate:"required"`
	Status   string `json:"status" validate:"required"`
}

type CreateUserResponse struct {
	Message string `json:"message"`
}

// Update User
type UpdateUserRequest struct {
	Name   string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Role   string `json:"role,omitempty" validate:"omitempty"`
	Status string `json:"status,omitempty" validate:"omitempty"`
}

type UpdateUserResponse struct {
	Message string `json:"message"`
}

// Delete User
type DeleteUserRequest struct {
	ID string `json:"id"`
}

type DeleteUserResponse struct {
	Message string `json:"message"`
}

// Get User Detail
type GetUserDetailRequest struct {
	ID string `json:"id"`
}

type GetUserDetailResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// List Users
type ListUsersRequest struct {
	Page   int                    `json:"page" validate:"min=1"`
	Limit  int                    `json:"limit" validate:"min=1,max=100"`
	Filter ListUsersFilter        `json:"filter"`
}

type ListUsersFilter struct {
	Name   string `json:"name,omitempty"`
	Role   string `json:"role,omitempty"`
	Status string `json:"status,omitempty"`
}

type UserListItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

type ListUsersResponse struct {
	Users      []UserListItem   `json:"users"`
	Pagination PaginationInfo   `json:"pagination"`
}

type PaginationInfo struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// User Management Use Case Interface
type UserManagementUseCase interface {
	CreateUser(ctx context.Context, req CreateUserRequest) error
	UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) error
	DeleteUser(ctx context.Context, userID string) error
	GetUserDetail(ctx context.Context, userID string) (*GetUserDetailResponse, error)
	ListUsers(ctx context.Context, req ListUsersRequest) (*ListUsersResponse, error)
}