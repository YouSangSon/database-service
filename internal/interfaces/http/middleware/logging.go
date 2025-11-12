package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

// RequestIDMiddleware는 각 요청에 고유 ID를 부여합니다
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		// 컨텍스트에 request ID 추가
		ctx := logger.WithFields(c.Request.Context(),
			logger.RequestID(requestID),
		)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// LoggingMiddleware는 HTTP 요청/응답을 로깅합니다
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Request body 로깅 (선택적, 프로덕션에서는 주의 필요)
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 요청 로깅
		ctx := c.Request.Context()
		logger.Info(ctx, "incoming request",
			logger.HTTPMethod(c.Request.Method),
			logger.HTTPPath(path),
			logger.RemoteAddr(c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("request_size", len(requestBody)),
		)

		// Custom response writer to capture status code and size
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 요청 처리
		c.Next()

		// 응답 로깅
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		// 에러가 있으면 error 레벨로 로깅
		if len(c.Errors) > 0 {
			logger.Error(ctx, "request completed with errors",
				logger.HTTPMethod(c.Request.Method),
				logger.HTTPPath(path),
				logger.HTTPStatus(statusCode),
				logger.Duration(duration),
				logger.DurationMs(duration),
				zap.Int("response_size", blw.body.Len()),
				zap.Errors("errors", c.Errors.Errors()),
			)
		} else {
			logLevel := logger.Info
			if statusCode >= 500 {
				logLevel = logger.Error
			} else if statusCode >= 400 {
				logLevel = logger.Warn
			}

			logLevel(ctx, "request completed",
				logger.HTTPMethod(c.Request.Method),
				logger.HTTPPath(path),
				logger.HTTPStatus(statusCode),
				logger.Duration(duration),
				logger.DurationMs(duration),
				zap.Int("response_size", blw.body.Len()),
			)
		}
	}
}

// bodyLogWriter는 응답 body를 캡처하는 커스텀 writer입니다
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
