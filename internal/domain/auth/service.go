package auth

import (
	"context"

	upb "github.com/FACorreiaa/ink-app-backend-protos/modules/user/generated"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
)

type ServiceAuth struct {
	upb.UnimplementedAuthServer
	ctx            context.Context
	repo           domain.AuthRepository
	pgpool         *pgxpool.Pool
	redis          *redis.Client
	SessionManager *SessionManager
}

func NewService(ctx context.Context, repo domain.AuthRepository,
	db *pgxpool.Pool,
	redis *redis.Client,
	sessionManager *SessionManager) *ServiceAuth {
	return &ServiceAuth{ctx: ctx, repo: repo, pgpool: db, redis: redis, SessionManager: sessionManager}
}

func (s *ServiceAuth) Register(ctx context.Context, req *upb.RegisterRequest) (*upb.RegisterResponse, error) {
	return s.repo.Register(ctx, req)
}

func (s *ServiceAuth) Login(ctx context.Context, req *upb.LoginRequest) (*upb.LoginResponse, error) {
	return s.repo.Login(ctx, req)
}

func (s *ServiceAuth) Logout(ctx context.Context, req *upb.NilReq) (*upb.NilRes, error) {
	return s.repo.Logout(ctx, req)
}

func (s *ServiceAuth) ChangePassword(ctx context.Context, req *upb.ChangePasswordRequest) (*upb.ChangePasswordResponse, error) {
	return s.repo.ChangePassword(ctx, req)
}

func (s *ServiceAuth) ChangeEmail(ctx context.Context, req *upb.ChangeEmailRequest) (*upb.ChangeEmailResponse, error) {
	return s.repo.ChangeEmail(ctx, req)
}

func (s *ServiceAuth) GetAllUsers(ctx context.Context, _ *upb.GetAllUsersRequest) (*upb.GetAllUsersResponse, error) {
	users, err := s.repo.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *ServiceAuth) GetUserByID(ctx context.Context, req *upb.GetUserByIDRequest) (*upb.GetUserByIDResponse, error) {
	user, err := s.repo.GetUserByID(ctx, req)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *ServiceAuth) DeleteUser(ctx context.Context, req *upb.DeleteUserRequest) (*upb.DeleteUserResponse, error) {
	return s.repo.DeleteUser(ctx, req)
}

func (s *ServiceAuth) UpdateUser(ctx context.Context, req *upb.UpdateUserRequest) (*upb.UpdateUserResponse, error) {
	return s.repo.UpdateUser(ctx, req)
}

func (s *ServiceAuth) InsertUser(ctx context.Context, req *upb.InsertUserRequest) (*upb.InsertUserResponse, error) {
	return s.repo.InsertUser(ctx, req)
}
