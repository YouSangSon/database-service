package router

import (
	"time"

	"github.com/YouSangSon/database-service/internal/application/usecase"
	"github.com/YouSangSon/database-service/internal/infrastructure/cache"
	httpHandler "github.com/YouSangSon/database-service/internal/interfaces/http/handler"
	"github.com/YouSangSon/database-service/internal/interfaces/http/middleware"
	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter sets up all routes for the API server
func SetupRouter(
	documentUC *usecase.DocumentUseCase,
	healthHandler *httpHandler.HealthHandler,
	redisCache *cache.RedisCache,
	m *metrics.Metrics,
	enableTracing bool,
	enableMetrics bool,
	environment string,
) *gin.Engine {
	// Set Gin mode based on environment
	if environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global Middlewares
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	if enableTracing {
		router.Use(middleware.Tracing())
	}

	if enableMetrics {
		router.Use(middleware.Metrics(m))
	}

	// Extended Redis client for rate limiting
	redisExtended := cache.NewRedisExtended(redisCache.Client())

	// Rate limiting middleware
	apiRateLimit := middleware.RateLimit(redisExtended, 1000, time.Minute)

	// Initialize handlers
	documentHandler := httpHandler.NewDocumentHandler(documentUC)
	documentHandlerExt := httpHandler.NewDocumentHandlerExtended(documentUC)

	// ============================================
	// Health & Metrics Endpoints (no rate limit)
	// ============================================
	router.GET("/health", documentHandlerExt.Health)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ============================================
	// API v1 Group with rate limiting and database selection
	// ============================================
	v1 := router.Group("/api/v1")
	v1.Use(apiRateLimit)
	v1.Use(middleware.DatabaseSelector())
	{
		// ========================================
		// Basic CRUD Operations
		// ========================================
		documents := v1.Group("/documents")
		{
			// Create document
			documents.POST("", documentHandler.Create)

			// Read document
			documents.GET("/:collection/:id", documentHandler.GetByID)

			// Update document
			documents.PUT("/:collection/:id", documentHandler.Update)

			// Replace document
			documents.PUT("/:collection/:id/replace", documentHandlerExt.Replace)

			// Delete document
			documents.DELETE("/:collection/:id", documentHandler.Delete)
		}

		// ========================================
		// Query & Search Operations
		// ========================================
		documents.GET("/:collection", documentHandler.List)
		documents.POST("/:collection/search", documentHandlerExt.Search)
		documents.POST("/:collection/count", documentHandlerExt.Count)
		documents.GET("/:collection/count/estimate", documentHandlerExt.EstimatedCount)

		// ========================================
		// Atomic Operations
		// ========================================
		documents.POST("/:collection/:id/find-and-update", documentHandlerExt.FindAndUpdate)
		documents.POST("/:collection/:id/find-and-replace", documentHandlerExt.FindAndReplace)
		documents.POST("/:collection/:id/find-and-delete", documentHandlerExt.FindAndDelete)
		documents.POST("/:collection/upsert", documentHandlerExt.Upsert)

		// ========================================
		// Aggregations
		// ========================================
		documents.POST("/:collection/aggregate", documentHandler.Aggregate)
		documents.POST("/:collection/distinct", documentHandlerExt.Distinct)

		// ========================================
		// Bulk Operations
		// ========================================
		bulk := v1.Group("/documents/bulk")
		{
			bulk.POST("/insert", documentHandlerExt.BulkInsert)
		}
		documents.POST("/:collection/update-many", documentHandlerExt.UpdateMany)
		documents.POST("/:collection/delete-many", documentHandlerExt.DeleteMany)
		v1.POST("/documents/bulk/write", documentHandlerExt.BulkWrite)

		// ========================================
		// Index Management
		// ========================================
		indexes := v1.Group("/indexes")
		{
			indexes.POST("/:collection", documentHandlerExt.CreateIndex)
			indexes.POST("/:collection/bulk", documentHandlerExt.CreateIndexes)
			indexes.DELETE("/:collection/:index_name", documentHandlerExt.DropIndex)
			indexes.GET("/:collection", documentHandlerExt.ListIndexes)
		}

		// ========================================
		// Collection Management
		// ========================================
		collections := v1.Group("/collections")
		{
			collections.POST("", documentHandlerExt.CreateCollection)
			collections.DELETE("/:collection", documentHandlerExt.DropCollection)
			collections.POST("/:old_name/rename", documentHandlerExt.RenameCollection)
			collections.GET("", documentHandlerExt.ListCollections)
			collections.GET("/:collection/exists", documentHandlerExt.CollectionExists)
		}

		// ========================================
		// Transactions
		// ========================================
		transactions := v1.Group("/transactions")
		{
			transactions.POST("/execute", documentHandlerExt.ExecuteTransaction)
		}

		// ========================================
		// Raw Query Execution
		// ========================================
		query := v1.Group("/query")
		{
			query.POST("/raw", documentHandlerExt.ExecuteRaw)
			query.POST("/raw/typed", documentHandlerExt.ExecuteRawTyped)
		}

		// ========================================
		// Health & Monitoring (with DB selector)
		// ========================================
		health := v1.Group("/health")
		{
			health.GET("/database/:db_type", documentHandlerExt.DatabaseHealth)
		}

		// Metrics endpoint
		v1.GET("/metrics", documentHandlerExt.GetMetrics)

		// Stats endpoints
		stats := v1.Group("/stats")
		{
			stats.GET("/database/:db_type", documentHandlerExt.GetDatabaseStats)
			stats.GET("/collection/:collection", documentHandlerExt.GetCollectionStats)
		}
	}

	return router
}
