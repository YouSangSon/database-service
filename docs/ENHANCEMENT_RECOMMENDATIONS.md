# 프로젝트 고도화 제안서

> **분석 일자**: 2025-11-12
> **대상 프로젝트**: Database Service - Enterprise Edition
> **현재 버전**: 1.0.0
> **Go 버전**: 1.25.4

## 📊 현재 구현 상태 분석

### ✅ 이미 구현된 기능 (우수한 수준)

1. **데이터베이스 추상화**
   - MongoDB (30+ 고급 메서드)
   - Vitess (30+ 고급 메서드)
   - Raw Query 실행 지원
   - Repository Pattern 적용

2. **보안 & 인프라**
   - HashiCorp Vault 통합 (동적 자격증명, Transit 암호화)
   - Redis Extended (Pub/Sub, Rate Limiting, Lock, Counter)
   - Kafka Producer (CDC 이벤트 발행)

3. **관찰성 (Observability)**
   - Prometheus Metrics
   - OpenTelemetry + Jaeger Tracing
   - Zap Structured Logging

4. **안정성**
   - Circuit Breaker
   - Retry Logic with Exponential Backoff
   - Graceful Shutdown

5. **CI/CD & 배포**
   - GitLab CI/CD 파이프라인 (Lint, Test, Build, Docker, Deploy)
   - Kubernetes Manifests (Deployment, Service, HPA, ConfigMap)
   - Docker 멀티스테이지 빌드

6. **아키텍처**
   - Clean Architecture (Domain, Application, Infrastructure, Interface)
   - DDD (Domain-Driven Design)
   - HTTP/gRPC 이중 서버

### ❌ 개선이 필요한 영역

## 🎯 고도화 제안 (우선순위별)

---

## 🔴 **Priority 1: Critical (즉시 구현 필요)**

### 1.1 테스트 커버리지 확대 ⚠️ **매우 중요**

**현재 상태**: 테스트 파일 1개만 존재, 커버리지 거의 0%

**제안 구현**:

```
test/
├── unit/                                # 유닛 테스트
│   ├── domain/
│   │   ├── entity_test.go              ✅ (기존)
│   │   └── repository_test.go          ❌ 추가 필요
│   ├── usecase/
│   │   └── document_usecase_test.go    ❌ 추가 필요
│   ├── infrastructure/
│   │   ├── mongodb_repository_test.go  ❌ 추가 필요
│   │   ├── vitess_repository_test.go   ❌ 추가 필요
│   │   └── redis_cache_test.go         ❌ 추가 필요
│   └── pkg/
│       ├── circuitbreaker_test.go      ❌ 추가 필요
│       ├── retry_test.go               ❌ 추가 필요
│       └── vault_test.go               ❌ 추가 필요
├── integration/                         # 통합 테스트
│   ├── mongodb_integration_test.go     ❌ 추가 필요
│   ├── vitess_integration_test.go      ❌ 추가 필요
│   ├── redis_integration_test.go       ❌ 추가 필요
│   ├── kafka_integration_test.go       ❌ 추가 필요
│   └── vault_integration_test.go       ❌ 추가 필요
├── e2e/                                 # E2E 테스트
│   ├── http_api_test.go                ❌ 추가 필요
│   ├── grpc_api_test.go                ❌ 추가 필요
│   └── scenarios/
│       ├── create_read_update_delete_test.go  ❌
│       └── high_load_test.go           ❌ 추가 필요
└── benchmark/                           # 벤치마크 테스트
    ├── document_benchmark_test.go      ❌ 추가 필요
    └── cache_benchmark_test.go         ❌ 추가 필요
```

**테스트 전략**:
- **유닛 테스트**: 목(Mock) 사용, 각 계층 독립 테스트, 목표 커버리지 80%+
- **통합 테스트**: Testcontainers 사용 (Docker 기반 실제 DB)
- **E2E 테스트**: 전체 플로우 테스트 (HTTP/gRPC → UseCase → Repository → DB)
- **벤치마크 테스트**: 성능 기준선 설정 및 회귀 방지

**예상 효과**:
- 코드 품질 향상
- 리팩토링 안전성 확보
- 버그 조기 발견
- 문서화 효과 (테스트가 사용 예제)

**구현 우선순위**:
1. 유닛 테스트 (UseCase, Repository)
2. 통합 테스트 (MongoDB, Vitess)
3. E2E 테스트 (API)
4. 벤치마크 테스트

---

### 1.2 API 문서화 (OpenAPI/Swagger) 📚

**현재 상태**: API 문서 없음, 개발자가 코드를 직접 읽어야 함

**제안 구현**:

```go
// cmd/api/main.go에 Swagger 추가
import (
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "github.com/YouSangSon/database-service/docs" // Swagger docs
)

// @title Database Service API
// @version 1.0
// @description Enterprise-grade database service with MongoDB and Vitess support
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https
func main() {
    // ...
    router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

```go
// internal/interfaces/http/handler/document_handler.go
// @Summary Create a new document
// @Description Create a new document in the specified collection
// @Tags documents
// @Accept json
// @Produce json
// @Param request body dto.CreateDocumentRequest true "Document creation request"
// @Success 201 {object} dto.CreateDocumentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /documents [post]
func (h *Handler) CreateDocument(c *gin.Context) {
    // ...
}
```

**추가 도구**:
- `swag init` 명령으로 자동 문서 생성
- Swagger UI (`/swagger/index.html`)
- ReDoc 통합 (더 나은 UX)

**예상 효과**:
- 프론트엔드 개발자와 협업 용이
- API 클라이언트 자동 생성 가능
- API 버전 관리 명확화
- 외부 사용자 온보딩 간소화

---

### 1.3 cmd/api와 cmd/grpc 현대화 🔧

**현재 상태**: main.go가 구 아키텍처 사용 (새 Clean Architecture 미적용)

**문제점**:
```go
// 현재 cmd/api/main.go
svc := service.NewService(db)  // ❌ 오래된 service 레이어
h := handler.NewHandler(svc)   // ❌ 오래된 handler
```

**제안 구현**:
```go
// cmd/api/main.go (개선)
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/YouSangSon/database-service/internal/config"
    "github.com/YouSangSon/database-service/internal/application/usecase"
    "github.com/YouSangSon/database-service/internal/infrastructure/persistence/mongodb"
    "github.com/YouSangSon/database-service/internal/infrastructure/cache"
    "github.com/YouSangSon/database-service/internal/infrastructure/messaging/kafka"
    "github.com/YouSangSon/database-service/internal/interfaces/http/handler"
    "github.com/YouSangSon/database-service/internal/interfaces/http/middleware"
    "github.com/YouSangSon/database-service/internal/pkg/logger"
    "github.com/YouSangSon/database-service/internal/pkg/metrics"
    "github.com/YouSangSon/database-service/internal/pkg/tracing"
    "github.com/YouSangSon/database-service/internal/pkg/vault"
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    // 1. 설정 로드
    cfg, err := config.LoadConfig("./configs", "config")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 2. Logger 초기화
    logger.Init(&logger.Config{
        Level:       cfg.App.LogLevel,
        Environment: cfg.App.Environment,
        ServiceName: cfg.App.Name,
    })

    ctx := context.Background()

    // 3. Metrics 초기화
    m := metrics.Init(cfg.App.Name)

    // 4. Tracing 초기화
    shutdown, err := tracing.Init(&tracing.Config{
        ServiceName:    cfg.App.Name,
        ServiceVersion: cfg.App.Version,
        Environment:    cfg.App.Environment,
        JaegerEndpoint: cfg.Observability.Jaeger.Endpoint,
        Enabled:        cfg.Observability.Tracing.Enabled,
    })
    if err != nil {
        logger.Fatal(ctx, "failed to initialize tracing", zap.Error(err))
    }
    defer shutdown(ctx)

    // 5. Vault 클라이언트 초기화
    var vaultClient *vault.Client
    if cfg.Vault.Enabled {
        vaultClient, err = vault.NewClient(&vault.Config{
            Address:    cfg.Vault.Address,
            Token:      cfg.Vault.Token,
            AuthMethod: cfg.Vault.AuthMethod,
            // ...
        })
        if err != nil {
            logger.Fatal(ctx, "failed to initialize vault", zap.Error(err))
        }
        defer vaultClient.Close()
    }

    // 6. MongoDB 리포지토리 초기화
    mongoRepo, err := mongodb.NewDocumentRepository(ctx, cfg.MongoDB.URI, cfg.MongoDB.Database, vaultClient)
    if err != nil {
        logger.Fatal(ctx, "failed to initialize mongodb repository", zap.Error(err))
    }
    defer mongoRepo.Close(ctx)

    // 7. Redis 캐시 초기화
    redisCache, err := cache.NewRedisCache(ctx, &cache.Config{
        Host:     cfg.Redis.Host,
        Port:     cfg.Redis.Port,
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
    })
    if err != nil {
        logger.Fatal(ctx, "failed to initialize redis", zap.Error(err))
    }
    defer redisCache.Close()

    // 8. Kafka Producer 초기화
    var kafkaProducer *kafka.Producer
    if cfg.Kafka.Enabled {
        kafkaProducer, err = kafka.NewProducer(&kafka.ProducerConfig{
            Brokers:  cfg.Kafka.Brokers,
            ClientID: cfg.App.Name,
            // ...
        })
        if err != nil {
            logger.Fatal(ctx, "failed to initialize kafka producer", zap.Error(err))
        }
        defer kafkaProducer.Close()
    }

    // 9. UseCase 초기화
    documentUC := usecase.NewDocumentUseCase(mongoRepo, redisCache, kafkaProducer)

    // 10. HTTP 핸들러 초기화
    documentHandler := handler.NewDocumentHandler(documentUC)

    // 11. Gin 라우터 설정
    gin.SetMode(gin.ReleaseMode)
    router := gin.New()

    // 미들웨어 적용
    router.Use(middleware.RequestID())
    router.Use(middleware.Logger())
    router.Use(middleware.Tracing())
    router.Use(middleware.Metrics(m))
    router.Use(middleware.Recovery())
    router.Use(middleware.CORS())

    // Health check
    router.GET("/health", handler.HealthCheck)
    router.GET("/ready", handler.ReadinessCheck(mongoRepo, redisCache))

    // Metrics endpoint
    router.GET("/metrics", gin.WrapH(promhttp.Handler()))

    // Swagger
    router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // API v1
    v1 := router.Group("/api/v1")
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

    // 12. HTTP 서버 시작
    srv := &http.Server{
        Addr:           fmt.Sprintf(":%d", cfg.Server.HTTP.Port),
        Handler:        router,
        ReadTimeout:    cfg.Server.HTTP.ReadTimeout,
        WriteTimeout:   cfg.Server.HTTP.WriteTimeout,
        MaxHeaderBytes: 1 << 20,
    }

    go func() {
        logger.Info(ctx, "starting HTTP server",
            zap.Int("port", cfg.Server.HTTP.Port))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal(ctx, "failed to start HTTP server", zap.Error(err))
        }
    }()

    // 13. Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info(ctx, "shutting down server...")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        logger.Error(ctx, "server forced to shutdown", zap.Error(err))
    }

    logger.Info(ctx, "server exited")
}
```

**예상 효과**:
- 새 아키텍처 완전 적용
- 의존성 주입 명확화
- 초기화 순서 명확화
- 에러 처리 개선

---

### 1.4 docker-compose.yml 확장 🐳

**현재 상태**: MongoDB만 있음, Redis/Kafka/Vault 없음

**제안 구현**:
```yaml
version: '3.8'

services:
  # ============================================
  # Databases
  # ============================================
  mongodb:
    image: mongo:7.0
    container_name: database-service-mongodb
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: password
      MONGO_INITDB_DATABASE: testdb
    volumes:
      - mongodb_data:/data/db
    networks:
      - database-service-network
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5

  vitess-mysql:
    image: vitess/vttestserver:latest
    container_name: database-service-vitess
    ports:
      - "15306:15306"  # vtgate MySQL protocol
      - "15000:15000"  # vtgate HTTP
    environment:
      - KEYSPACE=commerce
      - NUM_SHARDS=2
    networks:
      - database-service-network
    healthcheck:
      test: ["CMD", "mysql", "-h", "127.0.0.1", "-P", "15306", "-u", "root", "-e", "SELECT 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ============================================
  # Cache
  # ============================================
  redis:
    image: redis:7.0-alpine
    container_name: database-service-redis
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes --requirepass redispassword
    volumes:
      - redis_data:/data
    networks:
      - database-service-network
    healthcheck:
      test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ============================================
  # Message Queue
  # ============================================
  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    container_name: database-service-zookeeper
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    networks:
      - database-service-network

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    container_name: database-service-kafka
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
      - "29092:29092"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:29092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"
    networks:
      - database-service-network
    healthcheck:
      test: ["CMD", "kafka-broker-api-versions", "--bootstrap-server", "localhost:9092"]
      interval: 10s
      timeout: 10s
      retries: 5

  kafka-ui:
    image: provectuslabs/kafka-ui:latest
    container_name: database-service-kafka-ui
    depends_on:
      - kafka
    ports:
      - "8090:8080"
    environment:
      KAFKA_CLUSTERS_0_NAME: local
      KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS: kafka:9092
    networks:
      - database-service-network

  # ============================================
  # Secrets Management
  # ============================================
  vault:
    image: hashicorp/vault:1.15
    container_name: database-service-vault
    ports:
      - "8200:8200"
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: "dev-only-token"
      VAULT_DEV_LISTEN_ADDRESS: "0.0.0.0:8200"
    cap_add:
      - IPC_LOCK
    networks:
      - database-service-network
    healthcheck:
      test: ["CMD", "vault", "status"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ============================================
  # Observability
  # ============================================
  jaeger:
    image: jaegertracing/all-in-one:1.50
    container_name: database-service-jaeger
    ports:
      - "5775:5775/udp"
      - "6831:6831/udp"
      - "6832:6832/udp"
      - "5778:5778"
      - "16686:16686"  # UI
      - "14268:14268"  # Collector
      - "14250:14250"
      - "9411:9411"
    environment:
      COLLECTOR_ZIPKIN_HOST_PORT: ":9411"
    networks:
      - database-service-network

  prometheus:
    image: prom/prometheus:v2.47.0
    container_name: database-service-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./configs/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    networks:
      - database-service-network

  grafana:
    image: grafana/grafana:10.1.0
    container_name: database-service-grafana
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_USER: admin
      GF_SECURITY_ADMIN_PASSWORD: admin
      GF_USERS_ALLOW_SIGN_UP: "false"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./configs/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./configs/grafana/datasources:/etc/grafana/provisioning/datasources
    networks:
      - database-service-network

  # ============================================
  # Application Services
  # ============================================
  api:
    build:
      context: .
      dockerfile: Dockerfile.http
    container_name: database-service-api
    ports:
      - "8080:8080"
      - "9091:9091"  # Metrics
    environment:
      # App
      - APP_NAME=database-service
      - APP_VERSION=1.0.0-local
      - APP_ENVIRONMENT=local
      - APP_DEBUG=true

      # Server
      - APP_SERVER_HTTP_PORT=8080
      - APP_SERVER_GRPC_PORT=9090

      # MongoDB
      - APP_MONGODB_ENABLED=true
      - APP_MONGODB_URI=mongodb://admin:password@mongodb:27017
      - APP_MONGODB_DATABASE=testdb
      - APP_MONGODB_USE_VAULT=false

      # Vitess
      - APP_VITESS_ENABLED=true
      - APP_VITESS_HOST=vitess-mysql
      - APP_VITESS_PORT=15306
      - APP_VITESS_USER=root
      - APP_VITESS_DATABASE=commerce

      # Redis
      - APP_REDIS_ENABLED=true
      - APP_REDIS_HOST=redis
      - APP_REDIS_PORT=6379
      - APP_REDIS_PASSWORD=redispassword

      # Kafka
      - APP_KAFKA_ENABLED=true
      - APP_KAFKA_BROKERS=kafka:9092

      # Vault
      - APP_VAULT_ENABLED=true
      - APP_VAULT_ADDRESS=http://vault:8200
      - APP_VAULT_TOKEN=dev-only-token
      - APP_VAULT_AUTH_METHOD=token

      # Observability
      - APP_OBSERVABILITY_TRACING_ENABLED=true
      - APP_OBSERVABILITY_JAEGER_ENDPOINT=http://jaeger:14268/api/traces
      - APP_OBSERVABILITY_METRICS_ENABLED=true
    depends_on:
      mongodb:
        condition: service_healthy
      redis:
        condition: service_healthy
      kafka:
        condition: service_healthy
      vault:
        condition: service_healthy
    networks:
      - database-service-network
    restart: unless-stopped

  grpc:
    build:
      context: .
      dockerfile: Dockerfile.grpc
    container_name: database-service-grpc
    ports:
      - "9090:9090"
    environment:
      # (동일한 환경변수)
      - APP_SERVER_GRPC_PORT=9090
      # ...
    depends_on:
      mongodb:
        condition: service_healthy
      redis:
        condition: service_healthy
      kafka:
        condition: service_healthy
      vault:
        condition: service_healthy
    networks:
      - database-service-network
    restart: unless-stopped

volumes:
  mongodb_data:
  redis_data:
  prometheus_data:
  grafana_data:

networks:
  database-service-network:
    driver: bridge
```

**예상 효과**:
- 로컬 개발 환경 완전 구축
- 전체 스택 테스트 가능
- 새로운 개발자 온보딩 시간 단축
- CI/CD 없이도 로컬에서 E2E 테스트 가능

---

## 🟠 **Priority 2: High (조만간 구현 권장)**

### 2.1 Kafka Consumer 구현 📥

**현재 상태**: Producer만 있고 Consumer 없음 (이벤트를 발행만 하고 소비하지 않음)

**제안 구현**:
```go
// internal/infrastructure/messaging/kafka/consumer.go
package kafka

type Consumer struct {
    consumer sarama.ConsumerGroup
    config   *ConsumerConfig
    handlers map[string]MessageHandler
}

type ConsumerConfig struct {
    Brokers       []string
    GroupID       string
    Topics        []string
    InitialOffset string
}

type MessageHandler func(ctx context.Context, msg *sarama.ConsumerMessage) error

func NewConsumer(cfg *ConsumerConfig) (*Consumer, error) {
    // Sarama ConsumerGroup 초기화
}

func (c *Consumer) RegisterHandler(topic string, handler MessageHandler) {
    c.handlers[topic] = handler
}

func (c *Consumer) Start(ctx context.Context) error {
    // Consumer 시작, 메시지 수신 및 핸들러 호출
}
```

**사용 사례**:
1. **Analytics Service**: documents.created/updated/deleted 이벤트 소비하여 분석
2. **Audit Log Service**: 모든 변경 이력 기록
3. **Search Indexer**: Elasticsearch/MeiliSearch에 문서 인덱싱
4. **Notification Service**: 특정 이벤트 발생 시 알림 전송

**예상 효과**:
- 이벤트 기반 아키텍처 완성
- 서비스 간 느슨한 결합
- 확장성 향상

---

### 2.2 Rate Limiting 미들웨어 구현 🚦

**현재 상태**: `// TODO: Redis 기반 rate limiting 구현` 주석만 있음

**제안 구현**:
```go
// internal/interfaces/http/middleware/ratelimit.go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/YouSangSon/database-service/internal/infrastructure/cache"
)

// RateLimit는 IP 기반 rate limiting 미들웨어입니다
func RateLimit(redisCache *cache.RedisExtended, limit int64, window time.Duration) gin.HandlerFunc {
    rateLimiter := redisCache.NewRateLimiter("api:ratelimit")

    return func(c *gin.Context) {
        clientIP := c.ClientIP()

        allowed, err := rateLimiter.Allow(c.Request.Context(), clientIP, limit, window)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "rate limit check failed",
            })
            c.Abort()
            return
        }

        if !allowed {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "rate limit exceeded",
                "retry_after": window.Seconds(),
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// RateLimitByAPIKey는 API Key 기반 rate limiting입니다
func RateLimitByAPIKey(redisCache *cache.RedisExtended, limits map[string]int64, window time.Duration) gin.HandlerFunc {
    // API Key별 다른 제한 설정
}
```

**적용**:
```go
// cmd/api/main.go
router.Use(middleware.RateLimit(redisCache, 100, time.Minute)) // 분당 100 요청
```

**예상 효과**:
- DDoS 방어
- 악의적 사용자 차단
- 공정한 리소스 분배
- API 남용 방지

---

### 2.3 Health Check 고도화 🏥

**현재 상태**: 기본 health check만 있음

**제안 구현**:
```go
// internal/interfaces/http/handler/health.go
package handler

type HealthResponse struct {
    Status    string            `json:"status"` // "healthy", "degraded", "unhealthy"
    Timestamp time.Time         `json:"timestamp"`
    Version   string            `json:"version"`
    Checks    map[string]Check  `json:"checks"`
}

type Check struct {
    Status   string        `json:"status"`
    Message  string        `json:"message,omitempty"`
    Duration time.Duration `json:"duration_ms"`
}

func HealthCheck(
    mongoRepo repository.DocumentRepository,
    vitessRepo repository.DocumentRepository,
    redisCache repository.CacheRepository,
    kafkaProducer *kafka.Producer,
    vaultClient *vault.Client,
) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        response := HealthResponse{
            Status:    "healthy",
            Timestamp: time.Now(),
            Version:   "1.0.0",
            Checks:    make(map[string]Check),
        }

        // MongoDB health check
        start := time.Now()
        if err := mongoRepo.Ping(ctx); err != nil {
            response.Checks["mongodb"] = Check{
                Status:   "unhealthy",
                Message:  err.Error(),
                Duration: time.Since(start),
            }
            response.Status = "unhealthy"
        } else {
            response.Checks["mongodb"] = Check{
                Status:   "healthy",
                Duration: time.Since(start),
            }
        }

        // Vitess health check
        // Redis health check
        // Kafka health check
        // Vault health check

        statusCode := http.StatusOK
        if response.Status == "unhealthy" {
            statusCode = http.StatusServiceUnavailable
        }

        c.JSON(statusCode, response)
    }
}

// ReadinessCheck는 Kubernetes Readiness Probe용입니다
func ReadinessCheck(/* ... */) gin.HandlerFunc {
    // 서비스가 트래픽을 받을 준비가 되었는지 확인
}

// LivenessCheck는 Kubernetes Liveness Probe용입니다
func LivenessCheck() gin.HandlerFunc {
    // 서비스가 살아있는지 확인 (간단한 응답)
    return func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "alive"})
    }
}
```

**예상 효과**:
- 장애 조기 발견
- 의존성 상태 모니터링
- Kubernetes 통합 개선
- 운영 가시성 향상

---

### 2.4 Helm Charts 작성 ⛵

**현재 상태**: Raw Kubernetes manifests만 있음

**제안 구현**:
```
deployments/helm/
├── Chart.yaml
├── values.yaml
├── values-dev.yaml
├── values-staging.yaml
├── values-production.yaml
└── templates/
    ├── deployment.yaml
    ├── service.yaml
    ├── ingress.yaml
    ├── configmap.yaml
    ├── secret.yaml
    ├── hpa.yaml
    ├── pdb.yaml
    ├── serviceaccount.yaml
    └── _helpers.tpl
```

```yaml
# values.yaml
replicaCount: 3

image:
  repository: registry.gitlab.com/yousangson/database-service
  tag: "latest"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  http:
    port: 8080
  grpc:
    port: 9090

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: api.database-service.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: database-service-tls
      hosts:
        - api.database-service.example.com

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 250m
    memory: 256Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

mongodb:
  enabled: true
  uri: "mongodb://mongodb-cluster:27017"
  database: "production"

vault:
  enabled: true
  address: "https://vault.production.svc.cluster.local:8200"
  authMethod: "kubernetes"
  role: "database-service"
```

**사용**:
```bash
# Development
helm install database-service ./deployments/helm \
  -f deployments/helm/values-dev.yaml \
  -n development

# Production
helm upgrade --install database-service ./deployments/helm \
  -f deployments/helm/values-production.yaml \
  -n production
```

**예상 효과**:
- 환경별 배포 간소화
- 설정 관리 용이
- 롤백 기능
- 버전 관리 체계화

---

## 🟡 **Priority 3: Medium (향후 고려)**

### 3.1 CQRS (Command Query Responsibility Segregation) 패턴

**현재 상태**: 단일 리포지토리로 읽기/쓰기 처리

**제안 개념**:
```go
// internal/domain/repository/document_repository.go
type DocumentCommandRepository interface {
    // Write operations
    Save(ctx context.Context, doc *entity.Document) error
    Update(ctx context.Context, doc *entity.Document) error
    Delete(ctx context.Context, id string) error
}

type DocumentQueryRepository interface {
    // Read operations (read replicas, materialized views)
    FindByID(ctx context.Context, collection, id string) (*entity.Document, error)
    FindAll(ctx context.Context, collection string, opts *QueryOptions) ([]*entity.Document, error)
    Aggregate(ctx context.Context, collection string, pipeline []interface{}) ([]map[string]interface{}, error)
}
```

**장점**:
- 읽기/쓰기 최적화 분리
- Read Replica 활용 가능
- 확장성 향상

---

### 3.2 Event Sourcing

**개념**: 상태 변경을 이벤트 스트림으로 저장

```go
type Event struct {
    EventID     string
    AggregateID string
    EventType   string
    EventData   map[string]interface{}
    Version     int
    Timestamp   time.Time
}

type EventStore interface {
    Append(ctx context.Context, event *Event) error
    Load(ctx context.Context, aggregateID string) ([]*Event, error)
    LoadFrom(ctx context.Context, aggregateID string, version int) ([]*Event, error)
}
```

**장점**:
- 완벽한 감사 추적
- 시간 여행 (특정 시점 상태 재구성)
- 이벤트 재생 가능

---

### 3.3 GraphQL API

**개념**: REST API 외 GraphQL 엔드포인트 제공

```graphql
type Query {
  document(collection: String!, id: ID!): Document
  documents(collection: String!, filter: FilterInput, limit: Int, offset: Int): DocumentConnection
}

type Mutation {
  createDocument(input: CreateDocumentInput!): Document!
  updateDocument(id: ID!, input: UpdateDocumentInput!): Document!
  deleteDocument(id: ID!): Boolean!
}

type Subscription {
  documentChanged(collection: String!): DocumentChangeEvent!
}
```

**장점**:
- 클라이언트가 필요한 데이터만 요청
- Over-fetching/Under-fetching 문제 해결
- 실시간 구독 지원

---

### 3.4 WebSocket 실시간 알림

**개념**: 문서 변경 시 실시간 알림

```go
// internal/interfaces/websocket/hub.go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case message := <-h.broadcast:
            for client := range h.clients {
                client.send <- message
            }
        }
    }
}
```

**사용 사례**:
- 실시간 대시보드
- 협업 편집
- 알림 시스템

---

### 3.5 Admin CLI 도구

**개념**: 관리자용 CLI 도구

```bash
# 사용 예시
database-service-cli migrate --from=mongodb --to=vitess
database-service-cli backup --collection=users --output=backup.tar.gz
database-service-cli stats --collection=orders --format=json
database-service-cli reindex --collection=products
```

**장점**:
- 운영 작업 자동화
- 데이터 마이그레이션 간소화
- 장애 대응 신속화

---

### 3.6 Multi-tenancy 지원

**개념**: 여러 테넌트(고객)가 하나의 서비스 인스턴스 공유

```go
type TenantContext struct {
    TenantID   string
    TenantName string
    Database   string // 테넌트별 DB 분리
    Quotas     *Quotas
}

func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := c.GetHeader("X-Tenant-ID")
        // Tenant 정보 조회 및 context에 저장
        c.Set("tenant", tenant)
        c.Next()
    }
}
```

**격리 전략**:
1. **Database per Tenant**: 각 테넌트가 별도 DB 사용
2. **Schema per Tenant**: 같은 DB, 별도 스키마
3. **Shared Database**: 같은 DB/스키마, tenant_id로 구분

---

### 3.7 Prometheus AlertManager Rules

**개념**: 자동 알림 규칙

```yaml
# configs/prometheus/alerts.yml
groups:
  - name: database-service
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: |
          rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} requests/sec"

      - alert: HighLatency
        expr: |
          histogram_quantile(0.99, http_request_duration_seconds_bucket) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "P99 latency is {{ $value }} seconds"

      - alert: DatabaseConnectionFailed
        expr: |
          up{job="database-service"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database service is down"
```

---

### 3.8 Grafana Dashboards

**개념**: 사전 구성된 대시보드

```json
// configs/grafana/dashboards/database-service.json
{
  "dashboard": {
    "title": "Database Service Overview",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{ method }} {{ endpoint }}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])",
            "legendFormat": "Errors"
          }
        ]
      },
      {
        "title": "Latency (P50, P95, P99)",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, http_request_duration_seconds_bucket)",
            "legendFormat": "P50"
          }
        ]
      }
    ]
  }
}
```

---

## 🟢 **Priority 4: Low (장기 로드맵)**

### 4.1 Service Mesh (Istio/Linkerd) 통합
### 4.2 PostgreSQL/MySQL 네이티브 지원
### 4.3 Data Migration Tools
### 4.4 Backup/Restore Automation
### 4.5 Load Testing Suite (k6/Locust)

---

## 📊 구현 로드맵 제안

### Phase 1: 안정화 (1-2주)
- [ ] 테스트 커버리지 80% 달성
- [ ] API 문서화 (Swagger)
- [ ] cmd/main.go 현대화
- [ ] docker-compose.yml 확장

### Phase 2: 기능 확장 (2-3주)
- [ ] Kafka Consumer 구현
- [ ] Rate Limiting 미들웨어
- [ ] Health Check 고도화
- [ ] Helm Charts 작성

### Phase 3: 고급 기능 (1-2개월)
- [ ] CQRS 패턴 적용
- [ ] Event Sourcing (선택)
- [ ] GraphQL API
- [ ] WebSocket 알림

### Phase 4: 운영 최적화 (지속적)
- [ ] Prometheus AlertManager
- [ ] Grafana Dashboards
- [ ] Admin CLI
- [ ] Multi-tenancy

---

## 💡 우선순위 결정 가이드

1. **테스트 커버리지**: 모든 신규 기능 개발 전 필수
2. **API 문서화**: 팀 협업 개선을 위해 조속히 구현
3. **Rate Limiting**: 프로덕션 배포 전 필수
4. **Kafka Consumer**: 이벤트 기반 아키텍처 완성을 위해 중요
5. **나머지**: 비즈니스 요구사항에 따라 우선순위 조정

---

## 🎓 학습 리소스

- **Testing in Go**: "Learn Go with Tests" (https://quii.gitbook.io/learn-go-with-tests/)
- **CQRS & Event Sourcing**: Martin Fowler's blog
- **Kubernetes Patterns**: "Kubernetes Patterns" by Bilgin Ibryam
- **Microservices**: "Building Microservices" by Sam Newman
- **Observability**: "Distributed Systems Observability" by Cindy Sridharan

---

## 📞 문의

질문이나 제안 사항이 있으시면 이슈를 생성해주세요.
