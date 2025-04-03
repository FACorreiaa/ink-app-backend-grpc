package grpctenantinterceptor

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type tenantKey struct{}

// TenantInterceptor extracts the tenant ID and injects it into the context.
func TenantInterceptor(tenantValidator func(string) bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		var tenantID string

		// 1. Try X-Tenant Header
		tenantValues := md.Get("X-Tenant")
		if len(tenantValues) > 0 && tenantValues[0] != "" {
			tenantID = tenantValues[0]
		} else {
			// 2. Try :authority (host) subdomain
			hostValues := md.Get(":authority")
			if len(hostValues) > 0 {
				host := hostValues[0]
				// Basic subdomain extraction (adjust if needed for ports, complex hosts)
				parts := strings.Split(host, ".")
				// Assuming structure like tenant.domain.com or tenant.localhost:port
				if len(parts) > 1 && parts[0] != "" && parts[0] != "localhost" {
					tenantID = parts[0]
				}
			}
		}

		if tenantID == "" {
			return nil, status.Error(codes.Unauthenticated, "tenant identifier not found in request metadata")
		}

		// 3. Validate Tenant (Optional but Recommended)
		if tenantValidator != nil {
			if !tenantValidator(tenantID) {
				// Log the attempt with the invalid tenant ID
				fmt.Printf("Attempt with invalid tenant ID: %s\n", tenantID)
				return nil, status.Errorf(codes.PermissionDenied, "invalid tenant identifier: %s", tenantID)
			}
		}

		// 4. Inject tenant ID into context
		newCtx := context.WithValue(ctx, tenantKey{}, tenantID)

		// Proceed with the handler
		return handler(newCtx, req)
	}
}

// GetTenantFromContext retrieves the tenant ID injected by the interceptor.
func GetTenantFromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(tenantKey{}).(string)
	if !ok || tenantID == "" {
		// This should ideally not happen if the interceptor ran correctly
		return "", status.Error(codes.Internal, "tenant ID not found in context")
	}
	return tenantID, nil
}

// Example Tenant Validator (replace with your actual logic)
func SimpleTenantValidator(tenantID string) bool {
	// In a real app, query a DB table `tenants` or check a cached list
	allowedTenants := map[string]bool{"artist1": true, "studioxyz": true, "test": true}
	fmt.Printf("Validating tenant: %s\n", tenantID) // Log validation
	return allowedTenants[tenantID]
}
