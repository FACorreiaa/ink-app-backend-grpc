package main

import (
	"context"
	"os"

	"github.com/FACorreiaa/ink-app-backend-grpc/configs"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal"
	"github.com/FACorreiaa/ink-app-backend-grpc/logger"
	"github.com/FACorreiaa/ink-app-backend-protos/utils"
	"go.uber.org/zap"
)

// This is just an example - Do not copy verbatim
// ---
// In practice, everything other than main lives in various
// locations in the service's './internal' directory

func run() {
	log := logger.Log
	dbConfig, err := internal.NewDatabaseConfig()
	if err != nil {
		log.Error("failed to initialize database configuration", zap.Error(err))
		os.Exit(1)
	}

	pool, err := internal.Init(dbConfig.ConnectionURL)
	if err != nil {
		log.Error("failed to initialize database pool", zap.Error(err))
		os.Exit(1)
	}
	defer pool.Close()

	internal.WaitForDB(pool)

	redisConfig, err := internal.NewRedisConfig()

	if err = internal.Migrate(pool); err != nil {
		zap.Error(err)
		os.Exit(1)
	}
	if err != nil {
		log.Error("failed to initialize Redis configuration", zap.Error(err))
		return
	}
	log.Info("Connected to Redis", zap.String("host", redisConfig.Host))
}

func main() {
	ctx := context.Background()

	// You should get these from your config object instead
	// yml config
	cfg, err := configs.InitConfig()
	if err != nil {
		zap.Error(err)
	}
	// Setup logging (found in ./logger)
	if err := logger.Init(
		zap.DebugLevel,
		zap.String("service", "example"),
		zap.String("version", "v42.0.69"),
		zap.Strings("maintainers", []string{"@fc", "@FACorreiaa"}),
	); err != nil || logger.Log == nil {
		panic("failed to initialize logging")
	}

	log := logger.Log

	run()

	// Configure tracing & Prometheus first...
	tu := new(utils.TransportUtils)

	// Setup clients BEFORE setting up servers
	brokers := internal.ConfigureUpstreamClients(log, tu)
	if brokers == nil {
		logger.Log.Error("failed to configure brokers")

		return
	}

	// Listeners are blocking so make sure that you're running
	// them as goroutines. You could use a waitgroup, but you run
	// the risk of deadlock panics - We usually put the gRPC server
	// and any background workers in goroutines, and leave the HTTP
	// metrics server as the final keepalive for the process
	go func() {
		if err := internal.ServeGRPC(ctx, cfg.Server.GrpcPort, brokers); err != nil {
			logger.Log.Error("failed to serve grpc", zap.Error(err))
			return
		}
	}()

	if err := internal.ServeHTTP(cfg.Server.HttpPort); err != nil {
		logger.Log.Error("failed to serve http", zap.Error(err))
		return
	}

}
