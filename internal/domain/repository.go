package domain

import (
	"context"

	upb "github.com/FACorreiaa/ink-app-backend-protos/modules/auth/generated"
)

type AuthRepository interface {
	Register(ctx context.Context, req *upb.RegisterRequest) (*upb.RegisterResponse, error)
	Login(ctx context.Context, req *upb.LoginRequest) (*upb.LoginResponse, error)
	Logout(ctx context.Context, req *upb.NilReq) (*upb.NilRes, error)
	ChangePassword(ctx context.Context, req *upb.ChangePasswordRequest) (*upb.ChangePasswordResponse, error)
	ChangeEmail(ctx context.Context, req *upb.ChangeEmailRequest) (*upb.ChangeEmailResponse, error)

	// GetAllUsers Users Methods
	GetAllUsers(ctx context.Context) (*upb.GetAllUsersResponse, error)
	GetUserByID(ctx context.Context, req *upb.GetUserByIDRequest) (*upb.GetUserByIDResponse, error)
	DeleteUser(ctx context.Context, req *upb.DeleteUserRequest) (*upb.DeleteUserResponse, error)
	UpdateUser(ctx context.Context, req *upb.UpdateUserRequest) (*upb.UpdateUserResponse, error)
	InsertUser(ctx context.Context, req *upb.InsertUserRequest) (*upb.InsertUserResponse, error)
}
