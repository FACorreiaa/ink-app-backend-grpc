package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	ups "github.com/FACorreiaa/ink-app-backend-protos/modules/studio/generated"
)

func ValidateLoginRequest(req *ups.LoginRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}
	if req.Email == "" {
		return errors.New("email is required")
	}
	// Basic email format check (can use regex for more robustness)
	if !strings.Contains(req.Email, "@") {
		return errors.New("invalid email format")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	// Could add password length check here too if desired
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

type SessionManagerKey struct{}

// extractTenantFromContext extracts tenant from gRPC metadata
func ExtractTenantFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no metadata provided")
	}

	tenantValues := md.Get("X-Tenant")
	if len(tenantValues) > 0 && tenantValues[0] != "" {
		return tenantValues[0], nil
	}

	fmt.Printf("Metadata: %v\n", md)

	hostValues := md.Get(":authority")
	if len(hostValues) > 0 {
		host := hostValues[0]
		parts := strings.Split(host, ".")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0], nil
		}
	}

	return "", status.Error(codes.Unauthenticated, "tenant not specified")
}
