package studio

import (
	"context"
	"strings"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	ups "github.com/FACorreiaa/ink-app-backend-protos/modules/studio/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// StudioAuthService implements the gRPC server
type StudioAuthService struct {
	ups.UnimplementedStudioAuthServer
	ctx  context.Context
	repo domain.StudioAuthRepository
}

// NewStudioAuth creates a new StudioAuthService
func NewStudioAuth(ctx context.Context, repo domain.StudioAuthRepository) *StudioAuthService {
	return &StudioAuthService{
		ctx:  ctx,
		repo: repo,
	}
}

// extractTenantFromContext extracts tenant information from gRPC metadata
func extractTenantFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no metadata provided")
	}

	// Try to extract from X-Tenant header first
	tenantValues := md.Get("X-Tenant")
	if len(tenantValues) > 0 && tenantValues[0] != "" {
		return tenantValues[0], nil
	}

	// Fallback to host extraction if possible
	hostValues := md.Get(":authority")
	if len(hostValues) > 0 {
		host := hostValues[0]
		// Extract subdomain as tenant
		parts := strings.Split(host, ".")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0], nil
		}
	}

	return "", status.Error(codes.Unauthenticated, "tenant not specified")
}

// SignIn implements the SignIn RPC method
func (s *StudioAuthService) Login(ctx context.Context, req *ups.LoginRequest) (*ups.LoginResponse, error) {
	// Extract tenant from context
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}

	// Authenticate user
	sessionID, err := s.repo.Login(ctx, tenant, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication failed")
	}

	// Create response
	return &ups.LoginResponse{
		Token: sessionID,
	}, nil
}

// SignOut implements the SignOut RPC method
func (s *StudioAuthService) Logout(ctx context.Context, req *ups.LogoutRequest) (*ups.LogoutResponse, error) {
	// Extract tenant from context
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID required")
	}

	// Sign out
	err = s.repo.Logout(ctx, tenant, req.SessionId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to sign out")
	}

	// Create response
	return &ups.LogoutResponse{
		Success: true,
	}, nil
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
