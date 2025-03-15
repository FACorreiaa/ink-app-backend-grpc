package internal

import (
	"context"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
)

type AppContainer struct {
	Ctx          context.Context
	DBManager    *config.TenantDBManager
	RedisManager *config.TenantRedisManager
	//AuthServiceManager *auth.TenantServiceManager
}

func NewAppContainer(ctx context.Context, dbManager *config.TenantDBManager, redisManager *config.TenantRedisManager) *AppContainer {

	//sessionManager := auth.NewSessionManager(dbManager, redisManager)

	// Create tenant repository managers instead of individual service maps
	//authRepoManager := auth.NewTenantRepositoryManager(dbManager, redisManager, sessionManager)
	//customerRepoManager := customer.NewTenantRepositoryManager(dbManager, redisManager)

	// Create tenant-aware service managers
	//authServiceManager := auth.NewTenantServiceManager(ctx, authRepoManager)
	//customerServiceManager := customer.NewTenantServiceManager(ctx, customerRepoManager, dbManager, redisManager)

	return &AppContainer{
		Ctx:          ctx,
		DBManager:    dbManager,
		RedisManager: redisManager,
		//AuthServiceManager: authServiceManager,
		//CustomerServiceManager: customerServiceManager,
	}
}
