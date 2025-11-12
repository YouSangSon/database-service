# 로깅 가이드

## 개요

이 프로젝트는 구조화된 로깅을 위해 Uber의 [zap](https://github.com/uber-go/zap)을 사용합니다.
모든 로그는 JSON 형식으로 출력되며, 일관된 필드명을 사용하여 가독성과 검색성을 높입니다.

## 로그 레벨

### Debug (디버그)
개발 중 상세한 정보를 로깅합니다. Production에서는 일반적으로 비활성화됩니다.

```go
logger.Debug(ctx, "processing document",
    logger.DocumentID("abc123"),
    logger.Collection("users"),
)
```

### Info (정보)
일반적인 정보성 메시지입니다. 정상적인 흐름을 추적합니다.

```go
logger.Info(ctx, "document created successfully",
    logger.DocumentID(doc.ID()),
    logger.Collection("users"),
    logger.Duration(time.Since(start)),
)
```

### Warn (경고)
잠재적인 문제나 deprecated 기능 사용 시 로깅합니다.

```go
logger.Warn(ctx, "cache operation failed, continuing without cache",
    logger.CacheKey(key),
    zap.Error(err),
)
```

### Error (에러)
복구 가능한 에러를 로깅합니다.

```go
logger.Error(ctx, "failed to save document",
    logger.DocumentID(doc.ID()),
    logger.Collection("users"),
    zap.Error(err),
)
```

### Fatal (치명적)
복구 불가능한 에러로, 로깅 후 프로그램을 종료합니다.

```go
logger.Fatal(ctx, "failed to connect to database",
    logger.DatabaseHost(host),
    zap.Error(err),
)
```

## 구조화된 필드 사용

### 요청 관련

```go
// HTTP 요청
logger.Info(ctx, "handling create document request",
    logger.RequestID(requestID),
    logger.HTTPMethod("POST"),
    logger.HTTPPath("/api/v1/documents"),
    logger.RemoteAddr(r.RemoteAddr),
)

// 응답
logger.Info(ctx, "request completed",
    logger.RequestID(requestID),
    logger.HTTPStatus(200),
    logger.HTTPStatusText("OK"),
    logger.DurationMs(time.Since(start)),
)
```

### 데이터베이스 작업

```go
start := time.Now()
err := repo.Save(ctx, doc)

logger.LogDBOperation(ctx, "save", "users", time.Since(start).Milliseconds(), err,
    logger.DocumentID(doc.ID()),
    logger.Version(doc.Version()),
)

// 또는 더 상세하게:
if err != nil {
    logger.Error(ctx, "database save operation failed",
        logger.Operation("save"),
        logger.Collection("users"),
        logger.DocumentID(doc.ID()),
        logger.Duration(time.Since(start)),
        logger.ErrorCode(errors.GetCode(err).String()),
        zap.Error(err),
    )
}
```

### 캐시 작업

```go
val, err := cache.Get(ctx, key)
if err != nil {
    logger.LogCacheOperation(ctx, "get", key, false, err)
} else {
    logger.LogCacheOperation(ctx, "get", key, true, nil,
        logger.Size(int64(len(val))),
    )
}
```

### 분산 추적 통합

```go
import "github.com/YouSangSon/database-service/internal/pkg/tracing"

ctx, span := tracing.StartSpan(ctx, "CreateDocument")
defer span.End()

// 트레이스 ID를 로그에 자동 포함
logger.Info(ctx, "processing request",
    logger.TraceID(tracing.GetTraceID(ctx)),
    logger.SpanID(tracing.GetSpanID(ctx)),
)
```

### 에러 처리

구조화된 에러 사용:

```go
import "github.com/YouSangSon/database-service/internal/pkg/errors"

// 에러 생성
err := errors.New(errors.ErrCodeInvalidInput, "collection name is required")

// 에러 래핑
err = errors.Wrap(dbErr, errors.ErrCodeDatabaseQuery, "failed to query database")

// 메타데이터 추가
err = err.
    WithMetadata("collection", "users").
    WithMetadata("query_type", "find").
    WithDetails("connection timeout after 30s")

// 로깅
logger.Error(ctx, "operation failed",
    logger.ErrorCode(string(err.Code)),
    logger.ErrorMessage(err.Message),
    logger.Metadata(err.Metadata),
    zap.Error(err),
)
```

## 로그 출력 예제

### Production 환경 (JSON)

```json
{
  "level": "info",
  "timestamp": "2025-01-01T12:34:56.789Z",
  "caller": "usecase/document_usecase.go:45",
  "message": "document created successfully",
  "service": "database-service",
  "version": "1.0.0",
  "environment": "production",
  "pod_name": "database-service-api-7d5f8b9c4-xz9k2",
  "node_name": "node-1",
  "namespace": "database-service",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "request_id": "req-123456",
  "document_id": "507f1f77bcf86cd799439011",
  "collection": "users",
  "operation": "create",
  "duration_ms": 15.3,
  "version": 1
}
```

### Development 환경 (컬러풀)

```
2025-01-01T12:34:56.789+0900    INFO    usecase/document_usecase.go:45    document created successfully
    service: database-service
    document_id: 507f1f77bcf86cd799439011
    collection: users
    duration_ms: 15.3
```

## 모범 사례

### 1. 일관된 필드명 사용
```go
// ✅ Good - 헬퍼 함수 사용
logger.Info(ctx, "document updated",
    logger.DocumentID(id),
    logger.Collection(collection),
)

// ❌ Bad - 직접 필드명 작성
logger.Info(ctx, "document updated",
    zap.String("docId", id),  // 일관성 없음
    zap.String("coll", collection),
)
```

### 2. 컨텍스트에 필드 추가
```go
// 요청 시작 시 컨텍스트에 공통 필드 추가
ctx = logger.WithFields(ctx,
    logger.RequestID(requestID),
    logger.UserID(userID),
)

// 이후 모든 로그에 자동 포함
logger.Info(ctx, "processing request")
logger.Debug(ctx, "validating input")
```

### 3. 에러 로깅
```go
// ✅ Good - 구조화된 에러 정보
if err != nil {
    logger.Error(ctx, "database operation failed",
        logger.Operation("save"),
        logger.Collection("users"),
        logger.ErrorCode(string(errors.GetCode(err))),
        zap.Error(err),
    )
    return err
}

// ❌ Bad - 단순 에러 문자열
if err != nil {
    logger.Error(ctx, fmt.Sprintf("error: %v", err))
}
```

### 4. 성능 고려
```go
// ✅ Good - 조건부 로깅
if logger.GetLogger(ctx).Core().Enabled(zapcore.DebugLevel) {
    // 비용이 큰 연산은 debug 레벨이 활성화된 경우만 실행
    details := expensiveOperation()
    logger.Debug(ctx, "detailed info", zap.Any("details", details))
}

// ❌ Bad - 항상 실행
logger.Debug(ctx, "detailed info", zap.Any("details", expensiveOperation()))
```

### 5. 민감 정보 제외
```go
// ✅ Good - 민감 정보 마스킹
logger.Info(ctx, "user authenticated",
    logger.UserID(userID),
    zap.String("email_domain", strings.Split(email, "@")[1]),
)

// ❌ Bad - 민감 정보 노출
logger.Info(ctx, "user authenticated",
    zap.String("email", email),
    zap.String("password", password),  // 절대 금지!
)
```

## 초기화

애플리케이션 시작 시:

```go
import "github.com/YouSangSon/database-service/internal/pkg/logger"

func main() {
    // 로거 초기화
    err := logger.Init(logger.Config{
        Environment: os.Getenv("ENVIRONMENT"),
        Level:       os.Getenv("LOG_LEVEL"),
        ServiceName: "database-service",
        Version:     "1.0.0",
    })
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer logger.Sync()

    // ...
}
```

## Kubernetes 환경 변수

자동으로 로그에 포함되는 Kubernetes 정보:

```yaml
env:
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: NODE_NAME
  valueFrom:
    fieldRef:
      fieldPath: spec.nodeName
- name: NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: LOG_LEVEL
  value: "info"
- name: ENVIRONMENT
  value: "production"
```
