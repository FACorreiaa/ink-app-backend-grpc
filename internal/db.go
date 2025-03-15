package internal

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"context"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	uuidpg "github.com/vgarvardt/pgx-google-uuid/v5"

	"github.com/FACorreiaa/ink-app-backend-grpc/config"
	"github.com/FACorreiaa/ink-app-backend-grpc/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const retries = 25

type RedisConfig struct {
	Host     string
	Password string
	DB       int
}

type DatabaseConfig struct {
	ConnectionURL string
}

func NewRedisConfig() (*redis.Client, error) {
	cfg, err := config.InitConfig()
	if err != nil {
		zap.Error(err)
	}
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port), // Use both host and port
		Password: cfg.Redis.Pass,
		DB:       cfg.Redis.DB,
	}), nil
}

func NewTenantDBManager(cfg *config.Config) (*config.TenantDBManager, error) {
	log := logger.Log
	manager := &config.TenantDBManager{
		Tenants: make(map[string]*config.TenantDatabase),
		Config:  cfg,
	}

	if err := EnsureDatabasesExist(cfg); err != nil {
		log.Error("Failed to ensure databases exist", zap.Error(err))
		return nil, err
	}

	for _, tenant := range cfg.Tenants {
		dbConfig := tenant.Database
		connURL := url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(dbConfig.Username, dbConfig.Password),
			Host:   fmt.Sprintf("%s:%s", dbConfig.Host, dbConfig.Port),
			Path:   dbConfig.DB,
			RawQuery: url.Values{
				"sslmode":  []string{dbConfig.SSLMODE},
				"timezone": []string{"utc"},
			}.Encode(),
		}

		pool, err := Init(connURL.String())
		if err != nil {
			log.Error("Failed to initialize database for tenant", zap.String("subdomain", tenant.Subdomain), zap.Error(err))
			return nil, err
		}

		WaitForDB(pool)

		if err := RunMigrations(dbConfig.Host, dbConfig.Port, dbConfig.Username, dbConfig.Password, dbConfig.DB, tenant.Subdomain); err != nil {
			log.Error("Failed to run migrations for tenant", zap.String("subdomain", tenant.Subdomain), zap.Error(err))
			return nil, err
		}

		// Initialize the tenant system (create studio and owner if needed)
		if err := InitializeTenantSystem(context.Background(), pool, &tenant); err != nil {
			log.Error("Failed to initialize tenant system", zap.String("subdomain", tenant.Subdomain), zap.Error(err))
			return nil, err
		}

		manager.Tenants[tenant.Subdomain] = &config.TenantDatabase{Pool: pool}
		log.Info("Initialized database for tenant", zap.String("subdomain", tenant.Subdomain))
	}

	return manager, nil
}

func RunMigrations(host, port, username, password, database, subdomain string) error {
	log := logger.Log
	log.Info("Starting Migrations", zap.String("subdomain", subdomain))

	// Create migration DSN
	migrationDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		username, password, host, port, database)

	// Use your Migrate function that works with the embedded FS
	// or modify this function to use the embedded filesystem
	conn, err := pgxpool.New(context.Background(), migrationDSN)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Info("Successfully ran migrations", zap.String("subdomain", subdomain))

	// Use the Migrate function you already have that works with migrationFS
	return Migrate(conn)
}

func EnsureDatabasesExist(cfg *config.Config) error {
	log := logger.Log
	ctx := context.Background()

	// Connect to the default PostgreSQL database
	defaultConnURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.Tenants[0].Database.Username, cfg.Tenants[0].Database.Password),
		Host:   fmt.Sprintf("%s:%s", cfg.Tenants[0].Database.Host, cfg.Tenants[0].Database.Port),
		Path:   "postgres", // Connect to default database
		RawQuery: url.Values{
			"sslmode":  []string{cfg.Tenants[0].Database.SSLMODE},
			"timezone": []string{"utc"},
		}.Encode(),
	}

	// Connect to the PostgreSQL server
	defaultPool, err := pgxpool.New(ctx, defaultConnURL.String())
	if err != nil {
		log.Error("Failed to connect to default PostgreSQL database", zap.Error(err))
		return err
	}
	defer defaultPool.Close()

	// Check each tenant database and create if it doesn't exist
	for _, tenant := range cfg.Tenants {
		var exists bool
		err := defaultPool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)",
			tenant.Database.DB).Scan(&exists)
		if err != nil {
			log.Error("Failed to check if database exists", zap.String("db", tenant.Database.DB), zap.Error(err))
			return err
		}

		if !exists {
			log.Info("Creating database", zap.String("db", tenant.Database.DB))
			_, err = defaultPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", tenant.Database.DB))
			if err != nil {
				log.Error("Failed to create database", zap.String("db", tenant.Database.DB), zap.Error(err))
				return err
			}
		}
	}

	return nil
}

func NewTenantRedisManager(cfg *config.Config) (*config.TenantRedisManager, error) {
	log := logger.Log
	manager := &config.TenantRedisManager{
		Tenants: make(map[string]*config.TenantRedis),
		Config:  cfg,
	}

	for _, tenant := range cfg.Tenants {
		// You might want to add Redis configuration to your tenant config
		// For now, we'll use a simple approach using the tenant subdomain as a database index
		client := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Pass,
			DB:       getTenantRedisDB(tenant.Subdomain),
		})

		// Test the connection
		_, err := client.Ping(context.Background()).Result()
		if err != nil {
			log.Error("Failed to connect to Redis for tenant", zap.String("subdomain", tenant.Subdomain), zap.Error(err))
			return nil, err
		}

		manager.Tenants[tenant.Subdomain] = &config.TenantRedis{Client: client}
		log.Info("Initialized Redis client for tenant", zap.String("subdomain", tenant.Subdomain))
	}

	return manager, nil
}

// Helper function to map tenant subdomain to a Redis DB index
// Redis typically supports 16 databases (0-15) by default
func getTenantRedisDB(subdomain string) int {
	// This is a simple hash function - you might want something more sophisticated
	sum := 0
	for _, char := range subdomain {
		sum += int(char)
	}
	return sum % 16
}

func Init(connectionURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connectionURL)
	if err != nil {
		return nil, err
	}
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		uuidpg.Register(conn.TypeMap())
		return nil
	}
	return pgxpool.NewWithConfig(context.Background(), cfg)
}

func WaitForDB(pgpool *pgxpool.Pool) {
	ctx := context.Background()

	for attempts := 1; ; attempts++ {
		if attempts > retries {
			break
		}

		if err := pgpool.Ping(ctx); err == nil {
			break
		}

		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
	}
}

func hashVal(contents []byte) string {
	hashed := sha256.New()
	hashed.Write(contents)
	return fmt.Sprintf("%x", hashed.Sum(nil))
}

func containsCreateDatabase(sql string) bool {
	return strings.Contains(strings.ToUpper(sql), "CREATE DATABASE")
}

func Migrate(conn *pgxpool.Pool) error {

	// migrate db
	log := logger.Log
	log.Info("Running migrations")
	ctx := context.Background()
	files, err := migrationFS.ReadDir("migrations")
	if err != nil {
		log.Error("Failed executing migrations", zap.Error(err))
		return err
	}

	log.Info("Creating migrations table")
	_, err = conn.Exec(ctx, `
		create table if not exists _migrations (
			name text primary key,
			hash text not null,
			created_at timestamp default now()
		);
	`)
	if err != nil {
		log.Error("Failed loading Postgres config", zap.Error(err))
	}

	log.Info("Checking applied migrations")
	rows, _ := conn.Query(ctx, `select name, hash from _migrations order by created_at desc`)
	var name, hash string
	appliedMigrations := make(map[string]string)
	_, err = pgx.ForEachRow(rows, []any{&name, &hash}, func() error {
		appliedMigrations[name] = hash
		return nil
	})

	if err != nil {
		log.Error("Failed loading Postgres config", zap.Error(err))
		return err
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		contents, err := migrationFS.ReadFile("migrations/" + file.Name())
		if err != nil {
			return err
		}
		contentStr := string(contents)
		val := hashVal(contents)
		contentHash := fmt.Sprintf("%x", val)

		if prevHash, ok := appliedMigrations[file.Name()]; ok {
			if prevHash != contentHash {
				return fmt.Errorf("hash mismatch for migration %s", file.Name())
			}
			log.Info(file.Name() + " already applied")
			continue
		}

		// Check if this migration contains CREATE DATABASE
		if containsCreateDatabase(contentStr) {
			// Execute CREATE DATABASE outside of transaction
			_, err = conn.Exec(ctx, contentStr)
			if err != nil {
				log.Error("Failed executing CREATE DATABASE", zap.Error(err))
				return err
			}

			// Record this migration as applied
			_, err = conn.Exec(ctx, `insert into _migrations (name, hash) values ($1, $2)`,
				file.Name(), contentHash)
			if err != nil {
				log.Error("Failed recording migration", zap.Error(err))
				return err
			}

			log.Info(file.Name() + " applied (with CREATE DATABASE)")
			continue
		}

		// For other migrations, use transaction as before
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err = tx.Exec(ctx, contentStr); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `insert into _migrations (name, hash) values ($1, $2)`,
				file.Name(), contentHash); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			log.Error("Failed loading Postgres config", zap.Error(err))
			return err
		}
		log.Info(file.Name() + " applied")
	}

	log.Info("Migrations finished")
	return nil
}

func InitializeTenantSystem(ctx context.Context, pool *pgxpool.Pool, tenantCfg *config.TenantConfig) error {
	log := logger.Log

	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM studios").Scan(&count)
	if err != nil {
		log.Error("Failed to check if studios exist", zap.Error(err))
		return err
	}

	if count > 0 {
		log.Info("Tenant already initialized", zap.String("subdomain", tenantCfg.Subdomain))
		return nil
	}

	log.Info("Initializing tenant", zap.String("subdomain", tenantCfg.Subdomain))

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	studioID := uuid.New()
	ownerID := uuid.New()
	now := time.Now()

	_, err = tx.Exec(ctx,
		`INSERT INTO studios (id, name, subdomain, address, phone, email, website, created_at, updated_at) 
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		studioID, tenantCfg.Studio.Name, tenantCfg.Subdomain, tenantCfg.Studio.Address,
		tenantCfg.Studio.Phone, tenantCfg.Studio.Email, tenantCfg.Studio.Website, now)
	if err != nil {
		log.Error("Failed to create initial studio", zap.Error(err))
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tenantCfg.Owner.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Failed to hash password", zap.Error(err))
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, studio_id, email, hashed_password, role, display_name, username, first_name, last_name, created_at, updated_at)
         VALUES ($1, $2, $3, $4, 'OWNER', $5, $6, $7, $8, $9, $9)`,
		ownerID, studioID, tenantCfg.Owner.Email, hashedPassword,
		tenantCfg.Owner.DisplayName, tenantCfg.Owner.Username, tenantCfg.Owner.FirstName, tenantCfg.Owner.LastName, now)
	if err != nil {
		log.Error("Failed to create initial owner", zap.Error(err))
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	log.Info("Tenant initialized successfully",
		zap.String("subdomain", tenantCfg.Subdomain),
		zap.String("studio_id", studioID.String()),
		zap.String("owner_id", ownerID.String()))
	return nil
}

// // PgService Service represents a service that interacts with a database.
// type PgService interface {
// 	// Health returns a map of health status information.
// 	// The keys and values in the map are service-specific.
// 	Health() map[string]string

// 	// Close terminates the database connection.
// 	// It returns an error if the connection cannot be closed.
// 	Close() error
// }

// type pgService struct {
// 	db *sql.DB
// }

// // Health checks the health of the database connection by pinging the database.
// // It returns a map with keys indicating various health statistics.
// func (s *pgService) Health() map[string]string {
// 	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
// 	defer cancel()
// 	if err := logger.Init(zapcore.InfoLevel); err != nil {
// 		fmt.Println("Error initializing logger:", err)
// 		os.Exit(1)
// 	}
// 	log := logger.Log
// 	stats := make(map[string]string)

// 	// Ping the database
// 	err := s.db.PingContext(ctx)
// 	if err != nil {
// 		stats["status"] = "down"
// 		stats["error"] = fmt.Sprintf("db down: %v", err)
// 		log.Fatal(fmt.Sprintf("db down: %v", err)) // Log the error and terminate the program
// 		return stats
// 	}

// 	// Database is up, add more statistics
// 	stats["status"] = "up"
// 	stats["message"] = "It's healthy"

// 	// Get database stats (like open connections, in use, idle, etc.)
// 	dbStats := s.db.Stats()
// 	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
// 	stats["in_use"] = strconv.Itoa(dbStats.InUse)
// 	stats["idle"] = strconv.Itoa(dbStats.Idle)
// 	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
// 	stats["wait_duration"] = dbStats.WaitDuration.String()
// 	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
// 	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

// 	// Evaluate stats to provide a health message
// 	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
// 		stats["message"] = "The database is experiencing heavy load."
// 	}

// 	if dbStats.WaitCount > 1000 {
// 		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
// 	}

// 	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
// 		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
// 	}

// 	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
// 		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
// 	}

// 	return stats
// }

// // redis check

// type RedisService interface {
// 	Health() map[string]string
// }

// type redisService struct {
// 	db *redis.Client
// }

// // checkRedisHealth checks the health of the Redis server and adds the relevant statistics to the stats map.
// func (s *redisService) checkRedisHealth(ctx context.Context, stats map[string]string) map[string]string {
// 	// Ping the Redis server to check its availability.
// 	pong, err := s.db.Ping(ctx).Result()
// 	log := logger.Log

// 	// Note: By extracting and simplifying like this, `log.Fatalf(fmt.Sprintf("db down: %v", err))`
// 	// can be changed into a standard error instead of a fatal error.
// 	if err != nil {
// 		log.Fatal(fmt.Sprintf("db down: %v", err))
// 	}

// 	// Redis is up
// 	stats["redis_status"] = "up"
// 	stats["redis_message"] = "It's healthy"
// 	stats["redis_ping_response"] = pong

// 	// Retrieve Redis server information.
// 	info, err := s.db.Info(ctx).Result()
// 	if err != nil {
// 		stats["redis_message"] = fmt.Sprintf("Failed to retrieve Redis info: %v", err)
// 		return stats
// 	}

// 	// Parse the Redis info response.
// 	redisInfo := parseRedisInfo(info)

// 	// Get the pool stats of the Redis client.
// 	poolStats := s.db.PoolStats()

// 	// Prepare the stats map with Redis server information and pool statistics.
// 	// Note: The "stats" map in the code uses string keys and values,
// 	// which is suitable for structuring and serializing the data for the frontend (e.g., JSON, XML, HTMX).
// 	// Using string types allows for easy conversion and compatibility with various data formats,
// 	// making it convenient to create health stats for monitoring or other purposes.
// 	// Also note that any raw "memory" (e.g., used_memory) value here is in bytes and can be converted to megabytes or gigabytes as a float64.
// 	stats["redis_version"] = redisInfo["redis_version"]
// 	stats["redis_mode"] = redisInfo["redis_mode"]
// 	stats["redis_connected_clients"] = redisInfo["connected_clients"]
// 	stats["redis_used_memory"] = redisInfo["used_memory"]
// 	stats["redis_used_memory_peak"] = redisInfo["used_memory_peak"]
// 	stats["redis_uptime_in_seconds"] = redisInfo["uptime_in_seconds"]
// 	stats["redis_hits_connections"] = strconv.FormatUint(uint64(poolStats.Hits), 10)
// 	stats["redis_misses_connections"] = strconv.FormatUint(uint64(poolStats.Misses), 10)
// 	stats["redis_timeouts_connections"] = strconv.FormatUint(uint64(poolStats.Timeouts), 10)
// 	stats["redis_total_connections"] = strconv.FormatUint(uint64(poolStats.TotalConns), 10)
// 	stats["redis_idle_connections"] = strconv.FormatUint(uint64(poolStats.IdleConns), 10)
// 	stats["redis_stale_connections"] = strconv.FormatUint(uint64(poolStats.StaleConns), 10)
// 	stats["redis_max_memory"] = redisInfo["maxmemory"]

// 	// Calculate the number of active connections.
// 	// Note: We use math.Max to ensure that activeConns is always non-negative,
// 	// avoiding the need for an explicit check for negative values.
// 	// This prevents a potential underflow situation.
// 	activeConns := uint64(math.Max(float64(poolStats.TotalConns-poolStats.IdleConns), 0))
// 	stats["redis_active_connections"] = strconv.FormatUint(activeConns, 10)

// 	// Calculate the pool size percentage.
// 	poolSize := s.db.Options().PoolSize
// 	connectedClients, _ := strconv.Atoi(redisInfo["connected_clients"])
// 	poolSizePercentage := float64(connectedClients) / float64(poolSize) * 100
// 	stats["redis_pool_size_percentage"] = fmt.Sprintf("%.2f%%", poolSizePercentage)

// 	// Evaluate Redis stats and update the stats map with relevant messages.
// 	return s.evaluateRedisStats(redisInfo, stats)
// }

// // evaluateRedisStats evaluates the Redis server statistics and updates the stats map with relevant messages.
// func (s *redisService) evaluateRedisStats(redisInfo, stats map[string]string) map[string]string {
// 	poolSize := s.db.Options().PoolSize
// 	poolStats := s.db.PoolStats()
// 	connectedClients, _ := strconv.Atoi(redisInfo["connected_clients"])
// 	highConnectionThreshold := int(float64(poolSize) * 0.8)

// 	// Check if the number of connected clients is high.
// 	if connectedClients > highConnectionThreshold {
// 		stats["redis_message"] = "Redis has a high number of connected clients"
// 	}

// 	// Check if the number of stale connections exceeds a threshold.
// 	minStaleConnectionsThreshold := 500
// 	if int(poolStats.StaleConns) > minStaleConnectionsThreshold {
// 		stats["redis_message"] = fmt.Sprintf("Redis has %d stale connections.", poolStats.StaleConns)
// 	}

// 	// Check if Redis is using a significant amount of memory.
// 	usedMemory, _ := strconv.ParseInt(redisInfo["used_memory"], 10, 64)
// 	maxMemory, _ := strconv.ParseInt(redisInfo["maxmemory"], 10, 64)
// 	if maxMemory > 0 {
// 		usedMemoryPercentage := float64(usedMemory) / float64(maxMemory) * 100
// 		if usedMemoryPercentage >= 90 {
// 			stats["redis_message"] = "Redis is using a significant amount of memory"
// 		}
// 	}

// 	// Check if Redis has been recently restarted.
// 	uptimeInSeconds, _ := strconv.ParseInt(redisInfo["uptime_in_seconds"], 10, 64)
// 	if uptimeInSeconds < 3600 {
// 		stats["redis_message"] = "Redis has been recently restarted"
// 	}

// 	// Check if the number of idle connections is high.
// 	idleConns := int(poolStats.IdleConns)
// 	highIdleConnectionThreshold := int(float64(poolSize) * 0.7)
// 	if idleConns > highIdleConnectionThreshold {
// 		stats["redis_message"] = "Redis has a high number of idle connections"
// 	}

// 	// Check if the connection pool utilization is high.
// 	poolUtilization := float64(poolStats.TotalConns-poolStats.IdleConns) / float64(poolSize) * 100
// 	highPoolUtilizationThreshold := 90.0
// 	if poolUtilization > highPoolUtilizationThreshold {
// 		stats["redis_message"] = "Redis connection pool utilization is high"
// 	}

// 	return stats
// }

// // parseRedisInfo parses the Redis info response and returns a map of key-value pairs.
// func parseRedisInfo(info string) map[string]string {
// 	result := make(map[string]string)
// 	lines := strings.Split(info, "\r\n")
// 	for _, line := range lines {
// 		if strings.Contains(line, ":") {
// 			parts := strings.Split(line, ":")
// 			key := strings.TrimSpace(parts[0])
// 			value := strings.TrimSpace(parts[1])
// 			result[key] = value
// 		}
// 	}
// 	return result
// }
