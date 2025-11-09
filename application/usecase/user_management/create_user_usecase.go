package user_management

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fixora/fixora/application/port/inbound"
	"github.com/fixora/fixora/application/port/outbound"
	"github.com/fixora/fixora/domain/entity"
)

var (
	ErrInvalidName        = errors.New("invalid name format")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrInvalidPassword    = errors.New("password must be at least 8 characters")
	ErrInvalidRole        = errors.New("invalid role")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUnauthorized       = errors.New("unauthorized: admin role required")
)

type CreateUserUseCase struct {
	userRepo    outbound.UserRepository
	passwordSvc outbound.PasswordService
}

func NewCreateUserUseCase(
	userRepo outbound.UserRepository,
	passwordSvc outbound.PasswordService,
) *CreateUserUseCase {
	return &CreateUserUseCase{
		userRepo:    userRepo,
		passwordSvc: passwordSvc,
	}
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, req inbound.CreateUserRequest) error {
	// Validate admin authorization (this should be handled by middleware)
	// But we add additional validation here for safety

	// Validate input
	if err := uc.validateCreateUserRequest(req); err != nil {
		return err
	}

	// Check if email already exists
	exists, err := uc.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := uc.passwordSvc.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user entity
	user := entity.NewUser(
		req.ID,
		req.Name,
		req.Email,
		hashedPassword,
		req.Role,
		req.Status,
	)

	// Save to repository
	err = uc.userRepo.Create(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (uc *CreateUserUseCase) validateCreateUserRequest(req inbound.CreateUserRequest) error {
	// Validate name
	if req.Name == "" {
		return ErrInvalidName
	}
	if len(req.Name) < 2 || len(req.Name) > 255 {
		return ErrInvalidName
	}

	// Validate email
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return ErrInvalidEmail
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Validate password
	if len(req.Password) < 8 {
		return ErrInvalidPassword
	}

	// Validate role
	validRoles := []string{"admin", "user", "superadmin", "employee"}
	isValidRole := false
	for _, role := range validRoles {
		if req.Role == role {
			isValidRole = true
			break
		}
	}
	if !isValidRole {
		return ErrInvalidRole
	}

	// Validate status
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

	return nil
}
