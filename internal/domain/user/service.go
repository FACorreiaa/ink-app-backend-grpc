package user

import (
	upu "github.com/FACorreiaa/ink-app-backend-protos/modules/user/generated"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
)

type UserService struct {
	upu.UnimplementedUserServiceServer
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{repo: repo}
}
