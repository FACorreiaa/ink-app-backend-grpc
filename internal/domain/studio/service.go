package studio

import (
	"context"
	"strings"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/FACorreiaa/ink-app-backend-grpc/protocol/grpc/middleware/grpcrequest"
	ups "github.com/FACorreiaa/ink-app-backend-protos/modules/studio/generated"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// StudioAuthService implements the gRPC server
type StudioAuthService struct {
	ups.UnimplementedStudioAuthServer
	repo domain.StudioAuthRepository
}

// NewStudioAuthService creates a new StudioAuthService
func NewStudioAuthService(repo domain.StudioAuthRepository) *StudioAuthService {
	return &StudioAuthService{repo: repo}
}

// extractTenantFromContext extracts tenant from gRPC metadata
func extractTenantFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no metadata provided")
	}

	tenantValues := md.Get("X-Tenant")
	if len(tenantValues) > 0 && tenantValues[0] != "" {
		return tenantValues[0], nil
	}

	hostValues := md.Get(":authority")
	if len(hostValues) > 0 {
		host := hostValues[0]
		parts := strings.Split(host, ".")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0], nil
		}
	}

	return "", status.Error(codes.Unauthenticated, "tenant not specified")
}

// Register registers a new user
func (s *StudioAuthService) Register(ctx context.Context, req *ups.RegisterRequest) (*ups.RegisterResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "all fields are required")
	}

	err = s.repo.Register(ctx, tenant, req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	return &ups.RegisterResponse{Message: "User registered successfully"}, nil
}

// Login authenticates a user
func (s *StudioAuthService) Login(ctx context.Context, req *ups.LoginRequest) (*ups.LoginResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}

	accessToken, err := s.repo.Login(ctx, tenant, req.Email, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	return &ups.LoginResponse{
		AccessToken: accessToken,
		Message:     "Login successful",
	}, nil
}

// Logout invalidates a session
func (s *StudioAuthService) Logout(ctx context.Context, req *ups.LogoutRequest) (*ups.LogoutResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID required")
	}

	// Assuming session_id is the refresh token
	err = s.repo.Logout(ctx, tenant, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to logout: %v", err)
	}

	return &ups.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}, nil
}

func (s *StudioAuthService) RefreshToken(ctx context.Context, req *ups.RefreshTokenRequest) (*ups.TokenResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token required")
	}

	newAccessToken, newRefreshToken, err := s.repo.RefreshSession(ctx, tenant, req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token: %v", err)
	}

	return &ups.TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// ChangePassword updates a user's password
func (s *StudioAuthService) ChangePassword(ctx context.Context, req *ups.ChangePasswordRequest) (*ups.ChangePasswordResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Username == "" || req.OldPassword == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "all fields required")
	}

	// Fetch email by username
	_, _, _, err = s.repo.GetUserByEmail(ctx, tenant, req.Username) // Assuming username can be email
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	err = s.repo.ChangePassword(ctx, tenant, req.Username, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to change password: %v", err)
	}

	return &ups.ChangePasswordResponse{Message: "Password changed successfully"}, nil
}

// ChangeEmail updates a user's email
func (s *StudioAuthService) ChangeEmail(ctx context.Context, req *ups.ChangeEmailRequest) (*ups.ChangeEmailResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Username == "" || req.Password == "" || req.NewEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "all fields required")
	}

	// Fetch current email by username
	_, _, _, err = s.repo.GetUserByEmail(ctx, tenant, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	err = s.repo.ChangeEmail(ctx, tenant, req.Username, req.Password, req.NewEmail)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to change email: %v", err)
	}

	return &ups.ChangeEmailResponse{Message: "Email changed successfully"}, nil
}

// ValidateSession implements the ValidateSession RPC method
func (s *StudioAuthService) ValidateSession(ctx context.Context, req *ups.ValidateSessionRequest) (*ups.ValidateSessionResponse, error) {
	// Extract tenant from context
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID required")
	}

	// Get session
	session, err := s.repo.GetSession(ctx, tenant, req.SessionId)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired session")
	}

	// Create response
	return &ups.ValidateSessionResponse{
		Valid:    true,
		UserId:   session.ID,
		Username: session.Username,
		Email:    session.Email,
	}, nil
}

func (s *StudioAuthService) GetUserByID(ctx context.Context, req *ups.GetUserByIDRequest) (*ups.GetUserByIDResponse, error) {
	tracer := otel.Tracer("InkMe")
	ctx, span := tracer.Start(ctx, "StudioAuth/GetUserByID")
	defer span.End()

	requestID, ok := ctx.Value(grpcrequest.RequestIDKey{}).(string)
	if !ok {
		return nil, status.Error(codes.Internal, "request id not found in context")
	}

	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid tenant: %v", err)
	}

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID required")
	}

	user, err := s.repo.GetUserByID(ctx, tenant, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	span.SetAttributes(
		attribute.String("request.id", req.UserId),
		attribute.String("request.details", req.String()),
	)

	// Convert string role to User_Role enum
	var userRole ups.User_Role
	switch user.Role {
	case "admin":
		userRole = ups.User_ADMIN
	case "staff":
		userRole = ups.User_STAFF
	case "user":
		userRole = ups.User_USER
	case "moderator":
		userRole = ups.User_MODERATOR
	default:
		userRole = ups.User_ROLE_UNSPECIFIED
	}

	return &ups.GetUserByIDResponse{
		User: &ups.User{
			UserId:   user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     userRole,
		},
		Response: &ups.BaseResponse{
			RequestId: requestID,
			Status:    "success",
		},
	}, nil
}

func (s *StudioAuthService) GetAllUsers(ctx context.Context, req *ups.GetAllUsersRequest) (*ups.GetAllUsersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

func (s *StudioAuthService) UpdateUser(ctx context.Context, req *ups.UpdateUserRequest) (*ups.UpdateUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

func (s *StudioAuthService) DeleteUser(ctx context.Context, req *ups.DeleteUserRequest) (*ups.DeleteUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

func (s *StudioAuthService) InsertUser(ctx context.Context, req *ups.InsertUserRequest) (*ups.InsertUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

func (s *StudioAuthService) GetUserByEmail(ctx context.Context, req *ups.GetUserByEmailRequest) (*ups.GetUserByEmailResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

func (s *StudioAuthService) GetUserByUsername(ctx context.Context, req *ups.GetUserByUsernameRequest) (*ups.GetUserByUsernameResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}
