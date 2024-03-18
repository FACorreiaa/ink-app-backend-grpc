package internal

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/FACorreiaa/ink-app-backend-grpc/configs"
	"github.com/FACorreiaa/ink-app-backend-grpc/logger"
	"github.com/FACorreiaa/ink-app-backend-grpc/protocol/grpc"
	"github.com/FACorreiaa/ink-app-backend-protos/container"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	expo "go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.1"
	"go.uber.org/zap"
	"google.golang.org/grpc/reflection"
)

const meterName = "github.com/open-telemetry/opentelemetry-go/example/prometheus"

func setupPrometheusRegistry(ctx context.Context) (*prometheus.Registry, error) {
	// Initialize Prometheus registry
	reg := prometheus.NewRegistry()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	exporter, err := expo.New(expo.WithRegisterer(reg))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create OpenTelemetry Prometheus exporter")
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter(meterName)
	opt := api.WithAttributes(
		attribute.Key("A").String("B"),
		attribute.Key("C").String("D"),
	)
	// Register the promhttp handler for serving metrics
	http.Handle("/prometheus/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	counter, err := meter.Float64Counter("foo", api.WithDescription("a simple counter"))
	if err != nil {
		zap.Error(err)
	}
	counter.Add(ctx, 5, opt)

	gauge, err := meter.Float64ObservableGauge("bar", api.WithDescription("a fun little gauge"))
	if err != nil {
		log.Fatal(err)
	}
	_, err = meter.RegisterCallback(func(_ context.Context, o api.Observer) error {
		n := -10. + rng.Float64()*(90.) // [-10, 100)
		o.ObserveFloat64(gauge, n, opt)
		return nil
	}, gauge)
	if err != nil {
		log.Fatal(err)
	}

	// This is the equivalent of prometheus.NewHistogramVec
	histogram, err := meter.Float64Histogram(
		"baz",
		api.WithDescription("a histogram with custom buckets and rename"),
		api.WithExplicitBucketBoundaries(64, 128, 256, 512, 1024, 2048, 4096),
	)
	if err != nil {
		log.Fatal(err)
	}
	histogram.Record(ctx, 136, opt)
	histogram.Record(ctx, 64, opt)
	histogram.Record(ctx, 701, opt)
	histogram.Record(ctx, 830, opt)

	return reg, nil
}

func jaegerTraceProvider() (*sdktrace.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("todo-service"),
			semconv.DeploymentEnvironmentKey.String("production"),
		)),
	)
	return tp, nil
}

//func setupOtelTraceProvider() (func(context.Context) error, error) {
//	ctx := context.Background()
//
//	res, err := resource.New(ctx,
//		resource.WithAttributes(
//			// the service name used to display traces in backends
//			semconv.ServiceName("test-service"),
//		),
//	)
//	if err != nil {
//		return nil, fmt.Errorf("failed to create resource: %w", err)
//	}
//
//	// If the OpenTelemetry Collector is running on a local cluster (minikube or
//	// microk8s), it should be accessible through the NodePort service at the
//	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
//	// endpoint of your cluster. If you run the app inside k8s, then you can
//	// probably connect directly to the service through dns.
//	ctx, cancel := context.WithTimeout(ctx, time.Second)
//	defer cancel()
//	conn, err := grpc.DialContext(ctx, "localhost:30080",
//		// Note the use of insecure transport here. TLS is recommended in production.
//		grpc.WithTransportCredentials(insecure.NewCredentials()),
//		grpc.WithBlock(),
//	)
//	if err != nil {
//		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
//	}
//
//	// Set up a trace exporter
//	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
//	if err != nil {
//		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
//	}
//
//	// Register the trace exporter with a TracerProvider, using a batch
//	// span processor to aggregate spans before export.
//	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
//	tracerProvider := sdktrace.NewTracerProvider(
//		sdktrace.WithSampler(sdktrace.AlwaysSample()),
//		sdktrace.WithResource(res),
//		sdktrace.WithSpanProcessor(bsp),
//	)
//	otel.SetTracerProvider(tracerProvider)
//
//	// set global propagator to tracecontext (the default is no-op).
//	otel.SetTextMapPropagator(propagation.TraceContext{})
//
//	// Shutdown will flush any remaining spans and shut down the exporter.
//	return tracerProvider.Shutdown, nil
//}

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
	tp, err := jaegerTraceProvider()
	if err != nil {
		return errors.Wrap(err, "failed to configure jaeger trace provider")
	}
	server, listener, err := grpc.BootstrapServer(port, logger.Log, registry, tp)
	if err != nil {
		return errors.Wrap(err, "failed to configure grpc server")
	}

	// Replace with your actual handler service
	//implementation, err := service.NewDummyService(brokers)
	if err != nil {
		return errors.Wrap(err, "failed to initialize grpc handler service")
	}

	// Replace with your actual generated registration method
	//generated.RegisterDummyServer(server, implementation)

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
func ServeHTTP(HTTPPort string) error {
	logger.Log.Info("running http server", zap.String("port", HTTPPort))

	cfg, err := configs.InitConfig()
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
		Addr:              fmt.Sprintf(":%s", cfg.Server.HttpPort),
		ReadHeaderTimeout: cfg.Server.Timeout,
		Handler:           server,
	}

	if err := listener.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "failed to create telemetry server")
	}

	return nil
}
