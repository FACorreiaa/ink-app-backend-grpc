package internal

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/logger"
	"github.com/FACorreiaa/ink-app-backend-grpc/protocol/grpc"
	"github.com/FACorreiaa/ink-app-backend-protos/container"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc/reflection"
)

// -- Server components

// isReady is used for kube liveness probes, it's only latched to true once
// the gRPC server is ready to handle requests
var isReady atomic.Value

func ServeGRPC(ctx context.Context, port string, brokers *container.Brokers) error {
	// When you have a configured prometheus registry and OTEL trace provider,
	// pass in as param 3 & 4

	// configure prometheus registry
	registry, err := setupPrometheusRegistry(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to configure prometheus registry")
	}
	tp, err := otelTraceProvider(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to configure jaeger trace provider")
	}
	server, listener, err := grpc.BootstrapServer(port, logger.Log, registry, tp)
	if err != nil {
		return errors.Wrap(err, "failed to configure grpc server")
	}

	// Replace with your actual handler service
	// implementation, err := service.NewDummyService(brokers)
	if err != nil {
		return errors.Wrap(err, "failed to initialize grpc handler service")
	}

	// Replace with your actual generated registration method
	// generated.RegisterDummyServer(server, implementation)

	// Enable reflection to be able to use grpcui or insomnia without
	// having to manually maintain .proto files
	reflection.Register(server)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			logger.Log.Warn("shutting down grpc server")
			server.GracefulStop()
			<-ctx.Done()
		}
	}()

	isReady.Store(true)

	logger.Log.Info("running grpc server", zap.String("port", port))
	return server.Serve(listener)
}

// ServeHTTP creates a simple server to serve Prometheus metrics for
// the collector, and (not included) healthcheck endpoints for K8S to
// query readiness. By default these should serve on "/healthz" and "/readyz"
func ServeHTTP(port string) error {
	logger.Log.Info("running http server", zap.String("port", port))

	cfg, err := config.InitConfig()
	if err != nil {
		zap.Error(err)
	}
	server := http.NewServeMux()
	// Add healthcheck endpoints
	server.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Perform health check logic here
		// For example, check if the server is healthy
		// Respond with appropriate status code
		w.WriteHeader(http.StatusOK)
	})

	server.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Perform readiness check logic here
		// For example, check if the server is ready to receive traffic
		// Respond with appropriate status code
		w.WriteHeader(http.StatusOK)
	})
	server.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	// Add your healthchecks here too

	listener := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		ReadHeaderTimeout: cfg.Server.Timeout,
		Handler:           server,
	}

	if err := listener.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "failed to create telemetry server")
	}

	return nil
}
