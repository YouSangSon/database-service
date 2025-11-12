package main

import (
	"context"
	"fmt"
	"net"
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
	grpcHandler "github.com/YouSangSon/database-service/internal/interfaces/grpc/handler"
	"github.com/YouSangSon/database-service/internal/interfaces/grpc/interceptor"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"github.com/YouSangSon/database-service/internal/pkg/vault"
	pb "github.com/YouSangSon/database-service/proto/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

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
		ServiceName: cfg.App.Name + "-grpc",
		Version:     cfg.App.Version,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	logger.Info(ctx, "starting gRPC database service",
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.App.Environment),
		zap.String("go_version", runtime.Version()),
	)

	// ============================================
	// 3. Metrics Initialization
	// ============================================
	m := metrics.Init(cfg.App.Name + "-grpc")
	logger.Info(ctx, "metrics initialized")

	// ============================================
	// 4. Tracing Initialization
	// ============================================
	var tracingShutdown func(context.Context) error
	if cfg.Observability.Tracing.Enabled {
		tracingShutdown, err = tracing.Init(&tracing.Config{
			ServiceName:    cfg.App.Name + "-grpc",
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
			ClientID:         cfg.Kafka.ClientID + "-grpc",
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
	// 10. gRPC Handler Initialization
	// ============================================
	databaseHandler := grpcHandler.NewDatabaseHandler(documentUC)
	logger.Info(ctx, "gRPC handlers initialized")

	// ============================================
	// 11. gRPC Server Setup with Interceptors
	// ============================================
	var grpcServerOptions []grpc.ServerOption

	// Unary interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		interceptor.UnaryRecoveryInterceptor(),
		interceptor.UnaryLoggingInterceptor(),
	}

	if cfg.Observability.Tracing.Enabled {
		unaryInterceptors = append(unaryInterceptors, interceptor.UnaryTracingInterceptor())
	}

	if cfg.Observability.Metrics.Enabled {
		unaryInterceptors = append(unaryInterceptors, interceptor.UnaryMetricsInterceptor(m))
	}

	grpcServerOptions = append(grpcServerOptions,
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
	)

	// Stream interceptors
	streamInterceptors := []grpc.StreamServerInterceptor{
		interceptor.StreamRecoveryInterceptor(),
		interceptor.StreamLoggingInterceptor(),
	}

	if cfg.Observability.Tracing.Enabled {
		streamInterceptors = append(streamInterceptors, interceptor.StreamTracingInterceptor())
	}

	if cfg.Observability.Metrics.Enabled {
		streamInterceptors = append(streamInterceptors, interceptor.StreamMetricsInterceptor(m))
	}

	grpcServerOptions = append(grpcServerOptions,
		grpc.ChainStreamInterceptor(streamInterceptors...),
	)

	// Create gRPC server
	grpcServer := grpc.NewServer(grpcServerOptions...)

	// Register service
	pb.RegisterDatabaseServiceServer(grpcServer, databaseHandler)

	// Enable reflection for gRPC clients (grpcurl, etc.)
	reflection.Register(grpcServer)

	logger.Info(ctx, "gRPC server configured with interceptors")

	// ============================================
	// 12. Start gRPC Server
	// ============================================
	addr := fmt.Sprintf(":%d", cfg.Server.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal(ctx, "failed to listen", zap.Error(err))
	}

	// Start server in goroutine
	go func() {
		logger.Info(ctx, "starting gRPC server",
			zap.Int("port", cfg.Server.GRPC.Port),
			zap.String("environment", cfg.App.Environment),
		)

		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal(ctx, "failed to serve gRPC", zap.Error(err))
		}
	}()

	// ============================================
	// 13. Graceful Shutdown
	// ============================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "shutting down gRPC server gracefully...")

	// Graceful stop - waits for in-flight RPCs to complete
	// Use GracefulStop() instead of Stop() to allow graceful shutdown
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	// Wait for graceful stop with timeout
	select {
	case <-done:
		logger.Info(ctx, "gRPC server stopped gracefully")
	case <-time.After(15 * time.Second):
		logger.Warn(ctx, "gRPC server shutdown timeout, forcing stop")
		grpcServer.Stop()
	}

	logger.Info(ctx, "gRPC server exited successfully")
}
