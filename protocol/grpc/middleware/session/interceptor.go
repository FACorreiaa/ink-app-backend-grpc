package session

import (
	"context"
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
)

// TenantContextKey is the key used to store tenant information in context
type TenantContextKey struct{}

// InterceptorSession returns a gRPC interceptor that validates auth tokens and extracts tenant information
func InterceptorSession() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// List of methods that don't require authentication
		unauthenticatedMethods := map[string]bool{
			"/inkMe.auth.Auth/Register":    true,
			"/inkMe.auth.Auth/Login":       true,
			"/inkMe.auth.Auth/GetAllUsers": true,
		}

		// Extract tenant from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing context metadata")
		}

		// Extract tenant from headers (similar to subdomain in HTTP)
		var tenant string
		if tenantHeaders := md.Get("x-tenant"); len(tenantHeaders) > 0 {
			tenant = tenantHeaders[0]
		}

		// For methods that don't require auth, we still want tenant context
		// but we don't need to validate a token
		if tenant == "" {
			return nil, status.Error(codes.InvalidArgument, "tenant not specified")
		}

		// Add tenant to context
		ctx = context.WithValue(ctx, TenantContextKey{}, tenant)

		// For methods that don't require authentication, proceed with tenant context
		if unauthenticatedMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// For authenticated methods, validate the token
		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		tokenString := authHeader[0]
		// Support both with and without Bearer prefix
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = tokenString[7:]
		}

		// Parse and validate the token
		claims := &domain.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return domain.JwtSecretKey, nil
		})

		if err != nil || !token.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		// Validate tenant in token matches tenant in request
		if claims.Tenant != tenant {
			return nil, status.Error(codes.PermissionDenied, "token tenant mismatch")
		}

		// Add user info to context
		ctx = context.WithValue(ctx, "userID", claims.UserID)
		ctx = context.WithValue(ctx, "role", claims.Role)
		ctx = context.WithValue(ctx, "tenant", claims.Tenant)
		ctx = context.WithValue(ctx, "scope", claims.Scope)

		// Call the handler with the enriched context
		return handler(ctx, req)
	}
}

// GetTenantFromContext extracts the tenant from the context
func GetTenantFromContext(ctx context.Context) (string, bool) {
	tenant, ok := ctx.Value(TenantContextKey{}).(string)
	return tenant, ok
}

// HasPermission checks if a user has a specific permission
func HasPermission(userPermissions []string, requiredPermission string) bool {
	for _, perm := range userPermissions {
		if perm == requiredPermission {
			return true
		}
	}
	return false
}

// TenantInterceptor extracts tenant information and routes to the appropriate repository
// func TenantInterceptor(repoManager *auth.TenantRepositoryManager) grpc.UnaryServerInterceptor {
// 	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
// 		// Extract tenant from metadata
// 		md, ok := metadata.FromIncomingContext(ctx)
// 		if !ok {
// 			return nil, status.Error(codes.InvalidArgument, "missing metadata")
// 		}

// 		subdomains := md.Get("subdomain")
// 		if len(subdomains) == 0 {
// 			return nil, status.Error(codes.InvalidArgument, "missing subdomain")
// 		}
// 		tenantSubdomain := subdomains[0]

// 		// Get the tenant-specific repository
// 		repo, err := repoManager.GetRepository(tenantSubdomain)
// 		if err != nil {
// 			return nil, status.Errorf(codes.NotFound, "tenant not found: %v", err)
// 		}

// 		// Add the repository to the context
// 		newCtx := context.WithValue(ctx, repositoryKey, repo)

// 		// Continue with the handler
// 		return handler(newCtx, req)
// 	}
// }

// func AuthInterceptor(sessionManager *SessionManager, dbManager *config.TenantDBManager) grpc.UnaryServerInterceptor {
// 	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
// 		// Skip auth check for login and health check methods
// 		if info.FullMethod == "/auth.Auth/Login" || strings.Contains(info.FullMethod, "Health") {
// 			return handler(ctx, req)
// 		}

// 		// Extract tenant from metadata
// 		md, ok := metadata.FromIncomingContext(ctx)
// 		if !ok {
// 			return nil, status.Errorf(codes.Unauthenticated, "Metadata not provided")
// 		}

// 		// Get tenant from metadata
// 		tenantValues := md.Get("x-tenant")
// 		if len(tenantValues) == 0 {
// 			return nil, status.Errorf(codes.Unauthenticated, "Tenant not provided")
// 		}
// 		tenant := tenantValues[0]

// 		// Extract auth token
// 		authValues := md.Get("authorization")
// 		if len(authValues) == 0 {
// 			return nil, status.Errorf(codes.Unauthenticated, "Authorization not provided")
// 		}

// 		// Parse and validate JWT token
// 		tokenString := strings.TrimPrefix(authValues[0], "Bearer ")
// 		token, err := jwt.ParseWithClaims(tokenString, &domain.Claims{}, func(token *jwt.Token) (interface{}, error) {
// 			return domain.JwtSecretKey, nil
// 		})
// 		if err != nil {
// 			return nil, status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
// 		}

// 		claims, ok := token.Claims.(*domain.Claims)
// 		if !ok || !token.Valid || claims.Scope != "access" || claims.Tenant != tenant {
// 			return nil, status.Errorf(codes.Unauthenticated, "Invalid token claims")
// 		}

// 		// Get session info if needed
// 		sessionValues := md.Get("x-session-id")
// 		if len(sessionValues) > 0 {
// 			sessionID := sessionValues[0]
// 			userSession, err := sessionManager.GetSession(ctx, tenant, sessionID)
// 			if err != nil {
// 				return nil, status.Errorf(codes.Unauthenticated, "Invalid session: %v", err)
// 			}

// 			// Add session to context
// 			ctx = context.WithValue(ctx, SessionManagerKey{}, userSession)
// 		}

// 		// Add tenant to context
// 		ctx = context.WithValue(ctx, TenantContextKey{}, tenant)

// 		// Process request with enriched context
// 		return handler(ctx, req)
// 	}
// }
