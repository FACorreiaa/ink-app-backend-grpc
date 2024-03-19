package internal

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	expo "go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/zap"
)

const meterName = "github.com/open-telemetry/opentelemetry-go/example/prometheus"

func setupPrometheusRegistry(ctx context.Context) (*prometheus.Registry, error) {
	// Initialize Prometheus registry
	reg := prometheus.NewRegistry()
	//#nosec
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

func otelTraceProvider(ctx context.Context) (*trace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		zap.Error(err)
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("todo-service"),
			semconv.DeploymentEnvironmentKey.String("production"),
		)),
	)

	tracerProvider := trace.NewTracerProvider(trace.WithBatcher(exp))
	defer func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			zap.Error(err)
			panic(err)
		}
	}()
	otel.SetTracerProvider(tracerProvider)
	return tp, nil
}

// func setupOtelTraceProvider() (func(context.Context) error, error) {
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
