package internal

import (
	"context"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain/studio"
)

type AppContainer struct {
	Ctx          context.Context
	DBManager    *config.TenantDBManager
	RedisManager *config.TenantRedisManager

	// Service managers
	//StudioService *studio.StudioService
	StudioAuth *studio.StudioAuthService
	// Add other services as needed
}

func NewAppContainer(ctx context.Context, dbManager *config.TenantDBManager, redisManager *config.TenantRedisManager) *AppContainer {
	// Create repositories with tenant awareness
	studioAuthRepo := studio.NewStudioAuthRepository(dbManager, redisManager)
	//studioRepo := studio.NewUserRepository(dbManager)

	// // Get a pool from the manager for initialization
	// defaultPool := dbManager.GetDefaultPool()
	// defaultRedis := redisManager.GetDefaultClient()

	return &AppContainer{
		Ctx:          ctx,
		DBManager:    dbManager,
		RedisManager: redisManager,
		//StudioService: studio.NewStudioService(ctx, studioRepo, defaultPool, defaultRedis),
		StudioAuth: studio.NewStudioAuthService(studioAuthRepo),
	}
}
