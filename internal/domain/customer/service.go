package customer

import (
	"context"
	"time"

	upc "github.com/FACorreiaa/ink-app-backend-protos/modules/customer/generated"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/FACorreiaa/ink-app-backend-grpc/protocol/grpc/middleware/grpcrequest"
)

type ServiceCustomer struct {
	upc.UnimplementedCustomerServiceServer
	ctx    context.Context
	repo   domain.CustomerRepository
	pgpool *pgxpool.Pool
	redis  *redis.Client
}

func NewService(ctx context.Context, repo domain.CustomerRepository,
	db *pgxpool.Pool,
	redis *redis.Client) *ServiceCustomer {
	return &ServiceCustomer{ctx: ctx, repo: repo, pgpool: db, redis: redis}
}

func (s *ServiceCustomer) CreateCustomer(ctx context.Context, req *upc.CreateCustomerRequest) (*upc.CreateCustomerResponse, error) {
	// Validate request
	if req == nil || req.Customer == nil {
		return nil, status.Error(codes.InvalidArgument, "customer details are required")
	}

	tracer := otel.Tracer("SyncInk")
	ctx, span := tracer.Start(ctx, "/CustomerService/CreateCustomer")
	defer span.End()

	requestID, ok := ctx.Value(grpcrequest.RequestIDKey{}).(string)
	if !ok {
		return nil, status.Error(codes.Internal, "request id not found in context")
	}

	if req.Request == nil {
		req.Request = &upc.BaseRequest{}
	}

	req.Request.RequestId = requestID

	// Extract customer info from request
	protoCustomer := req.Customer

	// Parse birthday if provided
	var dateOfBirth time.Time
	if protoCustomer.Birthday != "" {
		var err error
		dateOfBirth, err = time.Parse("2006-01-02", protoCustomer.Birthday)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid birthday format: %v", err)
		}
	}

	userID := ctx.Value("userID").(string)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "studioID is missing in metadata")
	}

	// Map proto customer to domain customer
	customer := &domain.Customer{
		StudioID:     userID,
		FirstName:    protoCustomer.FirstName,
		LastName:     protoCustomer.LastName,
		FullName:     protoCustomer.FullName,
		Email:        protoCustomer.Email,
		Phone:        protoCustomer.Phone,
		Notes:        protoCustomer.Notes,
		NIF:          protoCustomer.Nif,
		Address:      protoCustomer.Address,
		City:         protoCustomer.City,
		PostalCode:   protoCustomer.PostalCode,
		Country:      protoCustomer.Country,
		IDCardNumber: protoCustomer.IdCardNumber,
		DateOfBirth:  dateOfBirth,
		IsArchived:   protoCustomer.IsArchived,
	}

	// Check if customer with same email already exists
	if customer.Email != "" {
		exists, err := s.repo.ExistsByEmail(ctx, customer.Email)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check customer existence: %v", err)
		}
		if exists {
			return nil, status.Error(codes.AlreadyExists, "customer with this email already exists")
		}
	}

	// Check if customer with same phone already exists
	if customer.Phone != "" {
		exists, err := s.repo.ExistsByPhone(ctx, customer.Phone)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check customer existence: %v", err)
		}
		if exists {
			return nil, status.Error(codes.AlreadyExists, "customer with this phone already exists")
		}
	}

	// Create customer in repository
	id, err := s.repo.Create(ctx, customer)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create customer: %v", err)
	}

	span.SetAttributes(
		attribute.String("request.id", req.Request.RequestId),
		attribute.String("request.details", req.String()),
	)

	// Return response
	return &upc.CreateCustomerResponse{
		CustomerId: id,
		Message:    "Customer created successfully",
		Response: &upc.BaseResponse{
			Status: upc.Status_name[int32(upc.Status_SUCCESS)],
		},
	}, nil
}

func (s *ServiceCustomer) GetCustomer(ctx context.Context, req *upc.GetCustomerRequest) (*upc.GetCustomerResponse, error) {
	return nil, nil
}

func (s *ServiceCustomer) UpdateCustomer(ctx context.Context, req *upc.UpdateCustomerRequest) (*upc.UpdateCustomerResponse, error) {
	return nil, nil
}

func (s *ServiceCustomer) DeleteCustomer(ctx context.Context, req *upc.DeleteCustomerRequest) (*upc.DeleteCustomerResponse, error) {
	return nil, nil
}

func (s *ServiceCustomer) ListCustomers(ctx context.Context, req *upc.ListCustomersRequest) (*upc.ListCustomersResponse, error) {
	return nil, nil
}

func (s *ServiceCustomer) ArchiveCustomer(ctx context.Context, req *upc.ArchiveCustomerRequest) (*upc.ArchiveCustomerResponse, error) {
	return nil, nil
}

func (s *ServiceCustomer) GetCustomerHistory(ctx context.Context, req *upc.GetCustomerHistoryRequest) (*upc.GetCustomerHistoryResponse, error) {
	return nil, nil
}

func (s *ServiceCustomer) AddCustomerNote(ctx context.Context, req *upc.AddCustomerNoteRequest) (*upc.AddCustomerNoteResponse, error) {
	return nil, nil
}

func (s *ServiceCustomer) SearchCustomers(ctx context.Context, req *upc.SearchCustomersRequest) (*upc.SearchCustomersResponse, error) {
	return nil, nil
}
