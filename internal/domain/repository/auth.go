package repository

import (
	"context"
	"errors"
	"time"

	pb "github.com/FACorreiaa/ink-app-backend-protos/modules/user/generated"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	pb.UnimplementedAuthServer
	pgpool *pgxpool.Pool
	redis  *redis.Client
}

// NewAuthService creates a new AuthService
func NewAuthService(db *pgxpool.Pool, redis *redis.Client) *AuthService {
	return &AuthService{pgpool: db, redis: redis}
}

func (a *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	_, err = a.pgpool.Exec(ctx, `INSERT INTO "user" (username, email, password_hash) VALUES ($1, $2, $3)`, req.Username, req.Email, hashedPassword)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterResponse{Message: "Registration successful"}, nil
}

func (a *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	var passwordHash string
	err := a.pgpool.QueryRow(ctx, `SELECT password_hash FROM "user" WHERE username=$1`, req.Username).Scan(&passwordHash)
	if err != nil {
		return nil, errors.New("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	token := "generated-jwt-token"

	err = a.redis.Set(ctx, req.Username, token, 0).Err()
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{Token: token}, nil
}

func (a *AuthService) Logout(ctx context.Context, req *pb.NilReq) (*pb.NilRes, error) {
	username := "extracted-username"

	err := a.redis.Del(ctx, username).Err()
	if err != nil {
		return nil, err
	}

	return &pb.NilRes{}, nil
}

func (a *AuthService) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	var passwordHash string
	err := a.pgpool.QueryRow(ctx, `SELECT password_hash FROM "user" WHERE username=$1`, req.Username).Scan(&passwordHash)
	if err != nil {
		return nil, errors.New("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.OldPassword))
	if err != nil {
		return nil, errors.New("invalid old password")
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	_, err = a.pgpool.Exec(ctx, `UPDATE "user" SET password_hash=$1, updated_at=now() WHERE username=$2`, newHashedPassword, req.Username)
	if err != nil {
		return nil, err
	}

	return &pb.ChangePasswordResponse{Message: "Password changed successfully"}, nil
}

func (a *AuthService) ChangeEmail(ctx context.Context, req *pb.ChangeEmailRequest) (*pb.ChangeEmailResponse, error) {
	var passwordHash string
	err := a.pgpool.QueryRow(ctx, `SELECT password_hash FROM "user" WHERE username=$1`, req.Username).Scan(&passwordHash)
	if err != nil {
		return nil, errors.New("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	_, err = a.pgpool.Exec(ctx, `UPDATE "user" SET email=$1, updated_at=now() WHERE username=$2`, req.NewEmail, req.Username)
	if err != nil {
		return nil, err
	}

	return &pb.ChangeEmailResponse{Message: "Email changed successfully"}, nil
}

func (a *AuthService) GetAllUsers(ctx context.Context) (*pb.GetAllUsersResponse, error) {
	rows, err := a.pgpool.Query(ctx, `SELECT user_id, username, email, created_at, updated_at FROM "user"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*pb.User
	for rows.Next() {
		var id, username, email string
		var createdAt time.Time
		var updatedAt *time.Time

		err := rows.Scan(&id, &username, &email, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}

		var updatedAtStr string
		if updatedAt != nil {
			updatedAtStr = updatedAt.Format(time.RFC3339)
		}

		users = append(users, &pb.User{
			Id:        id,
			Username:  username,
			Email:     email,
			CreatedAt: createdAt.Format(time.RFC3339),
			UpdatedAt: updatedAtStr,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &pb.GetAllUsersResponse{Users: users}, nil
}
