package main

import (
	"context"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/errors"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// 로깅 사용 예제입니다
func main() {
	// 1. 로거 초기화
	err := logger.Init(logger.Config{
		Environment: "development",
		Level:       "debug",
		ServiceName: "database-service",
		Version:     "1.0.0",
	})
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	ctx := context.Background()

	// 2. 기본 로깅
	logger.Info(ctx, "application started")
	logger.Debug(ctx, "debug mode enabled")

	// 3. 구조화된 필드 사용
	logger.Info(ctx, "processing document",
		logger.DocumentID("507f1f77bcf86cd799439011"),
		logger.Collection("users"),
		logger.Operation("create"),
	)

	// 4. 요청 로깅
	requestID := "req-123456"
	ctx = logger.WithFields(ctx,
		logger.RequestID(requestID),
		logger.TraceID("4bf92f3577b34da6a3ce929d0e0e4736"),
	)

	logger.LogRequest(ctx, "POST", "/api/v1/documents",
		logger.RemoteAddr("192.168.1.100"),
	)

	// 5. 데이터베이스 작업 로깅
	start := time.Now()
	// ... DB 작업 수행 ...
	duration := time.Since(start)

	logger.LogDBOperation(ctx, "save", "users", duration.Milliseconds(), nil,
		logger.DocumentID("507f1f77bcf86cd799439011"),
		logger.Version(1),
	)

	// 6. 에러 로깅 - 구조화된 에러 사용
	appErr := errors.New(errors.ErrCodeDatabaseQuery, "query timeout")
	appErr = appErr.
		WithMetadata("collection", "users").
		WithMetadata("query", "find").
		WithDetails("connection timeout after 30 seconds")

	logger.Error(ctx, "database operation failed",
		logger.Operation("find"),
		logger.Collection("users"),
		logger.ErrorCode(string(appErr.Code)),
		logger.Metadata(appErr.Metadata),
		zap.Error(appErr),
	)

	// 7. 캐시 작업 로깅
	cacheKey := "document:users:507f1f77bcf86cd799439011"
	logger.LogCacheOperation(ctx, "get", cacheKey, true, nil,
		logger.Size(1024),
	)

	// 8. 경고 로깅
	logger.Warn(ctx, "deprecated API called",
		logger.HTTPPath("/api/v1/old-endpoint"),
		zap.String("alternative", "/api/v2/new-endpoint"),
	)

	// 9. 응답 로깅
	logger.LogResponse(ctx, 200, time.Since(start).Milliseconds(),
		logger.Size(256),
	)

	// 10. 컨텍스트에 필드 추가
	ctx = logger.WithFields(ctx,
		logger.UserID("user-789"),
		logger.Component("document-handler"),
	)

	logger.Info(ctx, "user action completed")

	// 출력 예제:
	// Development 모드에서는 컬러풀한 출력
	// Production 모드에서는 JSON 출력:
	//
	// {
	//   "level": "info",
	//   "timestamp": "2025-01-01T12:34:56.789Z",
	//   "caller": "examples/logging_example.go:45",
	//   "message": "processing document",
	//   "service": "database-service",
	//   "version": "1.0.0",
	//   "environment": "production",
	//   "document_id": "507f1f77bcf86cd799439011",
	//   "collection": "users",
	//   "operation": "create"
	// }
}
