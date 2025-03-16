package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/FACorreiaa/ink-app-backend-protos/utils"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal"
	"github.com/FACorreiaa/ink-app-backend-grpc/logger"
)

func run() (*config.TenantDBManager, *config.TenantRedisManager, error) {
	cfg, err := config.InitConfig()
	if err != nil {
		logger.Log.Error("failed to initialize config", zap.Error(err))
		return nil, nil, err
	}

	log := logger.Log

	// Initialize tenant database manager
	dbManager, err := internal.NewTenantDBManager(&cfg)
	if err != nil {
		log.Error("failed to initialize tenant database manager", zap.Error(err))
		return nil, nil, err
	}

	// Initialize each tenant system
	ctx := context.Background()
	for _, tenant := range cfg.Tenants {
		pool, err := dbManager.GetTenantDB(tenant.Subdomain)
		if err != nil {
			log.Error("failed to get tenant DB", zap.String("subdomain", tenant.Subdomain), zap.Error(err))
			return nil, nil, err
		}
		if err := internal.InitializeTenantSystem(ctx, pool, &tenant); err != nil {
			log.Error("failed to initialize tenant system", zap.String("subdomain", tenant.Subdomain), zap.Error(err))
			return nil, nil, err
		}
		log.Info("Tenant initialized", zap.String("subdomain", tenant.Subdomain))
	}

	// Initialize Redis (shared across tenants)
	redisClient, err := internal.NewTenantRedisManager(&cfg)
	if err != nil {
		log.Error("failed to initialize Redis configuration", zap.Error(err))
		return nil, nil, err
	}
	log.Info("Connected to Redis", zap.String("host", cfg.Redis.Host))

	return dbManager, redisClient, nil
}

func startServer(ctx context.Context, cfg *config.Config, container *internal.AppContainer, reg *prometheus.Registry) error {
	errChan := make(chan error, 2)

	// Start gRPC server
	go func() {
		if err := internal.ServeGRPC(ctx, cfg.Server.GrpcPort, container, reg); err != nil {
			logger.Log.Error("gRPC server error", zap.Error(err))
			errChan <- err
		}
	}()
	logger.Log.Info("Serving gRPC", zap.String("port", cfg.Server.GrpcPort))

	// Start HTTP server (for metrics, etc.)
	go func() {
		if err := internal.ServeHTTP(cfg.Server.HTTPPort, reg); err != nil {
			logger.Log.Error("HTTP server error", zap.Error(err))
			errChan <- err
		}
	}()
	logger.Log.Info("Serving HTTP", zap.String("port", cfg.Server.HTTPPort))

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	reg := prometheus.NewRegistry()
	println("Loaded prometheus registry")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.InitConfig()
	if err != nil {
		zap.L().Error("failed to initialize config", zap.Error(err))
		return
	}

	if err := logger.Init(
		zap.DebugLevel,
		zap.String("service", "example"),
		zap.String("version", "v42.0.69"),
		zap.Strings("maintainers", []string{"@fc", "@FACorreiaa"}),
	); err != nil || logger.Log == nil {
		panic("failed to initialize logging")
	}

	log := logger.Log

	dbManager, redisManager, err := run()
	if err != nil {
		log.Error("failed to run the application", zap.Error(err))
		return
	}
	defer func() {
		for _, tenantDB := range dbManager.Tenants {
			tenantDB.Pool.Close()
		}
	}()

	defer func() {
		for _, tenantRedis := range redisManager.Tenants {
			tenantRedis.Client.Close()
		}
	}()

	tu := new(utils.TransportUtils)
	brokers := internal.ConfigureUpstreamClients(log, tu)
	if brokers == nil {
		log.Error("failed to configure brokers")
		return
	}

	// Pass dbManager to AppContainer instead of a single pool
	appContainer := internal.NewAppContainer(ctx, dbManager, redisManager)

	if err = startServer(ctx, &cfg, appContainer, reg); err != nil {
		logger.Log.Error("service error", zap.Error(err))
	}
}
