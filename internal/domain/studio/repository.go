package studio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/golang-jwt/jwt/v5"
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

// Claims defines JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Tenant   string `json:"tenant"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// generateAccessToken creates a JWT access token
func generateAccessToken(userID, username, email, tenant, role string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Tenant:   tenant,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // Short-lived
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(domain.JwtSecretKey) // Assume JwtSecretKey is a global secret
}

// generateRefreshToken creates a random refresh token
func generateRefreshToken() string {
	return uuid.NewString()
}

// generateSessionKey creates a Redis key with tenant prefix for proper isolation
func generateSessionKey(tenant, sessionID string) string {
	return fmt.Sprintf("session:%s:%s", tenant, sessionID)
}

// Login authenticates a user and returns an access token
func (r *StudioAuthRepository) Login(ctx context.Context, tenant, email, password string) (string, error) {
	if tenant == "" {
		return "", errors.New("tenant subdomain is required")
	}

	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return "", fmt.Errorf("invalid tenant: %w", err)
	}

	var user struct {
		ID       string
		Username string
		Email    string
		Password string
		Role     string
	}

	err = pool.QueryRow(ctx,
		"SELECT id, username, email, hashed_password, role FROM users WHERE email = $1",
		email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate access token
	accessToken, err := generateAccessToken(user.ID, user.Username, user.Email, tenant, user.Role)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate and store refresh token
	refreshToken := generateRefreshToken()
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	_, err = pool.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
		user.ID, refreshToken, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return accessToken, nil // Note: Refresh token not returned due to proto limitation
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

func (r *StudioAuthRepository) RefreshSession(ctx context.Context, tenant, refreshToken string) (string, string, error) {
	if tenant == "" {
		return "", "", errors.New("tenant subdomain is required")
	}

	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return "", "", fmt.Errorf("invalid tenant: %w", err)
	}

	var userID string
	var expiresAt time.Time
	var invalidatedAt *time.Time
	err = pool.QueryRow(ctx,
		"SELECT user_id, expires_at, invalidated_at FROM refresh_tokens WHERE token = $1",
		refreshToken).Scan(&userID, &expiresAt, &invalidatedAt)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	if time.Now().After(expiresAt) || invalidatedAt != nil {
		return "", "", errors.New("refresh token expired or invalidated")
	}

	var username, email, role string
	err = pool.QueryRow(ctx,
		"SELECT username, email, role FROM users WHERE id = $1",
		userID).Scan(&username, &email, &role)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	newAccessToken, err := generateAccessToken(userID, username, email, tenant, role)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken := generateRefreshToken()
	newExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = pool.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
		userID, newRefreshToken, newExpiresAt)
	if err != nil {
		return "", "", fmt.Errorf("failed to store new refresh token: %w", err)
	}

	_, err = pool.Exec(ctx,
		"UPDATE refresh_tokens SET invalidated_at = $1 WHERE token = $2",
		time.Now(), refreshToken)
	if err != nil {
		fmt.Printf("Warning: failed to invalidate old refresh token: %v\n", err)
	}

	return newAccessToken, newRefreshToken, nil
}

// Register creates a new user in the tenant's database
func (r *StudioAuthRepository) Register(ctx context.Context, tenant, username, email, password, role string) error {
	if tenant == "" {
		return errors.New("tenant subdomain is required")
	}

	pool, err := r.DBManager.GetTenantDB(tenant)
	if err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	// Get studio_id (assuming one studio per tenant DB)
	var studioID string
	err = pool.QueryRow(ctx,
		"SELECT id FROM studios WHERE subdomain = $1", tenant).Scan(&studioID)
	if err != nil {
		return fmt.Errorf("studio not found: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = pool.Exec(ctx,
		"INSERT INTO users (studio_id, username, email, hashed_password, role, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		studioID, username, email, string(hashedPassword), role, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// ChangePassword updates a user's password
func (r *StudioAuthRepository) ChangePassword(ctx context.Context, tenant, email, oldPassword, newPassword string) error {
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

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(oldPassword))
	if err != nil {
		return errors.New("invalid old password")
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	_, err = pool.Exec(ctx,
		"UPDATE users SET hashed_password = $1, updated_at = $2 WHERE id = $3",
		string(newHashedPassword), time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all refresh tokens
	_, err = pool.Exec(ctx,
		"UPDATE refresh_tokens SET invalidated_at = $1 WHERE user_id = $2 AND invalidated_at IS NULL",
		time.Now(), userID)
	if err != nil {
		fmt.Printf("Warning: failed to invalidate refresh tokens: %v\n", err)
	}

	return nil
}

// ChangeEmail updates a user's email
func (r *StudioAuthRepository) ChangeEmail(ctx context.Context, tenant, email, password, newEmail string) error {
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
