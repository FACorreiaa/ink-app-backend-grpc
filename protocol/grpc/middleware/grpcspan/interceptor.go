package grpcspan

import (
	"github.com/FACorreiaa/ink-me-backend-grpc/protocol/grpc/middleware"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// Interceptors wraps OpenTelemetry gRPC interceptors.
// We use a wrapper for this as the OpenTelemetry ecosystem changes
// *very* frequently, so we want to contain this change to a single point
// rather than having to update the servers and clients individually.
func Interceptors() (middleware.ClientInterceptor, middleware.ServerInterceptor) {
	clientInterceptor := middleware.ClientInterceptor{
		Unary:  otelgrpc.NewClientHandler(),
		Stream: otelgrpc.NewClientHandler(),
	}

	serverInterceptor := middleware.ServerInterceptor{
		Unary:  otelgrpc.NewServerHandler(),
		Stream: otelgrpc.NewServerHandler(),
	}

	return clientInterceptor, serverInterceptor
}
