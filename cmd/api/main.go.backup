package main

import (
	"context"
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
	"github.com/YouSangSon/database-service/internal/infrastructure/persistence/mongodb"
	httpHandler "github.com/YouSangSon/database-service/internal/interfaces/http/handler"
	"github.com/YouSangSon/database-service/internal/interfaces/http/middleware"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"github.com/YouSangSon/database-service/internal/pkg/vault"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	// Uncomment after running: go get -u github.com/swaggo/swag/cmd/swag github.com/swaggo/gin-swagger github.com/swaggo/files
	// _ "github.com/YouSangSon/database-service/docs"
	// ginSwagger "github.com/swaggo/gin-swagger"
	// swaggerFiles "github.com/swaggo/files"
)

// @title Database Service API
// @version 1.0
// @description Enterprise-grade database service with MongoDB/Vitess support, Clean Architecture, and comprehensive observability
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/YouSangSon/database-service
// @contact.email support@database-service.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// @x-extension-openapi {"example": "value on a json format"}

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
		Level:       cfg.App.LogLevel,
		Environment: cfg.App.Environment,
		ServiceName: cfg.App.Name,
		Version:     cfg.App.Version,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	logger.Info(ctx, "starting database service",
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
			JaegerEndpoint: cfg.Observability.Jaeger.Endpoint,
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
		logger.Info(ctx, "tracing initialized", zap.String("jaeger_endpoint", cfg.Observability.Jaeger.Endpoint))
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

		// Health check
		if err := vaultClient.HealthCheck(ctx); err != nil {
			logger.Warn(ctx, "vault health check failed", zap.Error(err))
		} else {
			logger.Info(ctx, "vault client initialized successfully")
		}
	}

	// ============================================
	// 6. MongoDB Repository Initialization
	// ============================================
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

	mongoRepo, err := mongodb.NewDocumentRepository(ctx, mongoURI, cfg.MongoDB.Database, vaultClient)
	if err != nil {
		logger.Fatal(ctx, "failed to initialize mongodb repository", zap.Error(err))
	}
	defer func() {
		if err := mongoRepo.Close(ctx); err != nil {
			logger.Error(ctx, "failed to close mongodb connection", zap.Error(err))
		}
	}()
	logger.Info(ctx, "mongodb repository initialized",
		zap.String("database", cfg.MongoDB.Database),
	)

	// ============================================
	// 7. Redis Cache Initialization
	// ============================================
	redisCache, err := cache.NewRedisCache(ctx, &cache.Config{
		Host:        cfg.Redis.Host,
		Port:        cfg.Redis.Port,
		Password:    cfg.Redis.Password,
		DB:          cfg.Redis.DB,
		MaxRetries:  cfg.Redis.MaxRetries,
		PoolSize:    cfg.Redis.PoolSize,
		MinIdleConn: cfg.Redis.MinIdleConn,
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
	// 8. Kafka Producer Initialization (Optional)
	// ============================================
	var kafkaProducer *kafka.Producer
	var cdcPublisher *kafka.CDCPublisher
	if cfg.Kafka.Enabled {
		kafkaProducer, err = kafka.NewProducer(&kafka.ProducerConfig{
			Brokers:          cfg.Kafka.Brokers,
			ClientID:         cfg.Kafka.ClientID,
			MaxMessageBytes:  1000000,
			RequiredAcks:     -1, // Wait for all replicas
			Compression:      1,  // Gzip
			MaxRetries:       3,
			RetryBackoff:     100 * time.Millisecond,
			EnableIdempotent: true,
			UseAsync:         false,
		})
		if err != nil {
			logger.Warn(ctx, "failed to initialize kafka producer", zap.Error(err))
		} else {
			defer kafkaProducer.Close()
			cdcPublisher = kafka.NewCDCPublisher(
				kafkaProducer,
				cfg.Kafka.Topics.Created,
				cfg.Kafka.Topics.Updated,
				cfg.Kafka.Topics.Deleted,
			)
			logger.Info(ctx, "kafka producer initialized",
				zap.Strings("brokers", cfg.Kafka.Brokers),
			)
		}
	}

	// ============================================
	// 9. UseCase Layer Initialization
	// ============================================
	documentUC := usecase.NewDocumentUseCase(mongoRepo, redisCache)
	logger.Info(ctx, "use cases initialized")

	// ============================================
	// 10. HTTP Handlers Initialization
	// ============================================
	documentHandler := httpHandler.NewDocumentHandler(documentUC)
	healthHandler := httpHandler.NewHealthHandler(mongoRepo, redisCache, vaultClient, kafkaProducer)
	logger.Info(ctx, "http handlers initialized")

	// ============================================
	// 11. Gin Router Setup
	// ============================================
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global Middlewares
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	if cfg.Observability.Tracing.Enabled {
		router.Use(middleware.Tracing())
	}

	if cfg.Observability.Metrics.Enabled {
		router.Use(middleware.Metrics(m))
	}

	// Extended Redis client for rate limiting
	redisExtended := cache.NewRedisExtended(redisCache.Client())

	// Apply rate limiting to API routes
	apiRateLimit := middleware.RateLimit(redisExtended, 100, time.Minute)

	// ============================================
	// 12. Routes Definition
	// ============================================

	// Health & Metrics Endpoints (no rate limit)
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger Documentation (uncomment after running swag init)
	// router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 Group with rate limiting
	v1 := router.Group("/api/v1")
	v1.Use(apiRateLimit)
	{
		documents := v1.Group("/documents")
		{
			documents.POST("", documentHandler.Create)
			documents.GET("/:collection/:id", documentHandler.GetByID)
			documents.PUT("/:collection/:id", documentHandler.Update)
			documents.DELETE("/:collection/:id", documentHandler.Delete)
			documents.GET("/:collection", documentHandler.List)
			documents.POST("/:collection/aggregate", documentHandler.Aggregate)
			documents.POST("/raw-query", documentHandler.ExecuteRawQuery)
		}
	}

	// ============================================
	// 13. HTTP Server Configuration
	// ============================================
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.HTTP.Port),
		Handler:        router,
		ReadTimeout:    cfg.Server.HTTP.ReadTimeout,
		WriteTimeout:   cfg.Server.HTTP.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
		IdleTimeout:    120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info(ctx, "starting HTTP server",
			zap.Int("port", cfg.Server.HTTP.Port),
			zap.String("environment", cfg.App.Environment),
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
