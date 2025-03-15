package studio

import (
	"context"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	ups "github.com/FACorreiaa/ink-app-backend-protos/modules/studio/generated"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// StudioServiceImpl handles studio domain operations
type StudioService struct {
	ups.UnimplementedStudioServiceServer
	ctx    context.Context
	repo   domain.StudioRepository // Use appropriate repository
	pgpool *pgxpool.Pool
	redis  *redis.Client
}

// StudioAuthImpl handles authentication operations
type StudioAuthService struct {
	ups.UnimplementedStudioAuthServer
	ctx    context.Context
	repo   domain.StudioAuthRepository // Use appropriate repository
	pgpool *pgxpool.Pool
	redis  *redis.Client
}

func NewStudioService(ctx context.Context, repo domain.StudioRepository,
	db *pgxpool.Pool, redis *redis.Client) *StudioService {
	return &StudioService{ctx: ctx, repo: repo, pgpool: db, redis: redis}
}

func NewStudioAuth(ctx context.Context, repo domain.StudioAuthRepository,
	db *pgxpool.Pool, redis *redis.Client) *StudioAuthService {
	return &StudioAuthService{ctx: ctx, repo: repo, pgpool: db, redis: redis}
}
