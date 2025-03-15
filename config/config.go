package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/spf13/viper"
)

//go:embed config.yml
var embeddedConfig []byte

// TenantConfig represents configuration for a single tenant (studio)
type TenantConfig struct {
	Studio    StudioConfig   `mapstructure:"studio"`
	Owner     OwnerConfig    `mapstructure:"owner"`
	Database  DatabaseConfig `mapstructure:"database"`
	Subdomain string         `mapstructure:"subdomain"`
}

// StudioConfig represents studio-specific details
type StudioConfig struct {
	Name    string `mapstructure:"name"`
	Address string `mapstructure:"address"`
	Phone   string `mapstructure:"phone"`
	Email   string `mapstructure:"email"`
	Website string `mapstructure:"website"`
}

// OwnerConfig represents the studio owner's details
type OwnerConfig struct {
	Email       string `mapstructure:"email"`
	Password    string `mapstructure:"password"`
	DisplayName string `mapstructure:"display_name"`
	Username    string `mapstructure:"username"`
	FirstName   string `mapstructure:"first_name"`
	LastName    string `mapstructure:"last_name"`
}

// DatabaseConfig represents tenant-specific database settings
type DatabaseConfig struct {
	Host              string `mapstructure:"host"`
	Port              string `mapstructure:"port"`
	Username          string `mapstructure:"username"`
	Password          string `mapstructure:"password"`
	DB                string `mapstructure:"db"`
	SSLMODE           string `mapstructure:"sslmode"`
	MaxConWaitingTime int    `mapstructure:"max_con_waiting_time"`
}

// Config represents the overall application configuration
type Config struct {
	Mode             string                 `mapstructure:"mode"`
	Dotenv           string                 `mapstructure:"dotenv"`
	Tenants          []TenantConfig         `mapstructure:"tenants"`
	Handlers         HandlersConfig         `mapstructure:"handlers"`
	Server           ServerConfig           `mapstructure:"server"`
	UpstreamServices UpstreamServicesConfig `mapstructure:"upstream_services"`
	Redis            RedisConfig            `mapstructure:"redis"`
}

// HandlersConfig, ServerConfig, UpstreamServicesConfig, RedisConfig remain similar
type HandlersConfig struct {
	ExternalAPI struct {
		Port      string `mapstructure:"port"`
		CertFile  string `mapstructure:"certFile"`
		KeyFile   string `mapstructure:"keyFile"`
		EnableTLS bool   `mapstructure:"enableTLS"`
	} `mapstructure:"externalAPI"`
	Pprof struct {
		Port      string `mapstructure:"port"`
		CertFile  string `mapstructure:"certFile"`
		KeyFile   string `mapstructure:"keyFile"`
		EnableTLS bool   `mapstructure:"enableTLS"`
	} `mapstructure:"pprof"`
	Prometheus struct {
		Port      string `mapstructure:"port"`
		CertFile  string `mapstructure:"certFile"`
		KeyFile   string `mapstructure:"keyFile"`
		EnableTLS bool   `mapstructure:"enableTLS"`
	} `mapstructure:"prometheus"`
}

type ServerConfig struct {
	HTTPPort string        `mapstructure:"HTTPPort"`
	GrpcPort string        `mapstructure:"GRPCPort"`
	Timeout  time.Duration `mapstructure:"HTTPTimeout"`
}

type UpstreamServicesConfig struct {
	Customer    string `mapstructure:"customer"`
	Auth        string `mapstructure:"auth"`
	Calculator  string `mapstructure:"calculator"`
	Activity    string `mapstructure:"activity"`
	Workout     string `mapstructure:"workout"`
	Measurement string `mapstructure:"measurement"`
	Ingredients string `mapstructure:"ingredients"`
	Meals       string `mapstructure:"meals"`
}

type RedisConfig struct {
	Host string        `mapstructure:"host"`
	Port string        `mapstructure:"port"`
	Pass string        `mapstructure:"pass"`
	DB   int           `mapstructure:"db"`
	TTL  time.Duration `mapstructure:"ttl"`
}

type TenantDatabase struct {
	Pool *pgxpool.Pool
}

type TenantDBManager struct {
	Tenants map[string]*TenantDatabase // Key is subdomain
	Config  *Config
}

type TenantRedis struct {
	Client *redis.Client
}

type TenantRedisManager struct {
	Tenants map[string]*TenantRedis // Key is subdomain
	Config  *Config
}

func InitConfig() (Config, error) {
	var config Config
	v := viper.New()

	// Add file-based config paths
	v.AddConfigPath(".")
	v.AddConfigPath("config")
	v.AddConfigPath("/app/config")
	v.AddConfigPath("/usr/local/bin")
	v.AddConfigPath("/usr/local/bin/inkme")

	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Try to load file-based config
	err := v.ReadInConfig()
	if err != nil {
		fmt.Printf("Warning: Failed to find file-based config: %s. Falling back to embedded config.\n", err)
		if err = v.ReadConfig(bytes.NewReader(embeddedConfig)); err != nil {
			return Config{}, fmt.Errorf("failed to read embedded config: %s", err)
		}
	}

	// Unmarshal the config into the Config struct
	if err = v.Unmarshal(&config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %s", err)
	}
	fmt.Println("Successfully loaded app configs...")
	return config, nil
}

// GetTenantConfig retrieves a tenant's configuration by subdomain
func (c *Config) GetTenantConfig(subdomain string) (*TenantConfig, error) {
	for _, tenant := range c.Tenants {
		if tenant.Subdomain == subdomain {
			return &tenant, nil
		}
	}
	return nil, fmt.Errorf("tenant with subdomain %s not found", subdomain)
}

func (m *TenantDBManager) GetTenantDB(subdomain string) (*pgxpool.Pool, error) {
	if tenantDB, ok := m.Tenants[subdomain]; ok {
		return tenantDB.Pool, nil
	}
	return nil, fmt.Errorf("no database found for tenant with subdomain: %s", subdomain)
}

func (m *TenantRedisManager) GetTenantRedis(subdomain string) (*redis.Client, error) {
	tenant, exists := m.Tenants[subdomain]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", subdomain)
	}
	return tenant.Client, nil
}
