package domain

import (
	"context"

	pb "github.com/FACorreiaa/ink-app-backend-protos/modules/customer/generated"
	"github.com/redis/go-redis/v9"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerService implements the Customer gRPC server
type CustomerService struct {
	pb.UnimplementedCustomerServer
	pgpool *pgxpool.Pool
	redis  *redis.Client
}

// NewCustomerService creates a new CustomerService
func NewCustomerService(db *pgxpool.Pool, redis *redis.Client) *CustomerService {
	return &CustomerService{pgpool: db, redis: redis}
}

func (s *CustomerService) GetCustomer(ctx context.Context, req *pb.GetCustomerReq) (*pb.GetCustomerRes, error) {
	// Implementation of GetCustomer
	return &pb.GetCustomerRes{}, nil
}

func (s *CustomerService) CreateCustomer(ctx context.Context, req *pb.CreateCustomerReq) (*pb.CreateCustomerRes, error) {
	// Implementation of CreateCustomer
	return &pb.CreateCustomerRes{}, nil
}

func (s *CustomerService) UpdateCustomer(ctx context.Context, req *pb.UpdateCustomerReq) (*pb.UpdateCustomerRes, error) {
	// Implementation of UpdateCustomer
	return &pb.UpdateCustomerRes{}, nil
}

func (s *CustomerService) DeleteCustomer(ctx context.Context, req *pb.DeleteCustomerReq) (*pb.NilRes, error) {
	// Implementation of DeleteCustomer
	return &pb.NilRes{}, nil
}
