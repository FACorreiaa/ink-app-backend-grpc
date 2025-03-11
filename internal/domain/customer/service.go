package customer

import (
	"context"

	upc "github.com/FACorreiaa/ink-app-backend-protos/modules/customer/generated"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
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
	return nil, nil
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
