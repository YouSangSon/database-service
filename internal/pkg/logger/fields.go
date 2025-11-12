package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 일관된 로그 필드를 위한 헬퍼 함수들

// RequestID는 요청 ID 필드를 반환합니다
func RequestID(id string) zap.Field {
	return zap.String("request_id", id)
}

// TraceID는 trace ID 필드를 반환합니다
func TraceID(id string) zap.Field {
	return zap.String("trace_id", id)
}

// SpanID는 span ID 필드를 반환합니다
func SpanID(id string) zap.Field {
	return zap.String("span_id", id)
}

// UserID는 사용자 ID 필드를 반환합니다
func UserID(id string) zap.Field {
	return zap.String("user_id", id)
}

// Collection은 컬렉션명 필드를 반환합니다
func Collection(name string) zap.Field {
	return zap.String("collection", name)
}

// DocumentID는 문서 ID 필드를 반환합니다
func DocumentID(id string) zap.Field {
	return zap.String("document_id", id)
}

// Operation은 작업명 필드를 반환합니다
func Operation(op string) zap.Field {
	return zap.String("operation", op)
}

// Duration은 작업 시간 필드를 반환합니다
func Duration(d time.Duration) zap.Field {
	return zap.Duration("duration", d)
}

// DurationMs는 작업 시간을 밀리초로 반환합니다
func DurationMs(d time.Duration) zap.Field {
	return zap.Float64("duration_ms", float64(d.Milliseconds()))
}

// HTTPMethod는 HTTP 메서드 필드를 반환합니다
func HTTPMethod(method string) zap.Field {
	return zap.String("http_method", method)
}

// HTTPPath는 HTTP 경로 필드를 반환합니다
func HTTPPath(path string) zap.Field {
	return zap.String("http_path", path)
}

// HTTPStatus는 HTTP 상태 코드 필드를 반환합니다
func HTTPStatus(status int) zap.Field {
	return zap.Int("http_status", status)
}

// HTTPStatusText는 HTTP 상태 텍스트 필드를 반환합니다
func HTTPStatusText(text string) zap.Field {
	return zap.String("http_status_text", text)
}

// RemoteAddr는 원격 주소 필드를 반환합니다
func RemoteAddr(addr string) zap.Field {
	return zap.String("remote_addr", addr)
}

// ErrorCode는 에러 코드 필드를 반환합니다
func ErrorCode(code string) zap.Field {
	return zap.String("error_code", code)
}

// ErrorMessage는 에러 메시지 필드를 반환합니다
func ErrorMessage(msg string) zap.Field {
	return zap.String("error_message", msg)
}

// ErrorStack는 에러 스택 필드를 반환합니다
func ErrorStack(stack string) zap.Field {
	return zap.String("error_stack", stack)
}

// Component는 컴포넌트명 필드를 반환합니다
func Component(name string) zap.Field {
	return zap.String("component", name)
}

// Service는 서비스명 필드를 반환합니다
func Service(name string) zap.Field {
	return zap.String("service", name)
}

// Version은 버전 필드를 반환합니다
func Version(v int) zap.Field {
	return zap.Int("version", v)
}

// Count는 카운트 필드를 반환합니다
func Count(n int) zap.Field {
	return zap.Int("count", n)
}

// Size는 크기 필드를 반환합니다
func Size(n int64) zap.Field {
	return zap.Int64("size", n)
}

// CacheKey는 캐시 키 필드를 반환합니다
func CacheKey(key string) zap.Field {
	return zap.String("cache_key", key)
}

// CacheHit는 캐시 히트 여부 필드를 반환합니다
func CacheHit(hit bool) zap.Field {
	return zap.Bool("cache_hit", hit)
}

// DatabaseHost는 데이터베이스 호스트 필드를 반환합니다
func DatabaseHost(host string) zap.Field {
	return zap.String("db_host", host)
}

// DatabaseName은 데이터베이스명 필드를 반환합니다
func DatabaseName(name string) zap.Field {
	return zap.String("db_name", name)
}

// QueryTime은 쿼리 실행 시간 필드를 반환합니다
func QueryTime(d time.Duration) zap.Field {
	return zap.Duration("query_time", d)
}

// Retry는 재시도 횟수 필드를 반환합니다
func Retry(attempt int) zap.Field {
	return zap.Int("retry_attempt", attempt)
}

// MaxRetries는 최대 재시도 횟수 필드를 반환합니다
func MaxRetries(max int) zap.Field {
	return zap.Int("max_retries", max)
}

// CircuitState는 circuit breaker 상태 필드를 반환합니다
func CircuitState(state string) zap.Field {
	return zap.String("circuit_state", state)
}

// Namespace는 Kubernetes 네임스페이스 필드를 반환합니다
func Namespace(ns string) zap.Field {
	return zap.String("namespace", ns)
}

// PodName은 Pod 이름 필드를 반환합니다
func PodName(name string) zap.Field {
	return zap.String("pod_name", name)
}

// NodeName은 노드 이름 필드를 반환합니다
func NodeName(name string) zap.Field {
	return zap.String("node_name", name)
}

// Any는 임의의 값 필드를 반환합니다
func Any(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

// Struct는 구조체를 JSON으로 직렬화한 필드를 반환합니다
func Struct(key string, value interface{}) zap.Field {
	return zap.Object(key, zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		return enc.AddReflected(key, value)
	}))
}

// Metadata는 메타데이터 맵 필드를 반환합니다
func Metadata(data map[string]interface{}) zap.Field {
	return zap.Any("metadata", data)
}

// Tags는 태그 배열 필드를 반환합니다
func Tags(tags []string) zap.Field {
	return zap.Strings("tags", tags)
}

// Environment는 환경 필드를 반환합니다
func Environment(env string) zap.Field {
	return zap.String("environment", env)
}
