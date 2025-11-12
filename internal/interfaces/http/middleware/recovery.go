package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/YouSangSon/database-service/internal/pkg/errors"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RecoveryMiddleware는 패닉을 복구하고 500 에러를 반환합니다
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 스택 트레이스 캡처
				stack := string(debug.Stack())

				ctx := c.Request.Context()
				logger.Error(ctx, "panic recovered",
					logger.HTTPMethod(c.Request.Method),
					logger.HTTPPath(c.Request.URL.Path),
					logger.RemoteAddr(c.ClientIP()),
					zap.Any("panic", err),
					zap.String("stack", stack),
				)

				// 에러 응답
				appErr := errors.New(errors.ErrCodeInternal, "internal server error")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    appErr.Code,
						"message": appErr.Message,
					},
					"request_id": c.GetString(RequestIDKey),
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

// ErrorHandlerMiddleware는 에러를 표준화된 형식으로 반환합니다
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 에러가 있는지 확인
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			ctx := c.Request.Context()

			// AppError로 변환
			var appErr *errors.AppError
			if !errors.As(err, &appErr) {
				// 알 수 없는 에러인 경우 Internal Error로 처리
				appErr = errors.Wrap(err, errors.ErrCodeInternal, "internal server error")
			}

			// 로깅
			logger.Error(ctx, "request error",
				logger.HTTPMethod(c.Request.Method),
				logger.HTTPPath(c.Request.URL.Path),
				logger.ErrorCode(string(appErr.Code)),
				logger.ErrorMessage(appErr.Message),
				logger.Metadata(appErr.Metadata),
				zap.Error(appErr),
			)

			// 응답
			status := appErr.HTTPStatus
			response := gin.H{
				"error": gin.H{
					"code":    appErr.Code,
					"message": appErr.Message,
				},
				"request_id": c.GetString(RequestIDKey),
			}

			// 세부 정보 추가 (개발 환경)
			if appErr.Details != "" {
				response["error"].(gin.H)["details"] = appErr.Details
			}

			// 메타데이터 추가 (있는 경우)
			if len(appErr.Metadata) > 0 {
				response["error"].(gin.H)["metadata"] = appErr.Metadata
			}

			c.JSON(status, response)
		}
	}
}

// CORSMiddleware는 CORS 헤더를 설정합니다
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// TimeoutMiddleware는 요청에 타임아웃을 설정합니다
func TimeoutMiddleware(timeout int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// context timeout은 이미 설정되어 있다고 가정
		// 추가적인 타임아웃 로직이 필요하면 여기에 구현
		c.Next()
	}
}

// RateLimitMiddleware는 속도 제한을 적용합니다
func RateLimitMiddleware() gin.HandlerFunc {
	// TODO: Redis 기반 rate limiting 구현
	// 현재는 placeholder
	return func(c *gin.Context) {
		c.Next()
	}
}

// HealthCheckMiddleware는 health check 엔드포인트를 처리합니다
func HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/health/live" {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
