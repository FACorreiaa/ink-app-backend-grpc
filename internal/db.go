package internal

import (
	"github.com/FACorreiaa/ink-app-backend-grpc/configs"
	"go.uber.org/zap"
)

type RedisConfig struct {
	Host     string
	Password string
	DB       int
}

func NewRedisConfig() (*RedisConfig, error) {
	cfg, err := configs.InitConfig()
	if err != nil {
		zap.Error(err)
	}

	return &RedisConfig{
		Host:     cfg.Repositories.Redis.Host,
		Password: cfg.Repositories.Redis.Pass,
		DB:       cfg.Repositories.Redis.DB,
	}, nil
}
