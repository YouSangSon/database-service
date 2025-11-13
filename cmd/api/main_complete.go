package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/YouSangSon/database-service/internal/application/usecase"
	"github.com/YouSangSon/database-service/internal/config"
	"github.com/YouSangSon/database-service/internal/infrastructure/cache"
	"github.com/YouSangSon/database-service/internal/infrastructure/messaging/kafka"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/cassandra"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/elasticsearch"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/mongodb"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/mysql"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/postgresql"
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/vitess"
	httpHandler "github.com/YouSangSon/database-service/internal/interfaces/http/handler"
	"github.com/YouSangSon/database-service/internal/interfaces/http/router"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"github.com/YouSangSon/database-service/internal/pkg/vault"
	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/gocql/gocql"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// @title Database Service API
// @version 2.0
// @description Enterprise-grade multi-database service supporting MongoDB, PostgreSQL, MySQL, Cassandra, Elasticsearch, and Vitess with Clean Architecture and comprehensive observability
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/YouSangSon/database-service
// @contact.email support@database-service.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

func main() {
	// ============================================
	// 1. Configuration
	// ============================================
	cfg, err := config.LoadConfig("./configs", "config")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// ============================================
	// 2. Logger Initialization
	// ============================================
	if err := logger.Init(&logger.Config{
		Level:       cfg.Observability.Logging.Level,
		Environment: cfg.App.Environment,
		ServiceName: cfg.App.Name,
		Version:     cfg.App.Version,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	logger.Info(ctx, "starting multi-database service",
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.App.Environment),
		zap.String("go_version", runtime.Version()),
	)

	// ============================================
	// 3. Metrics Initialization
	// ============================================
	m := metrics.Init(cfg.App.Name)
	logger.Info(ctx, "metrics initialized")

	// ============================================
	// 4. Tracing Initialization
	// ============================================
	var tracingShutdown func(context.Context) error
	if cfg.Observability.Tracing.Enabled {
		tracingShutdown, err = tracing.Init(&tracing.Config{
			ServiceName:    cfg.App.Name,
			ServiceVersion: cfg.App.Version,
			Environment:    cfg.App.Environment,
			JaegerEndpoint: cfg.Observability.Tracing.JaegerEndpoint,
			Enabled:        true,
		})
		if err != nil {
			logger.Fatal(ctx, "failed to initialize tracing", zap.Error(err))
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracingShutdown(shutdownCtx); err != nil {
				logger.Error(ctx, "failed to shutdown tracing", zap.Error(err))
			}
		}()
		logger.Info(ctx, "tracing initialized", zap.String("jaeger_endpoint", cfg.Observability.Tracing.JaegerEndpoint))
	}

	// ============================================
	// 5. Vault Client Initialization (Optional)
	// ============================================
	var vaultClient *vault.Client
	if cfg.Vault.Enabled {
		vaultClient, err = vault.NewClient(&vault.Config{
			Address:           cfg.Vault.Address,
			Token:             cfg.Vault.Token,
			AuthMethod:        cfg.Vault.AuthMethod,
			RoleID:            cfg.Vault.RoleID,
			SecretID:          cfg.Vault.SecretID,
			K8sRole:           cfg.Vault.K8sRole,
			MongoDBPath:       cfg.Vault.Paths.MongoDB,
			RenewInterval:     cfg.Vault.Renewal.Interval,
			RenewBeforeExpiry: cfg.Vault.Renewal.RenewBeforeExpiry,
		})
		if err != nil {
			logger.Fatal(ctx, "failed to initialize vault client", zap.Error(err))
		}
		defer vaultClient.Close()

		if err := vaultClient.HealthCheck(ctx); err != nil {
			logger.Warn(ctx, "vault health check failed", zap.Error(err))
		} else {
			logger.Info(ctx, "vault client initialized successfully")
		}
	}

	// ============================================
	// 6. Repository Manager Initialization
	// ============================================
	repoManager := persistence.NewRepositoryManager()
	logger.Info(ctx, "repository manager initialized")

	// Track which databases are enabled
	enabledDatabases := make([]string, 0, 6)

	// ============================================
	// 7. Database Repositories Initialization
	// ============================================

	// 7.1. MongoDB
	var mongoClient *mongo.Client
	if cfg.MongoDB.Enabled {
		var mongoURI string
		if cfg.MongoDB.UseVault && vaultClient != nil {
			username, password, err := vaultClient.GetMongoDBCredentials(ctx)
			if err != nil {
				logger.Fatal(ctx, "failed to get mongodb credentials from vault", zap.Error(err))
			}
			mongoURI = fmt.Sprintf("mongodb://%s:%s@%s", username, password, cfg.MongoDB.Host)
			logger.Info(ctx, "using vault-managed mongodb credentials")
		} else {
			mongoURI = cfg.MongoDB.URI
		}

		mongoRepo, client, err := mongodb.NewDocumentRepository(ctx, mongoURI, cfg.MongoDB.Database, vaultClient)
		if err != nil {
			logger.Fatal(ctx, "failed to initialize mongodb repository", zap.Error(err))
		}
		mongoClient = client

		// Register with RepositoryManager
		if err := repoManager.InitializeMongoDB(ctx, mongoClient, cfg.MongoDB.Database); err != nil {
			logger.Fatal(ctx, "failed to register mongodb repository", zap.Error(err))
		}

		enabledDatabases = append(enabledDatabases, "mongodb")
		logger.Info(ctx, "mongodb repository initialized and registered",
			zap.String("database", cfg.MongoDB.Database),
		)

		// Keep mongoRepo for health checks
		defer func() {
			if err := mongoRepo.Close(ctx); err != nil {
				logger.Error(ctx, "failed to close mongodb connection", zap.Error(err))
			}
		}()
	}

	// 7.2. PostgreSQL
	var postgresDB *sql.DB
	if cfg.PostgreSQL.Enabled {
		pgConfig := &postgresql.Config{
			Host:            cfg.PostgreSQL.Host,
			Port:            cfg.PostgreSQL.Port,
			User:            cfg.PostgreSQL.User,
			Password:        cfg.PostgreSQL.Password,
			Database:        cfg.PostgreSQL.Database,
			SSLMode:         cfg.PostgreSQL.SSLMode,
			MaxOpenConns:    cfg.PostgreSQL.MaxOpenConns,
			MaxIdleConns:    cfg.PostgreSQL.MaxIdleConns,
			ConnMaxLifetime: cfg.PostgreSQL.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.PostgreSQL.ConnMaxIdleTime,
		}

		postgresDB, err = postgresql.NewClient(ctx, pgConfig)
		if err != nil {
			logger.Fatal(ctx, "failed to initialize postgresql client", zap.Error(err))
		}

		// Register with RepositoryManager
		if err := repoManager.InitializePostgreSQL(ctx, postgresDB); err != nil {
			logger.Fatal(ctx, "failed to register postgresql repository", zap.Error(err))
		}

		enabledDatabases = append(enabledDatabases, "postgresql")
		logger.Info(ctx, "postgresql repository initialized and registered",
			zap.String("database", cfg.PostgreSQL.Database),
		)

		defer func() {
			if err := postgresDB.Close(); err != nil {
				logger.Error(ctx, "failed to close postgresql connection", zap.Error(err))
			}
		}()
	}

	// 7.3. MySQL
	var mysqlDB *sql.DB
	if cfg.MySQL.Enabled {
		mysqlConfig := &mysql.Config{
			Host:            cfg.MySQL.Host,
			Port:            cfg.MySQL.Port,
			User:            cfg.MySQL.User,
			Password:        cfg.MySQL.Password,
			Database:        cfg.MySQL.Database,
			Charset:         cfg.MySQL.Charset,
			ParseTime:       cfg.MySQL.ParseTime,
			MaxOpenConns:    cfg.MySQL.MaxOpenConns,
			MaxIdleConns:    cfg.MySQL.MaxIdleConns,
			ConnMaxLifetime: cfg.MySQL.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.MySQL.ConnMaxIdleTime,
		}

		mysqlDB, err = mysql.NewClient(ctx, mysqlConfig)
		if err != nil {
			logger.Fatal(ctx, "failed to initialize mysql client", zap.Error(err))
		}

		// Register with RepositoryManager
		if err := repoManager.InitializeMySQL(ctx, mysqlDB); err != nil {
			logger.Fatal(ctx, "failed to register mysql repository", zap.Error(err))
		}

		enabledDatabases = append(enabledDatabases, "mysql")
		logger.Info(ctx, "mysql repository initialized and registered",
			zap.String("database", cfg.MySQL.Database),
		)

		defer func() {
			if err := mysqlDB.Close(); err != nil {
				logger.Error(ctx, "failed to close mysql connection", zap.Error(err))
			}
		}()
	}

	// 7.4. Cassandra
	var cassandraSession *gocql.Session
	if cfg.Cassandra.Enabled {
		cassandraConfig := &cassandra.Config{
			Hosts:       cfg.Cassandra.Hosts,
			Port:        cfg.Cassandra.Port,
			Keyspace:    cfg.Cassandra.Keyspace,
			Username:    cfg.Cassandra.Username,
			Password:    cfg.Cassandra.Password,
			Consistency: cfg.Cassandra.Consistency,
			NumConns:    cfg.Cassandra.NumConns,
			Timeout:     cfg.Cassandra.Timeout,
		}

		cassandraSession, err = cassandra.NewClient(ctx, cassandraConfig)
		if err != nil {
			logger.Fatal(ctx, "failed to initialize cassandra client", zap.Error(err))
		}

		// Register with RepositoryManager
		if err := repoManager.InitializeCassandra(ctx, cassandraSession, cfg.Cassandra.Keyspace); err != nil {
			logger.Fatal(ctx, "failed to register cassandra repository", zap.Error(err))
		}

		enabledDatabases = append(enabledDatabases, "cassandra")
		logger.Info(ctx, "cassandra repository initialized and registered",
			zap.String("keyspace", cfg.Cassandra.Keyspace),
		)

		defer cassandraSession.Close()
	}

	// 7.5. Elasticsearch
	var elasticsearchClient *es.Client
	if cfg.Elasticsearch.Enabled {
		esConfig := &elasticsearch.Config{
			Addresses:  cfg.Elasticsearch.Addresses,
			Username:   cfg.Elasticsearch.Username,
			Password:   cfg.Elasticsearch.Password,
			APIKey:     cfg.Elasticsearch.APIKey,
			CloudID:    cfg.Elasticsearch.CloudID,
			MaxRetries: cfg.Elasticsearch.MaxRetries,
		}

		elasticsearchClient, err = elasticsearch.NewClient(ctx, esConfig)
		if err != nil {
			logger.Fatal(ctx, "failed to initialize elasticsearch client", zap.Error(err))
		}

		// Register with RepositoryManager
		if err := repoManager.InitializeElasticsearch(ctx, elasticsearchClient); err != nil {
			logger.Fatal(ctx, "failed to register elasticsearch repository", zap.Error(err))
		}

		enabledDatabases = append(enabledDatabases, "elasticsearch")
		logger.Info(ctx, "elasticsearch repository initialized and registered")
	}

	// 7.6. Vitess
	var vitessDB *sql.DB
	if cfg.Vitess.Enabled {
		vitessConfig := &vitess.Config{
			Host:            cfg.Vitess.Host,
			Port:            cfg.Vitess.Port,
			Keyspace:        cfg.Vitess.Keyspace,
			Username:        cfg.Vitess.Username,
			Password:        cfg.Vitess.Password,
			MaxOpenConns:    cfg.Vitess.MaxOpenConns,
			MaxIdleConns:    cfg.Vitess.MaxIdleConns,
			ConnMaxLifetime: cfg.Vitess.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.Vitess.ConnMaxIdleTime,
		}

		vitessDB, err = vitess.NewClient(ctx, vitessConfig)
		if err != nil {
			logger.Fatal(ctx, "failed to initialize vitess client", zap.Error(err))
		}

		// Register with RepositoryManager
		if err := repoManager.InitializeVitess(ctx, vitessDB); err != nil {
			logger.Fatal(ctx, "failed to register vitess repository", zap.Error(err))
		}

		enabledDatabases = append(enabledDatabases, "vitess")
		logger.Info(ctx, "vitess repository initialized and registered",
			zap.String("keyspace", cfg.Vitess.Keyspace),
		)

		defer func() {
			if err := vitessDB.Close(); err != nil {
				logger.Error(ctx, "failed to close vitess connection", zap.Error(err))
			}
		}()
	}

	// Check if at least one database is enabled
	if len(enabledDatabases) == 0 {
		logger.Fatal(ctx, "no database enabled in configuration")
	}

	logger.Info(ctx, "all enabled databases initialized",
		zap.Strings("databases", enabledDatabases),
		zap.Int("count", len(enabledDatabases)),
	)

	// ============================================
	// 8. Redis Cache Initialization
	// ============================================
	redisCache, err := cache.NewRedisCache(ctx, &cache.Config{
		Host:        cfg.Redis.Host,
		Port:        cfg.Redis.Port,
		Password:    cfg.Redis.Password,
		DB:          cfg.Redis.DB,
		MaxRetries:  cfg.Redis.MaxRetries,
		PoolSize:    cfg.Redis.PoolSize,
		MinIdleConn: cfg.Redis.MinIdleConns,
	})
	if err != nil {
		logger.Fatal(ctx, "failed to initialize redis cache", zap.Error(err))
	}
	defer redisCache.Close()
	logger.Info(ctx, "redis cache initialized",
		zap.String("host", cfg.Redis.Host),
		zap.Int("port", cfg.Redis.Port),
	)

	// ============================================
	// 9. Kafka Producer Initialization (Optional)
	// ============================================
	var kafkaProducer *kafka.Producer
	if cfg.Kafka.Enabled {
		kafkaProducer, err = kafka.NewProducer(&kafka.ProducerConfig{
			Brokers:          cfg.Kafka.Brokers,
			ClientID:         cfg.Kafka.ClientID,
			MaxMessageBytes:  1000000,
			RequiredAcks:     -1,
			Compression:      1,
			MaxRetries:       3,
			RetryBackoff:     100 * time.Millisecond,
			EnableIdempotent: true,
			UseAsync:         false,
		})
		if err != nil {
			logger.Warn(ctx, "failed to initialize kafka producer", zap.Error(err))
		} else {
			defer kafkaProducer.Close()
			logger.Info(ctx, "kafka producer initialized", zap.Strings("brokers", cfg.Kafka.Brokers))
		}
	}

	// ============================================
	// 10. UseCase Layer Initialization with RepositoryManager
	// ============================================
	documentUC := usecase.NewDocumentUseCaseWithManager(repoManager, redisCache)
	logger.Info(ctx, "use case initialized with repository manager")

	// ============================================
	// 11. HTTP Handlers Initialization
	// For health check, use first available repository
	var defaultRepo, _ = repoManager.GetRepository(enabledDatabases[0])
	healthHandler := httpHandler.NewHealthHandler(defaultRepo, redisCache, vaultClient, kafkaProducer)
	logger.Info(ctx, "http handlers initialized")

	// ============================================
	// 12. Router Setup with all 36 endpoints
	// ============================================
	r := router.SetupRouter(
		documentUC,
		healthHandler,
		redisCache,
		m,
		cfg.Observability.Tracing.Enabled,
		cfg.Observability.Metrics.Enabled,
		cfg.App.Environment,
	)

	logger.Info(ctx, "router initialized with 36 REST API endpoints supporting dynamic database selection via X-Database-Type header")

	// ============================================
	// 13. HTTP Server Configuration
	// ============================================
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.HTTP.Port),
		Handler:        r,
		ReadTimeout:    cfg.Server.HTTP.ReadTimeout,
		WriteTimeout:   cfg.Server.HTTP.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
		IdleTimeout:    120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info(ctx, "starting HTTP server with multi-database support",
			zap.Int("port", cfg.Server.HTTP.Port),
			zap.String("environment", cfg.App.Environment),
			zap.Strings("enabled_databases", enabledDatabases),
			zap.Bool("mongodb", cfg.MongoDB.Enabled),
			zap.Bool("postgresql", cfg.PostgreSQL.Enabled),
			zap.Bool("mysql", cfg.MySQL.Enabled),
			zap.Bool("cassandra", cfg.Cassandra.Enabled),
			zap.Bool("elasticsearch", cfg.Elasticsearch.Enabled),
			zap.Bool("vitess", cfg.Vitess.Enabled),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx, "failed to start HTTP server", zap.Error(err))
		}
	}()

	// ============================================
	// 14. Graceful Shutdown
	// ============================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "shutting down server gracefully...")

	// Give outstanding requests 15 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "server forced to shutdown", zap.Error(err))
	}

	logger.Info(ctx, "server exited successfully")
}
