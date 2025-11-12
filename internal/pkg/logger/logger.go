package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const (
	loggerKey contextKey = "logger"
)

var globalLogger *zap.Logger

// Config는 로거 설정입니다
type Config struct {
	Environment string
	Level       string
	ServiceName string
	Version     string
}

// Init은 글로벌 로거를 초기화합니다
func Init(cfg Config) error {
	var config zap.Config

	if cfg.Environment == "production" {
		config = zap.NewProductionConfig()
		// Production 환경용 설정
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.MessageKey = "message"
		config.EncoderConfig.LevelKey = "level"
		config.EncoderConfig.CallerKey = "caller"
		config.EncoderConfig.StacktraceKey = "stacktrace"
		config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		config.EncoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		// Development 환경용 설정 (컬러풀한 출력)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	// 로그 레벨 설정
	if cfg.Level != "" {
		level, err := zapcore.ParseLevel(cfg.Level)
		if err == nil {
			config.Level = zap.NewAtomicLevelAt(level)
		}
	}

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	// 서비스 정보를 기본 필드로 추가
	if cfg.ServiceName != "" {
		logger = logger.With(zap.String("service", cfg.ServiceName))
	}
	if cfg.Version != "" {
		logger = logger.With(zap.String("version", cfg.Version))
	}
	if cfg.Environment != "" {
		logger = logger.With(zap.String("environment", cfg.Environment))
	}

	// Kubernetes 정보 추가 (있는 경우)
	if podName := os.Getenv("POD_NAME"); podName != "" {
		logger = logger.With(zap.String("pod_name", podName))
	}
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		logger = logger.With(zap.String("node_name", nodeName))
	}
	if namespace := os.Getenv("NAMESPACE"); namespace != "" {
		logger = logger.With(zap.String("namespace", namespace))
	}

	globalLogger = logger
	return nil
}

// GetLogger는 컨텍스트에서 로거를 가져오거나 글로벌 로거를 반환합니다
func GetLogger(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
			return logger
		}
	}

	if globalLogger == nil {
		// 기본 로거 초기화
		globalLogger, _ = zap.NewProduction()
	}

	return globalLogger
}

// WithLogger는 컨텍스트에 로거를 추가합니다
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// WithFields는 컨텍스트의 로거에 필드를 추가합니다
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	logger := GetLogger(ctx).With(fields...)
	return WithLogger(ctx, logger)
}

// Info는 info 레벨 로그를 출력합니다
// 일반적인 정보성 메시지 (요청 처리, 작업 완료 등)
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Info(msg, fields...)
}

// Error는 error 레벨 로그를 출력합니다
// 에러 발생 시 사용 (복구 가능한 에러)
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Error(msg, fields...)
}

// Warn은 warn 레벨 로그를 출력합니다
// 경고 메시지 (잠재적 문제, deprecated 사용 등)
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Warn(msg, fields...)
}

// Debug는 debug 레벨 로그를 출력합니다
// 디버깅 정보 (상세한 실행 흐름, 변수 값 등)
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Debug(msg, fields...)
}

// Fatal은 fatal 레벨 로그를 출력하고 프로그램을 종료합니다
// 복구 불가능한 심각한 에러 (서비스 시작 실패 등)
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Fatal(msg, fields...)
	os.Exit(1)
}

// Panic은 panic 레벨 로그를 출력하고 패닉을 발생시킵니다
func Panic(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Panic(msg, fields...)
}

// Sync는 로거를 flush합니다
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}

// 편의 함수들

// LogError는 에러를 구조화된 형태로 로깅합니다
func LogError(ctx context.Context, err error, msg string, fields ...zap.Field) {
	allFields := append(fields, zap.Error(err))
	GetLogger(ctx).Error(msg, allFields...)
}

// LogRequest는 HTTP/gRPC 요청을 로깅합니다
func LogRequest(ctx context.Context, method, path string, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("method", method),
		zap.String("path", path),
	)
	GetLogger(ctx).Info("request received", allFields...)
}

// LogResponse는 HTTP/gRPC 응답을 로깅합니다
func LogResponse(ctx context.Context, status int, duration int64, fields ...zap.Field) {
	allFields := append(fields,
		zap.Int("status", status),
		zap.Int64("duration_ms", duration),
	)
	GetLogger(ctx).Info("request completed", allFields...)
}

// LogDBOperation은 데이터베이스 작업을 로깅합니다
func LogDBOperation(ctx context.Context, operation, collection string, duration int64, err error, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("operation", operation),
		zap.String("collection", collection),
		zap.Int64("duration_ms", duration),
	)

	if err != nil {
		allFields = append(allFields, zap.Error(err))
		GetLogger(ctx).Error("database operation failed", allFields...)
	} else {
		GetLogger(ctx).Debug("database operation completed", allFields...)
	}
}

// LogCacheOperation은 캐시 작업을 로깅합니다
func LogCacheOperation(ctx context.Context, operation, key string, hit bool, err error, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("operation", operation),
		zap.String("cache_key", key),
		zap.Bool("cache_hit", hit),
	)

	if err != nil {
		allFields = append(allFields, zap.Error(err))
		GetLogger(ctx).Warn("cache operation failed", allFields...)
	} else {
		GetLogger(ctx).Debug("cache operation completed", allFields...)
	}
}
