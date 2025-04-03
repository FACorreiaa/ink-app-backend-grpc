package studio

import (
	"errors"
	"strings"

	ups "github.com/FACorreiaa/ink-app-backend-protos/modules/studio/generated"
)

func validateLoginRequest(req *ups.LoginRequest) error {
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
