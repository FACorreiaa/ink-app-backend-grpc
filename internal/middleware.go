package internal

import (
	"github.com/FACorreiaa/ink-app-backend-grpc/.config"
	"github.com/FACorreiaa/ink-app-backend-grpc/logger"
	"github.com/FACorreiaa/ink-app-backend-protos/container"
	"github.com/FACorreiaa/ink-app-backend-protos/modules/customer"
	"github.com/FACorreiaa/ink-app-backend-protos/utils"
	"go.uber.org/zap"
)

// ConfigureUpstreamClients maintains the broker container so we have a struct that we can pass
// down to the service, with connections to all other services that we need
func ConfigureUpstreamClients(log *zap.Logger, transport *utils.TransportUtils) *container.Brokers {
	brokers := container.NewBrokers(transport)
	if brokers == nil {
		logger.Log.Error("failed to setup container - did you configure transport utils?")

		return nil
	}
	cfg, err := _config.InitConfig()
	if err != nil {
		logger.Log.Error("failed to initialize config")
		return nil
	}
	// If you have a lot of upstream services, you'll probably want to use an
	// itt here instead, but for the example we've only got the one.

	customerBroker, err := customer.NewBroker(cfg.UpstreamServices.Customer)
	if err != nil {
		logger.Log.Error("failed to create customer service broker", zap.Error(err))
		return nil
	}
	brokers.Customer = customerBroker
	return brokers
}
