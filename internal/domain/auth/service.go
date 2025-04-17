package auth

import (
	"context"
	"fmt"
	"log"

	pb "github.com/FACorreiaa/ink-app-backend-protos/modules/studio/generated"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"golang.org/x/crypto/bcrypt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/FACorreiaa/ink-app-backend-grpc/protocol/grpc/middleware/grpcrequest"
)

//"go.opentelemetry.io/otel"
//"go.opentelemetry.io/otel/attribute"
//"github.com/FACorreiaa/ink-app-backend-grpc/protocol/grpc/middleware/grpcrequest"

// StudioAuthService implements the gRPC server
type StudioAuthService struct {
	pb.UnimplementedAuthServiceServer
	authRepo domain.StudioAuthRepository
	userRepo domain.UserRepository
}

// NewStudioAuthService creates a new StudioAuthService
func NewStudioAuthService(authRepo domain.StudioAuthRepository, userRepo domain.UserRepository) *StudioAuthService {
	return &StudioAuthService{authRepo: authRepo, userRepo: userRepo}
}

func getAuthenticatedUserID(ctx context.Context) (string, error) {
	// ---> Use the typed key from the domain package <---
	value := ctx.Value(domain.UserIDKey)

	if value == nil {
		log.Println("getAuthenticatedUserID: Context value for key 'UserIDKey' is nil (key not found)")
		return "", status.Error(codes.Unauthenticated, "authentication claims missing from context (key not found)")
	}

	userID, ok := value.(string)
	if !ok {
		log.Printf("getAuthenticatedUserID: Context value for key 'UserIDKey' is not a string. Type is %T\n", value)
		return "", status.Error(codes.Internal, "invalid authentication claim type in context")
	}

	if userID == "" {
		log.Println("getAuthenticatedUserID: userID retrieved from context is an empty string")
		return "", status.Error(codes.Unauthenticated, "authentication claims invalid (empty user ID)")
	}

	return userID, nil
}

// Register registers a new user
func (s *StudioAuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	traceContext, span := otel.Tracer("ink-me").Start(ctx, "Register")
	defer span.End()
	traceID := span.SpanContext().TraceID().String()
	requestID, ok := ctx.Value(grpcrequest.RequestIDKey{}).(string)
	if !ok {
		return nil, status.Error(codes.Internal, "request id not found in context")
	}

	if req.Request == nil {
		req.Request = &pb.BaseRequest{}
	}

	req.Request.RequestId = requestID

	tenant, err := domain.ExtractTenantFromContext(traceContext)
	if err != nil {
		return nil, err
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "all fields are required")
	}

	err = s.authRepo.Register(ctx, tenant, req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	span.SetAttributes(
		semconv.ServiceNameKey.String("Workout"),
		attribute.String("grpc.method", "GetExercises"),
		attribute.String("request.id", req.GetRequest().RequestId),
	)

	return &pb.RegisterResponse{
		Message: "User registered successfully",
		Response: &pb.BaseResponse{
			RequestId: req.Request.RequestId,
			TraceId:   traceID,
		}}, nil
}

// Login authenticates a user
func (s *StudioAuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	tenant, err := domain.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err = domain.ValidateLoginRequest(req); err != nil {
		// Return InvalidArgument status code
		return nil, status.Errorf(codes.InvalidArgument, "invalid login request: %v", err)
	}

	accessToken, newRefreshToken, err := s.authRepo.Login(ctx, tenant, req.Email, req.Password)
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
	tenant, err := domain.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID required")
	}

	// Assuming session_id is the refresh token
	err = s.authRepo.Logout(ctx, tenant, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to logout: %v", err)
	}

	return &pb.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}, nil
}

func (s *StudioAuthService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.TokenResponse, error) {
	tenant, err := domain.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token required")
	}

	newAccessToken, newRefreshToken, err := s.authRepo.RefreshSession(ctx, tenant, req.RefreshToken)
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
	tenant, err := domain.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	actingUserID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.Username == "" || req.OldPassword == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "all fields required")
	}

	// Fetch email by username
	_, err = s.userRepo.GetUserByID(ctx, tenant, actingUserID) // Assuming username can be email
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
// func (s *StudioAuthService) ChangeEmail(ctx context.Context, req *pb.ChangeEmailRequest) (*pb.ChangeEmailResponse, error) {
// 	tenant, err := domain.ExtractTenantFromContext(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	actingUserID, err := getAuthenticatedUserID(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if req.Username == "" || req.Password == "" || req.NewEmail == "" {
// 		return nil, status.Error(codes.InvalidArgument, "all fields required")
// 	}

// 	// Fetch current email by username
// 	_, err = s.userRepo.GetUserByID(ctx, tenant, actingUserID)
// 	if err != nil {
// 		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
// 	}

// 	err = s.authRepo.ChangeEmail(ctx, tenant, req.Username, req.Password, req.NewEmail)
// 	if err != nil {
// 		return nil, status.Errorf(codes.InvalidArgument, "failed to change email: %v", err)
// 	}

// 	return &pb.ChangeEmailResponse{Message: "Email changed successfully"}, nil
// }

func (s *StudioAuthService) ChangeEmail(ctx context.Context, req *pb.ChangeEmailRequest) (*pb.ChangeEmailResponse, error) {
	tenant, err := domain.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 1. Get the ID of the user making the request from the context
	actingUserID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		// This means the token was invalid or missing
		return nil, err
	}

	// 2. Validate Inputs
	// Note: We don't need req.Username if we use the actingUserID from the token.
	// The request should probably contain current_password and new_email.
	// Adjust proto definition if needed. Let's assume req.Password is the current password.
	if req.Password == "" || req.NewEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "current password and new email are required")
	}
	// Add email format validation for req.NewEmail

	// 3. Verify the user's *current* password
	// We use actingUserID obtained securely from the token/context
	err = s.authRepo.VerifyPassword(ctx, tenant, actingUserID, req.Password)
	if err != nil {
		log.Printf("ChangeEmail: Password verification failed for user %s, tenant %s: %v\n", actingUserID, tenant, err)
		if err.Error() == "invalid password" { // Check for specific error if repo returns it
			return nil, status.Error(codes.Unauthenticated, "invalid current password")
		}
		// Check if userRepo.VerifyPassword can return "user not found"
		if err.Error() == "user not found" {
			return nil, status.Error(codes.NotFound, "authenticated user not found during password verification")
		}
		return nil, status.Errorf(codes.Internal, "failed to verify password: %v", err)
	}

	// 4. Call the CORRECT repository method to update the email
	// This method should exist on the UserRepository interface and implementation
	err = s.userRepo.UpdateEmail(ctx, tenant, actingUserID, req.NewEmail)
	if err != nil {
		// Log the detailed error
		log.Printf("ChangeEmail: Failed to update email for user %s, tenant %s: %v\n", actingUserID, tenant, err)
		// Check for specific errors, e.g., if the new email is already taken (unique constraint)
		// if isUniqueConstraintError(err) { // Implement this check based on DB driver
		//     return nil, status.Error(codes.AlreadyExists, "new email address is already in use")
		// }
		// Check if UpdateEmail can return "user not found" (shouldn't if VerifyPassword succeeded, but good practice)
		if err.Error() == "user not found" { // Or however your repo signals this
			return nil, status.Error(codes.NotFound, "user not found during email update")
		}
		return nil, status.Errorf(codes.Internal, "failed to update email: %v", err) // Generic internal error
	}

	// Optional: Invalidate refresh tokens as email change might warrant re-login
	// err = s.authRepo.InvalidateAllUserRefreshTokens(ctx, tenant, actingUserID)
	// if err != nil { log.Printf(...) }

	return &pb.ChangeEmailResponse{Message: "Email changed successfully"}, nil
}

// ValidateSession implements the ValidateSession RPC method
func (s *StudioAuthService) ValidateSession(ctx context.Context, req *pb.ValidateSessionRequest) (*pb.ValidateSessionResponse, error) {
	// Extract tenant from context
	tenant, err := domain.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID required")
	}

	// Get session
	session, err := s.authRepo.GetSession(ctx, tenant, req.SessionId)
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
	tenant, err := domain.ExtractTenantFromContext(ctx)
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
	err = s.authRepo.VerifyPassword(ctx, tenant, actingUserID, req.OldPassword)
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
	err = s.authRepo.UpdatePassword(ctx, tenant, actingUserID, string(newHashedPassword))
	if err != nil {
		fmt.Printf("Error updating password for user %s, tenant %s: %v\n", actingUserID, tenant, err)
		return nil, status.Errorf(codes.Internal, "failed to update password: %v", err)
	}

	// 4. Invalidate refresh tokens for security
	err = s.authRepo.InvalidateAllUserRefreshTokens(ctx, tenant, actingUserID)
	if err != nil {
		fmt.Printf("Warning: Failed to invalidate refresh tokens for user %s after password change, tenant %s: %v\n", actingUserID, tenant, err)
	}

	return &pb.ChangeOwnPasswordResponse{Message: "Password changed successfully"}, nil
}

// AdminResetUserPassword handles requests for an admin resetting another user's password.
func (s *StudioAuthService) AdminResetUserPassword(ctx context.Context, req *pb.AdminResetUserPasswordRequest) (*pb.AdminResetUserPasswordResponse, error) {
	tenant, err := domain.ExtractTenantFromContext(ctx)
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
	actingUserRole, err := s.authRepo.GetUserRole(ctx, tenant, actingAdminID)
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
	err = s.authRepo.UpdatePassword(ctx, tenant, req.TargetUserId, string(newHashedPassword))
	if err != nil {
		// Check if the error is "user not found"
		if err.Error() == "user not found or password unchanged" { // Or use specific domain error
			return nil, status.Errorf(codes.NotFound, "target user not found")
		}
		fmt.Printf("Error updating password via admin reset for target user %s, tenant %s: %v\n", req.TargetUserId, tenant, err)
		return nil, status.Errorf(codes.Internal, "failed to update password: %v", err)
	}

	// 4. Invalidate refresh tokens for the *target user*
	err = s.authRepo.InvalidateAllUserRefreshTokens(ctx, tenant, req.TargetUserId)
	if err != nil {
		fmt.Printf("Warning: Failed to invalidate refresh tokens for target user %s after admin password reset, tenant %s: %v\n", req.TargetUserId, tenant, err)
	}

	// Optional: Send a notification email to the target user?

	return &pb.AdminResetUserPasswordResponse{Message: "User password reset successfully by admin"}, nil
}
