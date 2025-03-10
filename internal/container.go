package internal

import (
	"context"

	"github.com/FACorreiaa/ink-app-backend-protos/container"
	user "github.com/FACorreiaa/ink-app-backend-protos/modules/auth/generated"
	customer "github.com/FACorreiaa/ink-app-backend-protos/modules/customer/generated"
	"github.com/FACorreiaa/ink-app-backend-protos/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain/auth"
)

type Brokers struct {
	Customer       customer.CustomerClient
	Auth           user.AuthClient
	TransportUtils *utils.TransportUtils
}

//func NewBrokers(cfg *config.Config, log *zap.Logger) (*Brokers, error) {
//	// create your gRPC connections or any transport used to talk to external services
//	//customerBroker, err := customer.NewBroker(cfg.UpstreamServices.Customer)
//	//if err != nil {
//	//	return nil, err
//	//}
//
//	authBroker, err := user.NewBroker(cfg.UpstreamServices.Auth)
//	if err != nil {
//		return nil, err
//	}
//
//	return &Brokers{
//		Customer: customerBroker,
//		Auth:     authBroker,
//	}, nil
//}

type UserContainer struct {
	AuthService *auth.ServiceAuth
}

type AppContainer struct {
	Brokers     *container.Brokers
	AuthService *auth.ServiceAuth
}

//func NewAppContainer(
//	ctx context.Context,
//	db *pgxpool.Pool,
//	redisClient *redis.Client,
//	brokers *container.Brokers,
//	log *zap.Logger,
//) *AppContainer {
//	// Create session manager, repos, etc.
//	sessionManager := auth.NewSessionManager(db, redisClient)
//	authRepo := auth.NewRepository(db, redisClient, sessionManager)
//	authService := auth.NewService(ctx, authRepo, db, redisClient, sessionManager)
//
//	// Meals
//	//mealPlanRepo := meals.NewMealPlanRepository(db, redisClient, sessionManager)
//	//mealPlanService := meals.NewMealPlanService(ctx, mealPlanRepo)
//	// ... repeat for other meal services
//	//mealServices := &MealServiceContainer{
//	//	MealPlanService: mealPlanService,
//	//	// ...
//	//}
//
//	return &AppContainer{
//		Brokers:     brokers,
//		AuthService: authService,
//		//MealServices: mealServices,
//	}
//}

func NewAppContainer(ctx context.Context, pgPool *pgxpool.Pool, redisClient *redis.Client) *AppContainer {
	sessionManager := auth.NewSessionManager(pgPool, redisClient)
	authRepo := auth.NewRepository(pgPool, redisClient, sessionManager)
	authService := auth.NewService(ctx, authRepo, pgPool, redisClient, sessionManager)
	return &AppContainer{
		AuthService: authService,
	}
}
