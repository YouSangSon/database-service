package interceptor

import (
	"context"
	"runtime/debug"

	"github.com/YouSangSon/database-service/internal/pkg/errors"
	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryRecoveryInterceptor는 gRPC unary 요청에서 패닉을 복구합니다
func UnaryRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// 스택 트레이스 캡처
				stack := string(debug.Stack())

				logger.Error(ctx, "panic recovered in gRPC unary handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", stack),
				)

				// gRPC 에러 반환
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor는 gRPC stream 요청에서 패닉을 복구합니다
func StreamRecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// 스택 트레이스 캡처
				stack := string(debug.Stack())

				ctx := ss.Context()
				logger.Error(ctx, "panic recovered in gRPC stream handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", stack),
				)

				// gRPC 에러 반환
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}

// UnaryErrorHandlerInterceptor는 에러를 gRPC 상태 코드로 변환합니다
func UnaryErrorHandlerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)

		if err != nil {
			// AppError를 gRPC 상태로 변환
			var appErr *errors.AppError
			if errors.As(err, &appErr) {
				code := mapErrorCodeToGRPC(appErr.Code)
				return resp, status.Error(code, appErr.Message)
			}
		}

		return resp, err
	}
}

// mapErrorCodeToGRPC는 AppError 코드를 gRPC 코드로 매핑합니다
func mapErrorCodeToGRPC(errCode errors.ErrorCode) codes.Code {
	switch errCode {
	case errors.ErrCodeBadRequest, errors.ErrCodeInvalidInput:
		return codes.InvalidArgument
	case errors.ErrCodeNotFound:
		return codes.NotFound
	case errors.ErrCodeConflict, errors.ErrCodeVersionConflict:
		return codes.AlreadyExists
	case errors.ErrCodeTimeout, errors.ErrCodeDatabaseTimeout:
		return codes.DeadlineExceeded
	case errors.ErrCodeServiceUnavailable, errors.ErrCodeCircuitOpen:
		return codes.Unavailable
	case errors.ErrCodeRateLimitExceeded:
		return codes.ResourceExhausted
	default:
		return codes.Internal
	}
}
