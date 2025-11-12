package middleware

import (
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// TracingMiddleware는 OpenTelemetry 분산 추적을 추가합니다
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Span 시작
		ctx, span := tracing.StartSpan(c.Request.Context(), c.Request.Method+" "+c.Request.URL.Path)
		defer span.End()

		// Request 정보를 span attributes로 추가
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.target", c.Request.URL.Path),
			attribute.String("http.host", c.Request.Host),
			attribute.String("http.scheme", c.Request.URL.Scheme),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("http.client_ip", c.ClientIP()),
		)

		// Request ID가 있으면 추가
		if requestID, exists := c.Get(RequestIDKey); exists {
			span.SetAttributes(attribute.String("request.id", requestID.(string)))
		}

		// Context에 trace 정보 추가
		ctx = logger.WithFields(ctx,
			logger.TraceID(tracing.GetTraceID(ctx)),
			logger.SpanID(tracing.GetSpanID(ctx)),
		)
		c.Request = c.Request.WithContext(ctx)

		// 요청 처리
		c.Next()

		// Response 정보 추가
		statusCode := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// 에러가 있으면 span에 기록
		if len(c.Errors) > 0 {
			span.RecordError(c.Errors.Last())
			span.SetStatus(codes.Error, c.Errors.String())
		} else if statusCode >= 400 {
			span.SetStatus(codes.Error, "HTTP "+string(rune(statusCode)))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}
