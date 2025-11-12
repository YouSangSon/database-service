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

// Init은 글로벌 로거를 초기화합니다
func Init(environment string) error {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
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
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Info(msg, fields...)
}

// Error는 error 레벨 로그를 출력합니다
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Error(msg, fields...)
}

// Warn은 warn 레벨 로그를 출력합니다
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Warn(msg, fields...)
}

// Debug는 debug 레벨 로그를 출력합니다
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Debug(msg, fields...)
}

// Fatal은 fatal 레벨 로그를 출력하고 프로그램을 종료합니다
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).Fatal(msg, fields...)
	os.Exit(1)
}

// Sync는 로거를 flush합니다
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}
