package middleware

import (
	"net/http"
	"time"

	"github.com/YouSangSon/database-service/internal/infrastructure/cache"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimit는 IP 기반 rate limiting 미들웨어입니다
func RateLimit(redisExtended *cache.RedisExtended, limit int64, window time.Duration) gin.HandlerFunc {
	rateLimiter := redisExtended.NewRateLimiter("api:ratelimit")

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		clientIP := c.ClientIP()

		// Check rate limit
		allowed, err := rateLimiter.Allow(ctx, clientIP, limit, window)
		if err != nil {
			logger.Error(ctx, "rate limit check failed",
				zap.String("client_ip", clientIP),
				zap.Error(err),
			)
			// On error, allow the request (fail open)
			c.Next()
			return
		}

		if !allowed {
			logger.Warn(ctx, "rate limit exceeded",
				zap.String("client_ip", clientIP),
				zap.Int64("limit", limit),
				zap.Duration("window", window),
			)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": int(window.Seconds()),
				"limit":       limit,
				"window":      window.String(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByAPIKey는 API Key 기반 rate limiting 미들웨어입니다
func RateLimitByAPIKey(redisExtended *cache.RedisExtended, limits map[string]int64, window time.Duration) gin.HandlerFunc {
	rateLimiter := redisExtended.NewRateLimiter("api:ratelimit:apikey")

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Get API key from header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// If no API key, use IP-based rate limiting
			apiKey = c.ClientIP()
		}

		// Get limit for this API key (default to 100 if not specified)
		limit, ok := limits[apiKey]
		if !ok {
			limit = 100
		}

		// Check rate limit
		allowed, err := rateLimiter.Allow(ctx, apiKey, limit, window)
		if err != nil {
			logger.Error(ctx, "rate limit check failed",
				zap.String("api_key", apiKey),
				zap.Error(err),
			)
			// On error, allow the request (fail open)
			c.Next()
			return
		}

		if !allowed {
			logger.Warn(ctx, "rate limit exceeded",
				zap.String("api_key", apiKey),
				zap.Int64("limit", limit),
				zap.Duration("window", window),
			)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": int(window.Seconds()),
				"limit":       limit,
				"window":      window.String(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByUser는 사용자 ID 기반 rate limiting 미들웨어입니다
func RateLimitByUser(redisExtended *cache.RedisExtended, limit int64, window time.Duration) gin.HandlerFunc {
	rateLimiter := redisExtended.NewRateLimiter("api:ratelimit:user")

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Get user ID from context (assuming authentication middleware sets this)
		userID, exists := c.Get("user_id")
		if !exists {
			// If no user ID, use IP-based rate limiting
			userID = c.ClientIP()
		}

		userIDStr := userID.(string)

		// Check rate limit
		allowed, err := rateLimiter.Allow(ctx, userIDStr, limit, window)
		if err != nil {
			logger.Error(ctx, "rate limit check failed",
				zap.String("user_id", userIDStr),
				zap.Error(err),
			)
			// On error, allow the request (fail open)
			c.Next()
			return
		}

		if !allowed {
			logger.Warn(ctx, "rate limit exceeded",
				zap.String("user_id", userIDStr),
				zap.Int64("limit", limit),
				zap.Duration("window", window),
			)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": int(window.Seconds()),
				"limit":       limit,
				"window":      window.String(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
