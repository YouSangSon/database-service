package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode는 에러 코드 타입입니다
type ErrorCode string

const (
	// 일반 에러
	ErrCodeInternal   ErrorCode = "INTERNAL_ERROR"
	ErrCodeBadRequest ErrorCode = "BAD_REQUEST"
	ErrCodeNotFound   ErrorCode = "NOT_FOUND"
	ErrCodeConflict   ErrorCode = "CONFLICT"
	ErrCodeTimeout    ErrorCode = "TIMEOUT"

	// 도메인 에러
	ErrCodeInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrCodeInvalidCollection ErrorCode = "INVALID_COLLECTION"
	ErrCodeInvalidDocument  ErrorCode = "INVALID_DOCUMENT"
	ErrCodeVersionConflict  ErrorCode = "VERSION_CONFLICT"

	// 데이터베이스 에러
	ErrCodeDatabaseConnection ErrorCode = "DATABASE_CONNECTION_ERROR"
	ErrCodeDatabaseQuery      ErrorCode = "DATABASE_QUERY_ERROR"
	ErrCodeDatabaseTimeout    ErrorCode = "DATABASE_TIMEOUT"

	// 캐시 에러
	ErrCodeCacheConnection ErrorCode = "CACHE_CONNECTION_ERROR"
	ErrCodeCacheOperation  ErrorCode = "CACHE_OPERATION_ERROR"

	// 서비스 에러
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeCircuitOpen        ErrorCode = "CIRCUIT_BREAKER_OPEN"
	ErrCodeRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
)

// AppError는 애플리케이션 에러입니다
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Err        error                  `json:"-"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Error는 error 인터페이스를 구현합니다
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap은 원본 에러를 반환합니다
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithMetadata는 메타데이터를 추가합니다
func (e *AppError) WithMetadata(key string, value interface{}) *AppError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithDetails는 상세 정보를 추가합니다
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// New는 새로운 AppError를 생성합니다
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
		Metadata:   make(map[string]interface{}),
	}
}

// Wrap은 기존 에러를 AppError로 래핑합니다
func Wrap(err error, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}

	// 이미 AppError인 경우
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
		Err:        err,
		Metadata:   make(map[string]interface{}),
	}
}

// Wrapf는 포맷팅된 메시지로 에러를 래핑합니다
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// Is는 에러가 특정 코드인지 확인합니다
func Is(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// GetCode는 에러 코드를 반환합니다
func GetCode(err error) ErrorCode {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ErrCodeInternal
}

// GetHTTPStatus는 에러의 HTTP 상태 코드를 반환합니다
func GetHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

// getHTTPStatus는 에러 코드에 대응하는 HTTP 상태 코드를 반환합니다
func getHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeBadRequest, ErrCodeInvalidInput, ErrCodeInvalidCollection, ErrCodeInvalidDocument:
		return http.StatusBadRequest
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeConflict, ErrCodeVersionConflict:
		return http.StatusConflict
	case ErrCodeTimeout, ErrCodeDatabaseTimeout:
		return http.StatusRequestTimeout
	case ErrCodeServiceUnavailable, ErrCodeCircuitOpen:
		return http.StatusServiceUnavailable
	case ErrCodeRateLimitExceeded:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// 미리 정의된 에러들
var (
	ErrInvalidInput         = New(ErrCodeInvalidInput, "invalid input")
	ErrInvalidCollection    = New(ErrCodeInvalidCollection, "invalid collection name")
	ErrInvalidDocument      = New(ErrCodeInvalidDocument, "invalid document")
	ErrDocumentNotFound     = New(ErrCodeNotFound, "document not found")
	ErrVersionConflict      = New(ErrCodeVersionConflict, "version conflict - document was modified by another request")
	ErrDatabaseConnection   = New(ErrCodeDatabaseConnection, "database connection error")
	ErrDatabaseQuery        = New(ErrCodeDatabaseQuery, "database query error")
	ErrCacheConnection      = New(ErrCodeCacheConnection, "cache connection error")
	ErrServiceUnavailable   = New(ErrCodeServiceUnavailable, "service unavailable")
	ErrCircuitBreakerOpen   = New(ErrCodeCircuitOpen, "circuit breaker is open - service temporarily unavailable")
	ErrRateLimitExceeded    = New(ErrCodeRateLimitExceeded, "rate limit exceeded")
)
