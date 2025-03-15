package auth

// import (
// 	"context"
// 	"database/sql"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"sync"
// 	"time"

// 	//upb "github.com/FACorreiaa/ink-app-backend-protos/modules/auth/generated"
// 	"github.com/golang-jwt/jwt/v5"
// 	"github.com/jackc/pgx/v5/pgxpool"
// 	"github.com/redis/go-redis/v9"
// 	"golang.org/x/crypto/bcrypt"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/metadata"
// 	"google.golang.org/grpc/status"

// 	"github.com/FACorreiaa/ink-app-backend-grpc/config"
// 	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
// )

// type RepositoryAuth struct {
// 	upb.UnimplementedAuthServer
// 	DBManager       *config.TenantDBManager
// 	RedisManager    *config.TenantRedisManager
// 	redis           *redis.Client
// 	Session         *SessionManager
// 	TenantDB        *pgxpool.Pool // Store the tenant-specific DB connection
// 	TenantRedis     *redis.Client // Store the tenant-specific Redis connection
// 	TenantSubdomain string        // Store the tenant identifier
// }

// // NewRepository creates a new AuthService
// func NewRepository(
// 	dbManager *config.TenantDBManager,
// 	redisManager *config.TenantRedisManager,
// 	sessionManager *SessionManager,
// 	tenantSubdomain string,
// ) (*RepositoryAuth, error) {
// 	// Initialize the repository
// 	repo := &RepositoryAuth{
// 		DBManager:       dbManager,
// 		RedisManager:    redisManager,
// 		Session:         sessionManager,
// 		TenantSubdomain: tenantSubdomain,
// 	}

// 	// Get the tenant DB connection once
// 	pool, err := dbManager.GetTenantDB(tenantSubdomain)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid tenant: %w", err)
// 	}
// 	repo.TenantDB = pool

// 	// Get the tenant Redis connection once
// 	redisClient, err := redisManager.GetTenantRedis(tenantSubdomain)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get tenant redis: %w", err)
// 	}
// 	repo.TenantRedis = redisClient

// 	return repo, nil
// }

// type TenantRepositoryManager struct {
// 	dbManager      *config.TenantDBManager
// 	redisManager   *config.TenantRedisManager
// 	sessionManager *SessionManager
// 	repos          map[string]*RepositoryAuth // Map of subdomain to RepositoryAuth
// 	mu             sync.RWMutex               // For thread-safe access
// }

// func NewTenantRepositoryManager(
// 	dbManager *config.TenantDBManager,
// 	redisManager *config.TenantRedisManager,
// 	sessionManager *SessionManager,
// ) *TenantRepositoryManager {
// 	return &TenantRepositoryManager{
// 		dbManager:      dbManager,
// 		redisManager:   redisManager,
// 		sessionManager: sessionManager,
// 		repos:          make(map[string]*RepositoryAuth),
// 	}
// }

// // TenantServiceManager acts as a proxy to the tenant-specific repository
// type TenantServiceManager struct {
// 	upb.UnimplementedAuthServer
// 	repoManager *TenantRepositoryManager
// }

// // NewTenantServiceManager creates a tenant service manager
// func NewTenantServiceManager(
// 	ctx context.Context,
// 	repoManager *TenantRepositoryManager,
// ) *TenantServiceManager {
// 	return &TenantServiceManager{
// 		repoManager: repoManager,
// 	}
// }

// func (r *RepositoryAuth) Register(ctx context.Context, req *upb.RegisterRequest) (*upb.RegisterResponse, error) {
// 	// md, ok := metadata.FromIncomingContext(ctx)
// 	// if !ok {
// 	// 	return nil, status.Error(codes.InvalidArgument, "missing metadata")
// 	// }

// 	// subdomains := md.Get("subdomain") // Assume subdomain is passed in metadata
// 	// if len(subdomains) == 0 {
// 	// 	return nil, status.Error(codes.InvalidArgument, "missing subdomain")
// 	// }
// 	// tenantSubdomain := subdomains[0]

// 	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
// 	}

// 	_, err = r.TenantDB.Exec(ctx, `INSERT INTO "users" (studio_id, username, email, hashed_password, role) VALUES ($1, $2, $3, $4, $5)`,
// 		req.StudioId, req.Username, req.Email, hashedPassword, req.Role)
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to insert user: %v", err)
// 	}

// 	return &upb.RegisterResponse{Message: "Registration successful"}, nil
// }

// func (r *RepositoryAuth) RefreshToken(ctx context.Context, req *upb.RefreshTokenRequest) (*upb.TokenResponse, error) {
// 	claims := &domain.Claims{}

// 	tenant, err := extractTenantFromContext(ctx)
// 	if err != nil {
// 		return nil, status.Errorf(codes.InvalidArgument, "Tenant not provided: %v", err)
// 	}

// 	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
// 		return domain.JwtRefreshSecretKey, nil
// 	})
// 	if err != nil || !token.Valid || claims.Scope != "refresh" {
// 		return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
// 	}

// 	claims, ok := token.Claims.(*domain.Claims)
// 	if !ok || !token.Valid || claims.Scope != "refresh" || claims.Tenant != tenant {
// 		return nil, status.Errorf(codes.Unauthenticated, "Invalid refresh token claims")
// 	}

// 	// Generate new tokens
// 	accessToken, refreshToken, err := GenerateTokens(claims.UserID, claims.Tenant, claims.Role)
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to generate new tokens: %v", err)
// 	}

// 	return &upb.TokenResponse{
// 		AccessToken:  accessToken,
// 		RefreshToken: refreshToken,
// 	}, nil
// }

// func extractTenantFromContext(ctx context.Context) (string, error) {
// 	md, ok := metadata.FromIncomingContext(ctx)
// 	if !ok {
// 		return "", errors.New("metadata not provided")
// 	}

// 	tenantValues := md.Get("x-tenant")
// 	if len(tenantValues) == 0 {
// 		return "", errors.New("tenant not provided")
// 	}

// 	return tenantValues[0], nil
// }

// func (r *RepositoryAuth) Login(ctx context.Context, req *upb.LoginRequest) (*upb.LoginResponse, error) {
// 	var user User
// 	err := r.TenantDB.QueryRow(ctx, `SELECT id, hashed_password, email FROM "users" WHERE username=$1`, req.Username).Scan(
// 		&user.ID, &user.Password, &user.Email)
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to query user: %v", err)
// 	}

// 	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
// 	if err != nil {
// 		return nil, status.Errorf(codes.Unauthenticated, "invalid password")
// 	}

// 	expirationTime := time.Now().Add(24 * time.Hour)
// 	claims := &domain.Claims{
// 		UserID: user.ID,
// 		Role:   "user",
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			ExpiresAt: jwt.NewNumericDate(expirationTime),
// 		},
// 	}

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	tokenString, err := token.SignedString(domain.JwtSecretKey)

// 	if err != nil {
// 		return nil, status.Error(codes.Internal, "could not create token")
// 	}

// 	//sessionID, err := a.sessionManager.GenerateSession(UserSession{
// 	//	ID:       userID,
// 	//	Email:    req.Email,
// 	//	Username: req.Username,
// 	//})

// 	//err = a.redis.Set(ctx, req.Username, sessionID, 0).Err()
// 	//if err != nil {
// 	//	return nil, err
// 	//}

// 	return &upb.LoginResponse{Token: tokenString, Message: "Login successful!"}, nil
// }

// func (r *RepositoryAuth) Logout(ctx context.Context, req *upb.NilReq) (*upb.NilRes, error) {
// 	md, ok := metadata.FromIncomingContext(ctx)
// 	if !ok {
// 		return nil, errors.New("unable to retrieve metadata")
// 	}

// 	authHeader := md["authorization"]
// 	if len(authHeader) != 1 {
// 		return nil, errors.New("invalid authorization header")
// 	}

// 	token := authHeader[0]
// 	sessionData, err := r.redis.Get(ctx, token).Result()
// 	if err != nil {
// 		return nil, errors.New("invalid or expired token")
// 	}

// 	var session UserSession
// 	err = json.Unmarshal([]byte(sessionData), &session)
// 	if err != nil {
// 		return nil, errors.New("invalid or expired token")
// 	}

// 	//username := session.Username
// 	//
// 	//err := a.sessionManager.SignOut(username)
// 	//if err != nil {
// 	//	return nil, err
// 	//}

// 	err = r.redis.Del(ctx, token).Err()
// 	if err != nil {
// 		return nil, errors.New("delete item")
// 	}

// 	return &upb.NilRes{}, nil
// }

// func (r *RepositoryAuth) ChangePassword(ctx context.Context, req *upb.ChangePasswordRequest) (*upb.ChangePasswordResponse, error) {
// 	var passwordHash string
// 	err := r.TenantDB.QueryRow(ctx, `SELECT hashed_password FROM "users" WHERE username=$1`, req.Username).Scan(&passwordHash)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}

// 	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.OldPassword))
// 	if err != nil {
// 		return nil, errors.New("invalid old password")
// 	}

// 	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
// 	if err != nil {
// 		return nil, err
// 	}

// 	_, err = r.TenantDB.Exec(ctx, `UPDATE "users" SET hashed_password=$1, updated_at=now() WHERE username=$2`, newHashedPassword, req.Username)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &upb.ChangePasswordResponse{Message: "Password changed successfully"}, nil
// }

// func (r *RepositoryAuth) ChangeEmail(ctx context.Context, req *upb.ChangeEmailRequest) (*upb.ChangeEmailResponse, error) {
// 	var passwordHash string
// 	err := r.TenantDB.QueryRow(ctx, `SELECT hashed_password FROM "users" WHERE username=$1`, req.Username).Scan(&passwordHash)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}

// 	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
// 	if err != nil {
// 		return nil, errors.New("invalid credentials")
// 	}

// 	_, err = r.TenantDB.Exec(ctx, `UPDATE "users" SET email=$1, updated_at=now() WHERE username=$2`, req.NewEmail, req.Username)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &upb.ChangeEmailResponse{Message: "Email changed successfully"}, nil
// }

// func (r *RepositoryAuth) GetAllUsers(ctx context.Context) (*upb.GetAllUsersResponse, error) {
// 	rows, err := r.TenantDB.Query(ctx, `SELECT id, username, email, role, created_at, updated_at FROM "users"`)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var users []*upb.User
// 	for rows.Next() {
// 		select {
// 		case <-ctx.Done():
// 			return nil, status.Errorf(codes.DeadlineExceeded, "operation cancelled: %v", ctx.Err())
// 		default:
// 		}

// 		var id, username, email, roleStr string
// 		var createdAt time.Time
// 		var updatedAt *time.Time

// 		err := rows.Scan(&id, &username, &email, &roleStr, &createdAt, &updatedAt)
// 		if err != nil {
// 			return nil, err
// 		}

// 		var role upb.User_Role
// 		switch roleStr {
// 		case "ADMIN":
// 			role = upb.User_ADMIN
// 		case "MODERATOR":
// 			role = upb.User_MODERATOR
// 		case "USER":
// 			role = upb.User_USER
// 		default:
// 			role = upb.User_ROLE_UNSPECIFIED
// 		}

// 		var updatedAtStr string
// 		if updatedAt != nil {
// 			updatedAtStr = updatedAt.Format(time.RFC3339)
// 		}

// 		users = append(users, &upb.User{
// 			Id:        id,
// 			Username:  username,
// 			Email:     email,
// 			Role:      role,
// 			CreatedAt: createdAt.Format(time.RFC3339),
// 			UpdatedAt: updatedAtStr,
// 		})
// 	}
// 	if err := rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return &upb.GetAllUsersResponse{Users: users}, nil
// }

// func (r *RepositoryAuth) GetUserByID(ctx context.Context, req *upb.GetUserByIDRequest) (*upb.GetUserByIDResponse, error) {
// 	var u upb.User
// 	var createdAt time.Time
// 	var updatedAt *time.Time

// 	err := r.TenantDB.QueryRow(ctx, `
// 			SELECT u.id, u.username, u.email, u.created_at, u.updated_at
// 			FROM "users" u
// 			WHERE id = $1`, req.Id).Scan(
// 		&u.Id, &u.Username, &u.Email, &createdAt, &updatedAt)

// 	if err != nil {
// 		if errors.Is(err, sql.ErrNoRows) {
// 			return nil, errors.New("user not found")
// 		}
// 		return nil, fmt.Errorf("error fetching user: %v", err)
// 	}
// 	if updatedAt != nil {
// 		u.UpdatedAt = updatedAt.Format(time.RFC3339)
// 	}

// 	u.CreatedAt = createdAt.Format(time.RFC3339)

// 	return &upb.GetUserByIDResponse{User: &u}, nil
// }

// func (r *RepositoryAuth) DeleteUser(ctx context.Context, req *upb.DeleteUserRequest) (*upb.DeleteUserResponse, error) {
// 	// Execute the delete query
// 	commandTag, err := r.TenantDB.Exec(ctx, `DELETE FROM "users" WHERE user = $1`, req.Id)
// 	if err != nil {
// 		return nil, fmt.Errorf("error deleting user: %v", err)
// 	}

// 	// Check if any row was deleted
// 	if commandTag.RowsAffected() == 0 {
// 		return nil, errors.New("user not found")
// 	}

// 	return &upb.DeleteUserResponse{Message: "user deleted successfully"}, nil
// }

// func (r *RepositoryAuth) UpdateUser(ctx context.Context, req *upb.UpdateUserRequest) (*upb.UpdateUserResponse, error) {
// 	// Execute the update query
// 	commandTag, err := r.TenantDB.Exec(ctx, `
// 		UPDATE "users"
// 		SET username = $1, email = $2, updated_at = NOW()
// 		WHERE id = $3`,
// 		req.User.Username, req.User.Email, req.User.Id)
// 	if err != nil {
// 		return nil, fmt.Errorf("error updating user: %v", err)
// 	}

// 	// Check if any row was updated
// 	if commandTag.RowsAffected() == 0 {
// 		return nil, errors.New("user not found")
// 	}

// 	return &upb.UpdateUserResponse{Message: "user updated successfully"}, nil
// }

// // InsertUser change this later

// func (r *RepositoryAuth) InsertUser(ctx context.Context, req *upb.InsertUserRequest) (*upb.InsertUserResponse, error) {
// 	// Insert the new user
// 	_, err := r.TenantDB.Exec(ctx, `
// 		INSERT INTO "users" (id, username, email, hashed_password, role, created_at, updated_at)
// 		VALUES ($1, $2, $3, $4, NOW(), NOW())`,
// 		req.User.Id, req.User.Username, req.User.Email, req.User.PasswordHash, req.User.IsAdmin)
// 	if err != nil {
// 		return nil, fmt.Errorf("error inserting user: %v", err)
// 	}

// 	return &upb.InsertUserResponse{Message: "user inserted successfully"}, nil
// }
