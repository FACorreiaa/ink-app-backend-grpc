package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/FACorreiaa/ink-app-backend-protos/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal"
	"github.com/FACorreiaa/ink-app-backend-grpc/logger"
)

func run() (*pgxpool.Pool, *redis.Client, error) {
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	cfg, err := config.InitConfig()
	if err != nil {
		logger.Log.Error("failed to initialize config", zap.Error(err))
		return nil, nil, err
	}

	log := logger.Log

	dbConfig, err := internal.NewDatabaseConfig()
	if err != nil {
		log.Error("failed to initialize database configuration", zap.Error(err))
		return nil, nil, err
	}

	pool, err := internal.Init(dbConfig.ConnectionURL)
	if err != nil {
		log.Error("failed to initialize database pool", zap.Error(err))
		return nil, nil, err
	}
	internal.WaitForDB(pool)
	log.Info("Connected to Postgres", zap.String("host", cfg.Repositories.Postgres.Host))

	redisClient, err := internal.NewRedisConfig()
	if err != nil {
		log.Error("failed to initialize Redis configuration", zap.Error(err))
		pool.Close()
		return nil, nil, err
	}

	log.Info("Connected to Redis", zap.String("host", cfg.Repositories.Redis.Host))

	if err = internal.Migrate(pool); err != nil {
		log.Error("failed to migrate database", zap.Error(err))
		pool.Close()
		redisClient.Close()
		return nil, nil, err
	}

	return pool, redisClient, nil
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

	// Start HTTP server
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

	pool, redisClient, err := run()
	if err != nil {
		log.Error("failed to run the application", zap.Error(err))
		return
	}
	defer pool.Close()
	defer redisClient.Close()

	tu := new(utils.TransportUtils)

	brokers := internal.ConfigureUpstreamClients(log, tu)
	if brokers == nil {
		log.Error("failed to configure brokers")
		return
	}

	appContainer := internal.NewAppContainer(ctx, pool, redisClient)

	if err = startServer(ctx, &cfg, appContainer, reg); err != nil {
		logger.Log.Error("service error", zap.Error(err))
	}
}
