package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/YouSangSon/database-service/internal/domain/repository"
	"github.com/YouSangSon/database-service/internal/infrastructure/messaging/kafka"
	"github.com/YouSangSon/database-service/internal/pkg/vault"
	"github.com/gin-gonic/gin"
)

// HealthHandler는 헬스체크 핸들러입니다
type HealthHandler struct {
	mongoRepo     repository.DocumentRepository
	redisCache    repository.CacheRepository
	vaultClient   *vault.Client
	kafkaProducer *kafka.Producer
}

// NewHealthHandler는 새로운 HealthHandler를 생성합니다
func NewHealthHandler(
	mongoRepo repository.DocumentRepository,
	redisCache repository.CacheRepository,
	vaultClient *vault.Client,
	kafkaProducer *kafka.Producer,
) *HealthHandler {
	return &HealthHandler{
		mongoRepo:     mongoRepo,
		redisCache:    redisCache,
		vaultClient:   vaultClient,
		kafkaProducer: kafkaProducer,
	}
}

// HealthResponse는 헬스체크 응답입니다
type HealthResponse struct {
	Status    string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck는 개별 의존성 체크 결과입니다
type HealthCheck struct {
	Status   string  `json:"status"` // "healthy", "unhealthy"
	Message  string  `json:"message,omitempty"`
	Duration float64 `json:"duration_ms"`
}

// Health godoc
// @Summary      Health check
// @Description  Check the health status of the service and its dependencies
// @Tags         health
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      503  {object}  HealthResponse
// @Router       /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Checks:    make(map[string]HealthCheck),
	}

	// MongoDB Health Check
	mongoStart := time.Now()
	if err := h.checkMongoDB(ctx); err != nil {
		response.Checks["mongodb"] = HealthCheck{
			Status:   "unhealthy",
			Message:  err.Error(),
			Duration: float64(time.Since(mongoStart).Milliseconds()),
		}
		response.Status = "unhealthy"
	} else {
		response.Checks["mongodb"] = HealthCheck{
			Status:   "healthy",
			Duration: float64(time.Since(mongoStart).Milliseconds()),
		}
	}

	// Redis Health Check
	redisStart := time.Now()
	if err := h.checkRedis(ctx); err != nil {
		response.Checks["redis"] = HealthCheck{
			Status:   "unhealthy",
			Message:  err.Error(),
			Duration: float64(time.Since(redisStart).Milliseconds()),
		}
		if response.Status == "healthy" {
			response.Status = "degraded"
		}
	} else {
		response.Checks["redis"] = HealthCheck{
			Status:   "healthy",
			Duration: float64(time.Since(redisStart).Milliseconds()),
		}
	}

	// Vault Health Check (optional)
	if h.vaultClient != nil {
		vaultStart := time.Now()
		if err := h.checkVault(ctx); err != nil {
			response.Checks["vault"] = HealthCheck{
				Status:   "unhealthy",
				Message:  err.Error(),
				Duration: float64(time.Since(vaultStart).Milliseconds()),
			}
			if response.Status == "healthy" {
				response.Status = "degraded"
			}
		} else {
			response.Checks["vault"] = HealthCheck{
				Status:   "healthy",
				Duration: float64(time.Since(vaultStart).Milliseconds()),
			}
		}
	}

	// Kafka Health Check (optional)
	if h.kafkaProducer != nil {
		// Kafka producer doesn't have a direct health check
		// We can check if it's not nil
		response.Checks["kafka"] = HealthCheck{
			Status:   "healthy",
			Message:  "producer initialized",
			Duration: 0,
		}
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// Ready godoc
// @Summary      Readiness check
// @Description  Check if the service is ready to accept traffic (Kubernetes readiness probe)
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx := c.Request.Context()

	// Check critical dependencies only
	if err := h.checkMongoDB(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "mongodb connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now(),
	})
}

// checkMongoDB checks MongoDB connection
func (h *HealthHandler) checkMongoDB(ctx context.Context) error {
	// Try to count documents in a test collection
	_, err := h.mongoRepo.Count(ctx, "__health_check__")
	return err
}

// checkRedis checks Redis connection
func (h *HealthHandler) checkRedis(ctx context.Context) error {
	// Try to set and get a test key
	testKey := "__health_check__"
	testValue := "ok"

	if err := h.redisCache.Set(ctx, testKey, testValue, 10); err != nil {
		return err
	}

	var result string
	if err := h.redisCache.Get(ctx, testKey, &result); err != nil {
		return err
	}

	// Clean up
	h.redisCache.Delete(ctx, testKey)

	return nil
}

// checkVault checks Vault connection
func (h *HealthHandler) checkVault(ctx context.Context) error {
	return h.vaultClient.HealthCheck(ctx)
}
