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
