package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
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

// GetUserByID implements domain.UserRepository.
func (r *UserRepository) GetUserByID(ctx context.Context, tenant, id string) (user *domain.User, err error) {
	// Get tenant-specific database pool
	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant: %w", err)
	}

	var username, email, role string
	err = pool.QueryRow(ctx,
		"SELECT id, username, email, role FROM users WHERE id = $1",
		id).Scan(&id, &username, &email, &role)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	user = &domain.User{
		ID:       id,
		Username: username,
		Email:    email,
		Role:     role,
	}

	return user, nil
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

func (r *UserRepository) UpdateEmail(ctx context.Context, tenant, userID, newEmail string) error {
	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return fmt.Errorf("update email: invalid tenant: %w", err)
	}

	tag, err := pool.Exec(ctx,
		`UPDATE users SET email = $1, updated_at = $2 WHERE id = $3`,
		newEmail, time.Now(), userID)
	if err != nil {
		// ---> Check for unique constraint violation error specifically <---
		// Example for pgx:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 is unique_violation
			return fmt.Errorf("new email already exists: %w", err) // Or a custom domain error
		}
		return fmt.Errorf("update email: db update failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// This shouldn't happen if VerifyPassword passed, but good check
		return errors.New("user not found during update")
	}
	return nil
}

func (r *UserRepository) GetAllUsers(ctx context.Context, tenant string) ([]*domain.User, error) {
	// Get tenant-specific database pool
	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant: %w", err)
	}

	rows, err := pool.Query(ctx,
		"SELECT id, username, email, role FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		err = rows.Scan(&user.ID, &user.Username, &user.Email, &user.Role)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, tenant string, user *domain.User) error {
	return nil
}

func (r *UserRepository) InsertUser(ctx context.Context, tenant string, user *domain.User) error {
	if tenant == "" {
		return errors.New("tenant subdomain is required")
	}

	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	_, err = pool.Exec(ctx,
		"INSERT INTO users (id, username, email, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		user.ID, user.Username, user.Email, user.Role, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, tenant, userID string) error {
	if tenant == "" {
		return errors.New("tenant subdomain is required")
	}

	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	_, err = pool.Exec(ctx,
		"DELETE FROM users WHERE id = $1",
		userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, tenant, email string) (*domain.User, error) {
	if tenant == "" {
		return nil, errors.New("tenant subdomain is required")
	}

	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant: %w", err)
	}

	var user domain.User
	err = pool.QueryRow(ctx,
		"SELECT id, username, email, role FROM users WHERE email = $1",
		email).Scan(&user.ID, &user.Username, &user.Email, &user.Role)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) ChangePassword(ctx context.Context, tenant, email, oldPassword, newPassword string) error {
	return nil
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, tenant, username string) (*domain.User, error) {
	if tenant == "" {
		return nil, errors.New("tenant subdomain is required")
	}

	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant: %w", err)
	}

	var user domain.User
	err = pool.QueryRow(ctx,
		"SELECT id, username, email, role FROM users WHERE username = $1",
		username).Scan(&user.ID, &user.Username, &user.Email, &user.Role)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}
