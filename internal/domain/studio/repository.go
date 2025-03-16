package studio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// StudioAuthRepository handles database operations for studio authentication
type StudioAuthRepository struct {
	DBManager    *config.TenantDBManager
	RedisManager *config.TenantRedisManager
}

// NewStudioAuthRepository creates a new StudioAuthRepository
func NewStudioAuthRepository(dbManager *config.TenantDBManager, redisManager *config.TenantRedisManager) *StudioAuthRepository {
	return &StudioAuthRepository{
		DBManager:    dbManager,
		RedisManager: redisManager,
	}
}

// generateSessionKey creates a Redis key with tenant prefix for proper isolation
func generateSessionKey(tenant, sessionID string) string {
	return fmt.Sprintf("session:%s:%s", tenant, sessionID)
}

// SignIn authenticates a user and creates a session
func (r *StudioAuthRepository) Login(ctx context.Context, tenant, email, password string) (string, error) {
	// Validate tenant
	if tenant == "" {
		return "", errors.New("tenant subdomain is required")
	}

	// Get tenant-specific database pool
	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return "", fmt.Errorf("invalid tenant: %w", err)
	}

	redis, err := r.RedisManager.GetTenantRedis(tenant)
	if err != nil {
		return "", fmt.Errorf("invalid tenant: %w", err)
	}

	// Find user in the tenant's database
	var user struct {
		ID       string
		Username string
		Email    string
		Password string
		Role     string
	}

	err = pool.QueryRow(ctx,
		"SELECT id, username, email, password, role FROM users WHERE email = $1",
		email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// Create session
	sessionID := uuid.NewString()
	session := domain.StudioSession{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Tenant:   tenant,
	}

	// Serialize session data
	jsonData, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store in Redis with tenant-specific key
	sessionKey := generateSessionKey(tenant, sessionID)
	err = redis.Set(ctx, sessionKey, string(jsonData), 24*time.Hour).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	// Store session in database for audit/tracking
	_, err = pool.Exec(ctx,
		"INSERT INTO sessions (session_id, user_id, created_at, expires_at) VALUES ($1, $2, $3, $4)",
		sessionID, user.ID, time.Now(), time.Now().Add(24*time.Hour))
	if err != nil {
		// If DB insert fails, clean up Redis
		redis.Del(ctx, sessionKey)
		return "", fmt.Errorf("failed to record session: %w", err)
	}

	return sessionID, nil
}

// SignOut invalidates a user session
func (r *StudioAuthRepository) Logout(ctx context.Context, tenant, sessionID string) error {
	// Delete from Redis
	sessionKey := generateSessionKey(tenant, sessionID)

	// Get tenant DB to update session record
	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	redis, err := r.RedisManager.GetTenantRedis(tenant)
	if err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	err = redis.Del(ctx, sessionKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session from cache: %w", err)
	}

	// Mark session as invalidated in database
	_, err = pool.Exec(ctx,
		"UPDATE sessions SET invalidated_at = $1 WHERE session_id = $2",
		time.Now(), sessionID)
	// We don't return this error as the primary operation (Redis delete) succeeded
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: Failed to update session record: %v\n", err)
	}

	return nil
}

// GetSession retrieves a user session
func (r *StudioAuthRepository) GetSession(ctx context.Context, tenant, sessionID string) (*domain.StudioSession, error) {
	// Try to get from Redis first
	sessionKey := generateSessionKey(tenant, sessionID)

	redis, err := r.RedisManager.GetTenantRedis(tenant)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant: %w", err)
	}

	data, err := redis.Get(ctx, sessionKey).Result()
	if err != nil {
		// If not in Redis, check if it's in DB (might have been evicted from cache)
		pool, err := r.DBManager.GetTenantDB(tenant)
		if err != nil {
			return nil, fmt.Errorf("invalid tenant: %w", err)
		}

		var userID string
		var expiresAt time.Time
		var invalidatedAt *time.Time

		err = pool.QueryRow(ctx,
			"SELECT user_id, expires_at, invalidated_at FROM sessions WHERE session_id = $1",
			sessionID).Scan(&userID, &expiresAt, &invalidatedAt)
		if err != nil {
			return nil, errors.New("session not found")
		}

		// Check if session is expired or invalidated
		if time.Now().After(expiresAt) || (invalidatedAt != nil) {
			return nil, errors.New("session expired or invalidated")
		}

		// Session exists in DB but not in Redis, need to fetch user details
		var user struct {
			Username string
			Email    string
		}

		err = pool.QueryRow(ctx,
			"SELECT username, email FROM users WHERE id = $1",
			userID).Scan(&user.Username, &user.Email)
		if err != nil {
			return nil, errors.New("user not found")
		}

		// Recreate session object
		session := &domain.StudioSession{
			ID:       userID,
			Username: user.Username,
			Email:    user.Email,
			Tenant:   tenant,
		}

		// Restore in Redis
		jsonData, _ := json.Marshal(session)
		redis.Set(ctx, sessionKey, string(jsonData), time.Until(expiresAt))

		return session, nil
	}

	// Session found in Redis, unmarshal
	var userSession domain.StudioSession
	err = json.Unmarshal([]byte(data), &userSession)
	if err != nil {
		return nil, errors.New("invalid session data")
	}

	return &userSession, nil
}

// GetUserByEmail retrieves a user by email
func (r *StudioAuthRepository) GetUserByEmail(ctx context.Context, tenant, email string) (string, string, string, error) {
	// Get tenant-specific database pool
	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid tenant: %w", err)
	}

	var id, username, role string
	err = pool.QueryRow(ctx,
		"SELECT id, username, role FROM users WHERE email = $1",
		email).Scan(&id, &username, &role)
	if err != nil {
		return "", "", "", fmt.Errorf("user not found: %w", err)
	}

	return id, username, role, nil
}

// ValidateCredentials validates user credentials
func (r *StudioAuthRepository) ValidateCredentials(ctx context.Context, tenant, email, password string) (bool, error) {
	// Get tenant-specific database pool
	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return false, fmt.Errorf("invalid tenant: %w", err)
	}

	var hashedPassword string
	err = pool.QueryRow(ctx,
		"SELECT password FROM users WHERE email = $1",
		email).Scan(&hashedPassword)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil, nil
}
