package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	DBManager    *config.TenantDBManager
	RedisManager *config.TenantRedisManager
}

// NewRepository creates a new AuthService
func NewUserRepository(dbManager *config.TenantDBManager, redisManager *config.TenantRedisManager) *UserRepository {
	return &UserRepository{
		DBManager:    dbManager,
		RedisManager: redisManager,
	}
}

// ChangePassword implements domain.UserRepository.
func (r *UserRepository) ChangePassword(ctx context.Context, tenant string, email string, oldPassword string, newPassword string) error {
	panic("unimplemented")
}

// DeleteUser implements domain.UserRepository.
func (r *UserRepository) DeleteUser(ctx context.Context, tenant string, userID string) error {
	panic("unimplemented")
}

// GetAllUsers implements domain.UserRepository.
func (r *UserRepository) GetAllUsers(ctx context.Context, tenant string) ([]*domain.User, error) {
	panic("unimplemented")
}

// GetUserByEmail implements domain.UserRepository.
func (r *UserRepository) GetUserByEmail(ctx context.Context, tenant string, email string) (string, string, string, error) {
	panic("unimplemented")
}

// GetUserByID implements domain.UserRepository.
func (r *UserRepository) GetUserByID(ctx context.Context, tenant string, userID string) (*domain.User, error) {
	panic("unimplemented")
}

// InsertUser implements domain.UserRepository.
func (r *UserRepository) InsertUser(ctx context.Context, tenant string, user *domain.User) error {
	panic("unimplemented")
}

// UpdateUser implements domain.UserRepository.
func (r *UserRepository) UpdateUser(ctx context.Context, tenant string, user *domain.User) error {
	panic("unimplemented")
}

func (r *UserRepository) ChangeEmail(ctx context.Context, tenant, email, password, newEmail string) error {
	if tenant == "" {
		return errors.New("tenant subdomain is required")
	}

	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	var userID, hashedPassword string
	err = pool.QueryRow(ctx,
		"SELECT id, hashed_password FROM users WHERE email = $1",
		email).Scan(&userID, &hashedPassword)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return errors.New("invalid credentials")
	}

	_, err = pool.Exec(ctx,
		"UPDATE users SET email = $1, updated_at = $2 WHERE id = $3",
		newEmail, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	return nil
}

// func (r *AuthRepository) GetAllUsers(ctx context.Context, tenant string) ([]*domain.User, error) {
// 	return nil, nil
// }

// func (r *AuthRepository) UpdateUser(ctx context.Context, tenant string, user *domain.User) error {
// 	return nil
// }

// func (r *AuthRepository) InsertUser(ctx context.Context, tenant string, user *domain.User) error {
// 	return nil
// }

// func (r *AuthRepository) DeleteUser(ctx context.Context, tenant, userID string) error {
// 	return nil
// }
