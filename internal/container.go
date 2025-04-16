package internal

import (
	"context"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain/auth"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain/studio"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain/user"
)

type AppContainer struct {
	Ctx          context.Context
	DBManager    *config.TenantDBManager
	RedisManager *config.TenantRedisManager

	// Service managers
	//StudioService *studio.StudioService
	AuthService   *auth.StudioAuthService
	StudioService *studio.StudioService
	UserService   *user.UserService
	// Add other services as needed
}

func NewAppContainer(ctx context.Context, dbManager *config.TenantDBManager, redisManager *config.TenantRedisManager) *AppContainer {
	// Create repositories with tenant awareness
	studioAuthRepo := auth.NewAuthRepository(dbManager, redisManager)
	studioRepo := studio.NewStudioRepository(dbManager, redisManager)
	userRepo := user.NewUserRepository(dbManager, redisManager)

	// // Get a pool from the manager for initialization
	// defaultPool := dbManager.GetDefaultPool()
	// defaultRedis := redisManager.GetDefaultClient()

	return &AppContainer{
		Ctx:           ctx,
		DBManager:     dbManager,
		RedisManager:  redisManager,
		StudioService: studio.NewStudioService(studioRepo),
		AuthService:   auth.NewStudioAuthService(studioAuthRepo, userRepo),
		UserService:   user.NewUserService(userRepo),
	}
}
