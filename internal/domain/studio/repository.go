package studio

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
)

// StudioAuthRepository handles database operations for studio authentication
type StudioRepository struct {
	DBManager    *config.TenantDBManager
	RedisManager *config.TenantRedisManager
}

func (r *StudioRepository) CreateStudio(ctx context.Context, tenant string, studio *domain.Studio, owner *domain.OwnerInfo) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) UpdateStudio(ctx context.Context, tenant, studioID string, studio *domain.Studio, updateMask *fieldmaskpb.FieldMask) error {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) GetStudio(ctx context.Context, tenant, studioID string) (*domain.Studio, error) {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) ListStudios(ctx context.Context, tenant string, pageSize, pageNumber int32, filter, ownerID string) ([]*domain.Studio, int32, error) {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) AddStaffMember(ctx context.Context, tenant, studioID, userID, role string, permissions []string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) UpdateStaffMember(ctx context.Context, tenant, staffID, role string) error {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) RemoveStaffMember(ctx context.Context, tenant, staffID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) ListStaffMembers(ctx context.Context, tenant, studioID string) ([]*domain.StaffMember, error) {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) AddStudioUser(ctx context.Context, tenant, studioID, email, password, role, displayName, username, firstName, lastName string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) UpdateStudioUser(ctx context.Context, tenant, userID, studioID, email, role, displayName, username, firstName, lastName string, updateMask *fieldmaskpb.FieldMask) error {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) RemoveStudioUser(ctx context.Context, tenant, userID, studioID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) SetStaffPermissions(ctx context.Context, tenant, staffID string, permissions []string) error {
	//TODO implement me
	panic("implement me")
}

func (r *StudioRepository) GetStaffPermissions(ctx context.Context, tenant, staffID string) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

// NewStudioAuthRepository creates a new StudioAuthRepository
func NewStudioRepository(dbManager *config.TenantDBManager, redisManager *config.TenantRedisManager) *StudioRepository {
	return &StudioRepository{
		DBManager:    dbManager,
		RedisManager: redisManager,
	}
}

// Claims defines JWT claims
//type Claims struct {
//	UserID   string `json:"user_id"`
//	Username string `json:"username"`
//	Email    string `json:"email"`
//	Tenant   string `json:"tenant"`
//	Role     string `json:"role"`
//	jwt.RegisteredClaims
//}

// generateAccessToken creates a JWT access token
func generateAccessToken(userID, username, email, tenant, role string) (string, error) {
	claims := domain.Claims{
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

// GetUserByEmail retrieves a user by email
//func (r *StudioAuthRepository) GetUserByEmail(ctx context.Context, tenant, email string) (string, string, string, error) {
//	// Get tenant-specific database pool
//	pool, err := r.DBManager.GetTenantDB(tenant)
//	if err != nil {
//		return "", "", "", fmt.Errorf("invalid tenant: %w", err)
//	}
//
//	var id, username, role string
//	err = pool.QueryRow(ctx,
//		"SELECT id, username, role FROM users WHERE email = $1",
//		email).Scan(&id, &username, &role)
//	if err != nil {
//		return "", "", "", fmt.Errorf("user not found: %w", err)
//	}
//
//	return id, username, role, nil
//}

// Register creates a new user in the tenant's database

// ChangePassword updates a user's password
//func (r *StudioAuthRepository) ChangePassword(ctx context.Context, tenant, email, oldPassword, newPassword string) error {
//	if tenant == "" {
//		return errors.New("tenant subdomain is required")
//	}
//
//	pool, err := r.DBManager.GetTenantDB(tenant)
//	if err != nil {
//		return fmt.Errorf("invalid tenant: %w", err)
//	}
//
//	var userID, hashedPassword string
//	err = pool.QueryRow(ctx,
//		"SELECT id, hashed_password FROM users WHERE email = $1",
//		email).Scan(&userID, &hashedPassword)
//	if err != nil {
//		return fmt.Errorf("user not found: %w", err)
//	}
//
//	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(oldPassword))
//	if err != nil {
//		return errors.New("invalid old password")
//	}
//
//	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
//	if err != nil {
//		return fmt.Errorf("failed to hash new password: %w", err)
//	}
//
//	_, err = pool.Exec(ctx,
//		"UPDATE users SET hashed_password = $1, updated_at = $2 WHERE id = $3",
//		string(newHashedPassword), time.Now(), userID)
//	if err != nil {
//		return fmt.Errorf("failed to update password: %w", err)
//	}
//
//	// Invalidate all refresh tokens
//	_, err = pool.Exec(ctx,
//		"UPDATE refresh_tokens SET invalidated_at = $1 WHERE user_id = $2 AND invalidated_at IS NULL",
//		time.Now(), userID)
//	if err != nil {
//		fmt.Printf("Warning: failed to invalidate refresh tokens: %v\n", err)
//	}
//
//	return nil
//}

// ChangeEmail updates a user's email
//func (r *StudioAuthRepository) ChangeEmail(ctx context.Context, tenant, email, password, newEmail string) error {
//	if tenant == "" {
//		return errors.New("tenant subdomain is required")
//	}
//
//	pool, err := r.DBManager.GetTenantDB(tenant)
//	if err != nil {
//		return fmt.Errorf("invalid tenant: %w", err)
//	}
//
//	var userID, hashedPassword string
//	err = pool.QueryRow(ctx,
//		"SELECT id, hashed_password FROM users WHERE email = $1",
//		email).Scan(&userID, &hashedPassword)
//	if err != nil {
//		return fmt.Errorf("user not found: %w", err)
//	}
//
//	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
//	if err != nil {
//		return errors.New("invalid credentials")
//	}
//
//	_, err = pool.Exec(ctx,
//		"UPDATE users SET email = $1, updated_at = $2 WHERE id = $3",
//		newEmail, time.Now(), userID)
//	if err != nil {
//		return fmt.Errorf("failed to update email: %w", err)
//	}
//
//	return nil
//}

//func (r *StudioAuthRepository) GetUserByID(ctx context.Context, tenant, userID string) (*domain.User, error) {
//	pool, err := r.DBManager.GetTenantDB(tenant)
//	if err != nil {
//		return nil, fmt.Errorf("invalid tenant: %w", err)
//	}
//
//	var user domain.User
//	err = pool.QueryRow(ctx,
//		"SELECT id, username, email, role FROM users WHERE id = $1",
//		userID).Scan(&user.ID, &user.Username, &user.Email, &user.Role)
//	if err != nil {
//		return nil, fmt.Errorf("user not found: %w", err)
//	}
//
//	user.StudioID = tenant
//	user.CreatedAt = time.Now()
//	user.UpdatedAt = time.Now()
//
//	return &user, nil
//}
//
//func (r *StudioAuthRepository) GetAllUsers(ctx context.Context, tenant string) ([]*domain.User, error) {
//	return nil, nil
//}
//
//func (r *StudioAuthRepository) UpdateUser(ctx context.Context, tenant string, user *domain.User) error {
//	return nil
//}
//
//func (r *StudioAuthRepository) InsertUser(ctx context.Context, tenant string, user *domain.User) error {
//	return nil
//}
//
//func (r *StudioAuthRepository) DeleteUser(ctx context.Context, tenant, userID string) error {
//	return nil
//}
