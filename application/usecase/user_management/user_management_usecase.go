package user_management

import (
	"context"

	"github.com/vobe/auth-service/application/port/inbound"
	"github.com/vobe/auth-service/application/port/outbound"
)

type UserManagementUseCaseImpl struct {
	createUserUseCase   *CreateUserUseCase
	updateUserUseCase   *UpdateUserUseCase
	deleteUserUseCase   *DeleteUserUseCase
	getUserDetailUseCase *GetUserDetailUseCase
	listUsersUseCase    *ListUsersUseCase
}

func NewUserManagementUseCase(
	userRepo outbound.UserRepository,
	passwordSvc outbound.PasswordService,
) inbound.UserManagementUseCase {
	return &UserManagementUseCaseImpl{
		createUserUseCase:   NewCreateUserUseCase(userRepo, passwordSvc),
		updateUserUseCase:   NewUpdateUserUseCase(userRepo),
		deleteUserUseCase:   NewDeleteUserUseCase(userRepo),
		getUserDetailUseCase: NewGetUserDetailUseCase(userRepo),
		listUsersUseCase:    NewListUsersUseCase(userRepo),
	}
}

func (uc *UserManagementUseCaseImpl) CreateUser(ctx context.Context, req inbound.CreateUserRequest) error {
	return uc.createUserUseCase.Execute(ctx, req)
}

func (uc *UserManagementUseCaseImpl) UpdateUser(ctx context.Context, userID string, req inbound.UpdateUserRequest) error {
	return uc.updateUserUseCase.Execute(ctx, userID, req)
}

func (uc *UserManagementUseCaseImpl) DeleteUser(ctx context.Context, userID string) error {
	return uc.deleteUserUseCase.Execute(ctx, userID)
}

func (uc *UserManagementUseCaseImpl) GetUserDetail(ctx context.Context, userID string) (*inbound.GetUserDetailResponse, error) {
	return uc.getUserDetailUseCase.Execute(ctx, userID)
}

func (uc *UserManagementUseCaseImpl) ListUsers(ctx context.Context, req inbound.ListUsersRequest) (*inbound.ListUsersResponse, error) {
	return uc.listUsersUseCase.Execute(ctx, req)
}