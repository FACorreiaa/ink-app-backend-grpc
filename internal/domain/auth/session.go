package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// TenantContextKey is the key used to store tenant information in context
type TenantContextKey struct{}

// SessionManager handles user authentication sessions
type SessionManager struct {
	DBManager    *config.TenantDBManager
	RedisManager *config.TenantRedisManager
}

func NewSessionManager(dbManager *config.TenantDBManager, redis *config.TenantRedisManager) *SessionManager {
	return &SessionManager{
		DBManager:    dbManager,
		RedisManager: redis,
	}
}

// generateSessionKey creates a Redis key with tenant prefix for proper isolation
func generateSessionKey(tenant, sessionID string) string {
	return fmt.Sprintf("session:%s:%s", tenant, sessionID)
}

// GenerateTokens creates JWT access and refresh tokens for a user
func GenerateTokens(userID, tenant, role string) (accessToken string, refreshToken string, err error) {
	// Access Token with tenant information
	accessClaims := &domain.Claims{
		UserID: userID,
		Tenant: tenant, // Add tenant to claims
		Role:   role,
		Scope:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(domain.JwtSecretKey)
	if err != nil {
		return "", "", err
	}

	// Refresh Token with tenant information
	refreshClaims := &domain.Claims{
		UserID: userID,
		Tenant: tenant, // Add tenant to claims
		Scope:  "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(domain.JwtRefreshSecretKey)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// SignIn authenticates a user and creates a session
func (s *SessionManager) SignIn(ctx context.Context, tenant, email, password string) (string, error) {
	// Validate tenant
	if tenant == "" {
		return "", errors.New("tenant subdomain is required")
	}

	// Get tenant-specific database pool
	pool, err := s.DBManager.GetTenantDB(tenant)
	if err != nil {
		return "", fmt.Errorf("invalid tenant: %w", err)
	}

	redis, err := s.RedisManager.GetTenantRedis(tenant)
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

	// Generate tokens
	// accessToken, refreshToken, err := GenerateTokens(user.ID, tenant, user.Role)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to generate tokens: %w", err)
	// }

	// Create session
	sessionID := uuid.NewString()
	session := UserSession{
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
	err = s.Set(ctx, sessionKey, string(jsonData), 24*time.Hour).Err()
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
func (s *SessionManager) SignOut(ctx context.Context, tenant, sessionID string) error {
	// Delete from Redis
	sessionKey := generateSessionKey(tenant, sessionID)

	// Get tenant DB to update session record
	pool, err := s.DBManager.GetTenantDB(tenant)
	if err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	redis, err := s.RedisManager.GetTenantRedis(tenant)
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

// Set wraps Redis SET operation
func (s *SessionManager) Set(ctx context.Context, key string, value string, expiration time.Duration) (cmd *redis.StatusCmd) {
	// Get tenant from context or key
	tenant := extractTenantFromKey(key)
	redis, err := s.RedisManager.GetTenantRedis(tenant)
	if err != nil {
		// Create a StatusCmd with an error
		cmd.SetErr(fmt.Errorf("invalid tenant: %w", err))
		return cmd
	}
	// Perform the SET operation
	return redis.Set(ctx, key, value, expiration)
}

// extractTenantFromKey parses tenant from session key
func extractTenantFromKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// Fix the GetUserSession to use tenant
func (s *SessionManager) GetUserSession(ctx context.Context, tenant, userID string) (string, error) {
	redis, err := s.RedisManager.GetTenantRedis(tenant)
	if err != nil {
		return "", fmt.Errorf("invalid tenant: %w", err)
	}

	key := fmt.Sprintf("%s:user:%s", tenant, userID)
	return redis.Get(ctx, key).Result()
}

// GetSession retrieves a user session
func (s *SessionManager) GetSession(ctx context.Context, tenant, sessionID string) (*UserSession, error) {
	// Try to get from Redis first
	sessionKey := generateSessionKey(tenant, sessionID)

	redis, error := s.RedisManager.GetTenantRedis(tenant)
	if error != nil {
		return nil, fmt.Errorf("invalid tenant: %w", error)
	}
	data, err := redis.Get(ctx, sessionKey).Result()
	if err != nil {
		// If not in Redis, check if it's in DB (might have been evicted from cache)
		pool, err := s.DBManager.GetTenantDB(tenant)
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
		session := &UserSession{
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
	var userSession UserSession
	err = json.Unmarshal([]byte(data), &userSession)
	if err != nil {
		return nil, errors.New("invalid session data")
	}

	return &userSession, nil
}

// DELETE REFACTOR

// TenantMiddleware extracts tenant information from the request
// func TenantMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// Extract tenant from subdomain
// 		host := r.Host
// 		subdomain := strings.Split(host, ".")[0]

// 		// For local development, also check headers
// 		tenantHeader := r.Header.Get("X-Tenant")
// 		if tenantHeader != "" {
// 			subdomain = tenantHeader
// 		}

// 		if subdomain == "" {
// 			http.Error(w, "Tenant not specified", http.StatusBadRequest)
// 			return
// 		}

// 		// Add tenant to context
// 		ctx := context.WithValue(r.Context(), TenantContextKey{}, subdomain)
// 		next.ServeHTTP(w, r.WithContext(ctx))
// 	})
// }

// // SessionMiddleware validates the session and adds user info to context
// func SessionMiddleware(sessionManager *SessionManager) func(next http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			// Extract tenant from context (added by TenantMiddleware)
// 			tenantVal := r.Context().Value(TenantContextKey{})
// 			tenant, ok := tenantVal.(string)
// 			if !ok || tenant == "" {
// 				http.Error(w, "Tenant not found in context", http.StatusBadRequest)
// 				return
// 			}

// 			// Extract session token
// 			authHeader := r.Header.Get("Authorization")
// 			if authHeader == "" || len(authHeader) < 8 || !strings.HasPrefix(authHeader, "Bearer ") {
// 				http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
// 				return
// 			}

// 			sessionID := authHeader[7:] // Remove "Bearer " prefix

// 			// Get session
// 			userSession, err := sessionManager.GetSession(r.Context(), tenant, sessionID)
// 			if err != nil {
// 				http.Error(w, "Invalid session", http.StatusUnauthorized)
// 				return
// 			}

// 			// Add session info to context
// 			ctx := context.WithValue(r.Context(), SessionManagerKey{}, userSession)

// 			// Continue to next handler
// 			next.ServeHTTP(w, r.WithContext(ctx))
// 		})
// 	}
// }

// RefreshSession extends a session's validity
// func (s *SessionManager) RefreshSession(ctx context.Context, tenant, sessionID string) error {
// 	sessionKey := generateSessionKey(tenant, sessionID)

// 	// Check if session exists
// 	exists, err := s.Redis.Exists(ctx, sessionKey).Result()
// 	if err != nil || exists == 0 {
// 		return errors.New("session not found")
// 	}

// 	// Extend TTL in Redis
// 	err = s.Redis.Expire(ctx, sessionKey, 24*time.Hour).Err()
// 	if err != nil {
// 		return fmt.Errorf("failed to extend session: %w", err)
// 	}

// 	// Update expiration in database
// 	pool, err := s.DBManager.GetTenantDB(tenant)
// 	if err != nil {
// 		return fmt.Errorf("invalid tenant: %w", err)
// 	}

// 	_, err = pool.Exec(ctx,
// 		"UPDATE sessions SET expires_at = $1 WHERE session_id = $2",
// 		time.Now().Add(24*time.Hour), sessionID)
// 	if err != nil {
// 		// Log but don't fail if DB update fails as Redis is the primary session store
// 		fmt.Printf("Warning: Failed to update session expiration in database: %v\n", err)
// 	}

// 	return nil
// }
