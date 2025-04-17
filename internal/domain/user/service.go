package user

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/bcrypt"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/FACorreiaa/ink-app-backend-grpc/protocol/grpc/middleware/grpcrequest"
	pb "github.com/FACorreiaa/ink-app-backend-protos/modules/user/generated"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	pb.UnimplementedUserServiceServer
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func mapDomainRoleToProto(domainRole string) pb.User_Role {
	var protoRole pb.User_Role

	// Use ToUpper for case-insensitive matching
	switch strings.ToUpper(domainRole) {
	case "ADMIN", "OWNER": // Grouping similar roles if they map the same
		protoRole = pb.User_ADMIN
	case "STAFF":
		protoRole = pb.User_STAFF
	case "USER": // Assuming "customer" in domain might map to "USER" in proto? Adjust as needed.
		protoRole = pb.User_USER
	case "MODERATOR":
		protoRole = pb.User_MODERATOR
	default:
		// Log potentially unknown roles if helpful for debugging
		// log.Printf("Warning: Mapping unknown domain role '%s' to ROLE_UNSPECIFIED", domainRole)
		protoRole = pb.User_ROLE_UNSPECIFIED
	}

	return protoRole
}

func mapProtoRoleToDomain(protoRole pb.User_Role) (string, error) {
	switch protoRole {
	case pb.User_ADMIN:
		return "ADMIN", nil // Or "OWNER" if that's your domain representation
	case pb.User_STAFF:
		return "STAFF", nil
	case pb.User_USER:
		return "USER", nil
	case pb.User_MODERATOR:
		return "MODERATOR", nil
	case pb.User_ROLE_UNSPECIFIED:
		return "", fmt.Errorf("role cannot be ROLE_UNSPECIFIED")
	default:
		// This case handles potential future enum values not yet known
		// or invalid integer values passed somehow.
		return "", fmt.Errorf("unknown or invalid proto role value: %d", protoRole)
	}
}

func (s *UserService) GetAllUsers(ctx context.Context, req *pb.GetUsersReq) (*pb.GetUsersRes, error) {
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

	users, err := s.repo.GetAllUsers(ctx, tenant)
	if err != nil {
		return nil, err
	}

	var domainUser *domain.User

	protoRole := mapDomainRoleToProto(domainUser.Role)

	res := &pb.GetUsersRes{}
	for _, user := range users {
		res.Users = append(res.Users, &pb.User{
			UserId:   user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     protoRole,
		})
	}

	span.SetAttributes(
		semconv.ServiceNameKey.String("Workout"),
		attribute.String("grpc.method", "GetExercises"),
		attribute.String("request.id", req.GetRequest().RequestId),
	)

	return &pb.GetUsersRes{
		//Message: "User registered successfully",
		Users: res.Users,
		Response: &pb.BaseResponse{
			Success:   true,
			RequestId: req.Request.RequestId,
			TraceId:   traceID,
		}}, nil
}
func (s *UserService) GetUserByID(ctx context.Context, req *pb.GetUserByIDReq) (*pb.GetUserByIDRes, error) {
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

	user, err := s.repo.GetUserByID(traceContext, tenant, req.UserId)
	if err != nil {
		return nil, err
	}

	var domainUser *domain.User
	protoRole := mapDomainRoleToProto(domainUser.Role)

	span.SetAttributes(
		semconv.ServiceNameKey.String("Workout"),
		attribute.String("grpc.method", "GetExercises"),
		attribute.String("request.id", req.GetRequest().RequestId),
	)

	res := &pb.GetUserByIDRes{
		User: &pb.User{
			UserId:   user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     protoRole,
		},
	}

	return &pb.GetUserByIDRes{
		User: res.User,
		Response: &pb.BaseResponse{
			Success:   true,
			RequestId: req.Request.RequestId,
			TraceId:   traceID,
		}}, nil
}

func (s *UserService) InsertUser(ctx context.Context, req *pb.InsertUserReq) (*pb.InsertUserRes, error) {
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

	tenant, err := domain.ExtractTenantFromContext(traceContext) // Assuming this function exists
	if err != nil {
		return nil, err
	}

	// --- Validate Input ---
	if req.User == nil {
		return nil, status.Error(codes.InvalidArgument, "user data is required")
	}
	if req.User.Username == "" || req.User.Email == "" || req.User.PasswordHash == "" {
		return nil, status.Error(codes.InvalidArgument, "username, email, and password are required")
	}

	// --- Map Proto Role to Domain Role String ---
	domainRoleString, err := mapProtoRoleToDomain(req.User.Role)
	if err != nil {
		// Return an error if the role is unspecified or invalid
		return nil, status.Errorf(codes.InvalidArgument, "invalid user role provided: %v", err)
	}

	// --- Hash Password (Assuming PasswordHash in request is plain text) ---
	// IMPORTANT: The request field is named PasswordHash, but it should contain the
	// PLAIN TEXT password for a *new* user insertion. You need to hash it here.
	// If the client is sending an already hashed password, this needs rethinking.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.User.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		// Log internal error
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	// --- Create Domain User Struct ---
	user := &domain.User{
		Username: req.User.Username,
		Email:    req.User.Email,
		Password: string(hashedPassword), // Store the *hashed* password
		Role:     domainRoleString,       // Assign the converted string
		StudioID: req.User.StudioId,      // Assign StudioID if provided
		// CreatedAt/UpdatedAt will be set by DB usually or in repo
	}

	// --- Call Repository ---
	// Assuming InsertUser in repo handles setting CreatedAt/UpdatedAt
	// and potentially returning the generated ID if needed
	err = s.repo.InsertUser(traceContext, tenant, user)
	if err != nil {
		// Check for specific errors like email/username already exists
		return nil, status.Errorf(codes.Internal, "failed to insert user: %v", err) // Or more specific codes
	}

	span.SetAttributes(
		semconv.ServiceNameKey.String("Workout"),
		attribute.String("grpc.method", "GetExercises"),
		attribute.String("request.id", req.GetRequest().RequestId),
	)

	return &pb.InsertUserRes{
		Message: "User created successfully",
		Response: &pb.BaseResponse{
			Success:   true,
			RequestId: req.Request.RequestId,
			TraceId:   traceID,
		}}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserReq) (*pb.UpdateUserRes, error) {
	return nil, nil
}
func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserReq) (*pb.DeleteUserRes, error) {
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

	err = s.repo.DeleteUser(traceContext, tenant, req.UserId)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(
		semconv.ServiceNameKey.String("Workout"),
		attribute.String("grpc.method", "GetExercises"),
		attribute.String("request.id", req.GetRequest().RequestId),
	)

	return &pb.DeleteUserRes{
		//Message: "User registered successfully",
		Response: &pb.BaseResponse{
			Success:   true,
			RequestId: req.Request.RequestId,
			TraceId:   traceID,
		}}, nil
}
func (s *UserService) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailReq) (*pb.GetUserByEmailRes, error) {
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

	var domainUser *domain.User
	protoRole := mapDomainRoleToProto(domainUser.Role)

	user, err := s.repo.GetUserByEmail(traceContext, tenant, req.Email)
	if err != nil {
		return nil, err
	}

	res := &pb.GetUserByEmailRes{
		User: &pb.User{
			UserId:   user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     protoRole,
		},
	}

	span.SetAttributes(
		semconv.ServiceNameKey.String("Workout"),
		attribute.String("grpc.method", "GetExercises"),
		attribute.String("request.id", req.GetRequest().RequestId),
	)

	return &pb.GetUserByEmailRes{
		//Message: "User registered successfully",
		User: res.User,
		Response: &pb.BaseResponse{
			Success:   true,
			RequestId: req.Request.RequestId,
			TraceId:   traceID,
		}}, nil
}
func (s *UserService) GetUserByUsername(ctx context.Context, req *pb.GetUserByUsernameReq) (*pb.GetUserByUsernameRes, error) {
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

	var domainUser *domain.User
	protoRole := mapDomainRoleToProto(domainUser.Role)

	user, err := s.repo.GetUserByUsername(ctx, tenant, req.Username)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(
		semconv.ServiceNameKey.String("Workout"),
		attribute.String("grpc.method", "GetExercises"),
		attribute.String("request.id", req.GetRequest().RequestId),
	)

	res := &pb.GetUserByUsernameRes{
		User: &pb.User{
			UserId:   user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     protoRole,
		},
	}

	return &pb.GetUserByUsernameRes{
		//Message: "User registered successfully",
		User: res.User,
		Response: &pb.BaseResponse{
			Success:   true,
			RequestId: req.Request.RequestId,
			TraceId:   traceID,
		}}, nil
}
