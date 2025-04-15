package auth

import (
	"context"
	"fmt"
	"log"
	"strings"

	pb "github.com/FACorreiaa/ink-app-backend-protos/modules/studio/generated"
	"golang.org/x/crypto/bcrypt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
)

//"go.opentelemetry.io/otel"
//"go.opentelemetry.io/otel/attribute"
//"github.com/FACorreiaa/ink-app-backend-grpc/protocol/grpc/middleware/grpcrequest"

// StudioAuthService implements the gRPC server
type StudioAuthService struct {
	pb.UnimplementedAuthServiceServer
	repo     domain.StudioAuthRepository
	userRepo domain.UserRepository
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

	fmt.Printf("Metadata: %v\n", md)

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

func getAuthenticatedUserID(ctx context.Context) (string, error) {
	value := ctx.Value("user_id")
	if value == nil {
		log.Println("getAuthenticatedUserID: Context value for key 'userID' is nil (key not found)") // DEBUG LOG
		return "", status.Error(codes.Unauthenticated, "authentication claims missing from context (key not found)")
	}

	userID, ok := value.(string)
	if !ok {
		log.Printf("getAuthenticatedUserID: Context value for key 'userID' is not a string. Type is %T\n", value) // DEBUG LOG
		return "", status.Error(codes.Internal, "invalid authentication claim type in context")                   // Internal error because interceptor messed up
	}

	if userID == "" {
		log.Println("getAuthenticatedUserID: userID retrieved from context is an empty string")
		return "", status.Error(codes.Unauthenticated, "authentication claims invalid (empty user ID)")
	}

	return userID, nil
}

// Register registers a new user
func (s *StudioAuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
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

	return &pb.RegisterResponse{Message: "User registered successfully"}, nil
}

// Login authenticates a user
func (s *StudioAuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err = validateLoginRequest(req); err != nil {
		// Return InvalidArgument status code
		return nil, status.Errorf(codes.InvalidArgument, "invalid login request: %v", err)
	}

	accessToken, newRefreshToken, err := s.repo.Login(ctx, tenant, req.Email, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		Message:      "Login successful",
	}, nil
}

// Logout invalidates a session
func (s *StudioAuthService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
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

	return &pb.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}, nil
}

func (s *StudioAuthService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.TokenResponse, error) {
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

	return &pb.TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// ChangePassword updates a user's password
func (s *StudioAuthService) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Username == "" || req.OldPassword == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "all fields required")
	}

	// Fetch email by username
	_, _, _, err = s.userRepo.GetUserByEmail(ctx, tenant, req.Username) // Assuming username can be email
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	err = s.userRepo.ChangePassword(ctx, tenant, req.Username, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to change password: %v", err)
	}

	return &pb.ChangePasswordResponse{Message: "Password changed successfully"}, nil
}

// ChangeEmail updates a user's email
func (s *StudioAuthService) ChangeEmail(ctx context.Context, req *pb.ChangeEmailRequest) (*pb.ChangeEmailResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Username == "" || req.Password == "" || req.NewEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "all fields required")
	}

	// Fetch current email by username
	_, _, _, err = s.userRepo.GetUserByEmail(ctx, tenant, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	err = s.userRepo.ChangeEmail(ctx, tenant, req.Username, req.Password, req.NewEmail)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to change email: %v", err)
	}

	return &pb.ChangeEmailResponse{Message: "Email changed successfully"}, nil
}

// ValidateSession implements the ValidateSession RPC method
func (s *StudioAuthService) ValidateSession(ctx context.Context, req *pb.ValidateSessionRequest) (*pb.ValidateSessionResponse, error) {
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
	return &pb.ValidateSessionResponse{
		Valid:    true,
		UserId:   session.ID,
		Username: session.Username,
		Email:    session.Email,
	}, nil
}

func (s *StudioAuthService) ChangeOwnPassword(ctx context.Context, req *pb.ChangeOwnPasswordRequest) (*pb.ChangeOwnPasswordResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	actingUserID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Validation
	if req.OldPassword == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "old and new passwords are required")
	}
	if req.OldPassword == req.NewPassword {
		return nil, status.Error(codes.InvalidArgument, "new password must be different")
	}
	// Add complexity checks for req.NewPassword

	// 1. Verify the old password
	err = s.repo.VerifyPassword(ctx, tenant, actingUserID, req.OldPassword)
	if err != nil {
		// Log the specific internal error if needed
		return nil, status.Errorf(codes.Unauthenticated, "invalid old password") // Don't reveal too much
	}

	// 2. Hash the new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Error hashing new password for user %s: %v\n", actingUserID, err)
		return nil, status.Error(codes.Internal, "failed to process new password")
	}

	// 3. Update the password in the repository
	err = s.repo.UpdatePassword(ctx, tenant, actingUserID, string(newHashedPassword))
	if err != nil {
		fmt.Printf("Error updating password for user %s, tenant %s: %v\n", actingUserID, tenant, err)
		return nil, status.Errorf(codes.Internal, "failed to update password: %v", err)
	}

	// 4. Invalidate refresh tokens for security
	err = s.repo.InvalidateAllUserRefreshTokens(ctx, tenant, actingUserID)
	if err != nil {
		fmt.Printf("Warning: Failed to invalidate refresh tokens for user %s after password change, tenant %s: %v\n", actingUserID, tenant, err)
	}

	return &pb.ChangeOwnPasswordResponse{Message: "Password changed successfully"}, nil
}

// AdminResetUserPassword handles requests for an admin resetting another user's password.
func (s *StudioAuthService) AdminResetUserPassword(ctx context.Context, req *pb.AdminResetUserPasswordRequest) (*pb.AdminResetUserPasswordResponse, error) {
	tenant, err := extractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	actingAdminID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Validation
	if req.TargetUserId == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "target user ID and new password are required")
	}
	if actingAdminID == req.TargetUserId {
		return nil, status.Error(codes.InvalidArgument, "use ChangeOwnPassword endpoint for self-service")
	}
	// Add complexity checks for req.NewPassword

	// 1. Verify the acting user is an admin
	// You might fetch the acting user's role or have it in claims
	actingUserRole, err := s.repo.GetUserRole(ctx, tenant, actingAdminID)
	if err != nil {
		fmt.Printf("Error fetching role for admin %s, tenant %s: %v\n", actingAdminID, tenant, err)
		return nil, status.Errorf(codes.Internal, "failed to verify admin status")
	}
	// Check if the role has admin permissions (e.g., "OWNER", "ADMIN")
	isAdmin := actingUserRole == "OWNER" || actingUserRole == "ADMIN" // Define roles properly
	if !isAdmin {
		return nil, status.Error(codes.PermissionDenied, "action requires administrative privileges")
	}

	// 2. Hash the new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Error hashing new password during admin reset for target user %s: %v\n", req.TargetUserId, err)
		return nil, status.Error(codes.Internal, "failed to process new password")
	}

	// 3. Update the target user's password in the repository
	err = s.repo.UpdatePassword(ctx, tenant, req.TargetUserId, string(newHashedPassword))
	if err != nil {
		// Check if the error is "user not found"
		if err.Error() == "user not found or password unchanged" { // Or use specific domain error
			return nil, status.Errorf(codes.NotFound, "target user not found")
		}
		fmt.Printf("Error updating password via admin reset for target user %s, tenant %s: %v\n", req.TargetUserId, tenant, err)
		return nil, status.Errorf(codes.Internal, "failed to update password: %v", err)
	}

	// 4. Invalidate refresh tokens for the *target user*
	err = s.repo.InvalidateAllUserRefreshTokens(ctx, tenant, req.TargetUserId)
	if err != nil {
		fmt.Printf("Warning: Failed to invalidate refresh tokens for target user %s after admin password reset, tenant %s: %v\n", req.TargetUserId, tenant, err)
	}

	// Optional: Send a notification email to the target user?

	return &pb.AdminResetUserPasswordResponse{Message: "User password reset successfully by admin"}, nil
}
