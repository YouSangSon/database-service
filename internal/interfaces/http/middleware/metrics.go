package middleware

import (
	"strconv"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"github.com/gin-gonic/gin"
)

// MetricsMiddleware는 Prometheus 메트릭을 수집합니다
func MetricsMiddleware() gin.HandlerFunc {
	m := metrics.GetMetrics()

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Request size
		requestSize := computeApproximateRequestSize(c.Request)

		// 요청 처리
		c.Next()

		// Response 정보 수집
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		responseSize := c.Writer.Size()

		// 메트릭 기록
		m.RecordHTTPRequest(
			c.Request.Method,
			path,
			strconv.Itoa(statusCode),
			duration,
			requestSize,
			responseSize,
		)
	}
}

// computeApproximateRequestSize는 요청 크기를 대략적으로 계산합니다
func computeApproximateRequestSize(r interface{}) int {
	// 간단한 구현: Content-Length 사용
	// 실제로는 헤더 크기 등도 포함해야 하지만 근사값으로 충분
	s := 0
	if req, ok := r.(interface{ ContentLength() int64 }); ok {
		s = int(req.ContentLength())
	}
	return s
}
