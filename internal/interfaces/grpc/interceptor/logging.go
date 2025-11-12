package interceptor

import (
	"context"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	RequestIDMetadataKey = "x-request-id"
)

// UnaryLoggingInterceptor는 gRPC unary 요청을 로깅합니다
func UnaryLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Request ID 추출 또는 생성
		requestID := extractOrGenerateRequestID(ctx)

		// Context에 request ID 추가
		ctx = logger.WithFields(ctx,
			logger.RequestID(requestID),
		)

		// 요청 로깅
		logger.Info(ctx, "incoming gRPC request",
			zap.String("method", info.FullMethod),
			zap.Any("request", req),
		)

		// 요청 처리
		resp, err := handler(ctx, req)

		// 응답 로깅
		duration := time.Since(start)
		statusCode := status.Code(err)

		if err != nil {
			logger.Error(ctx, "gRPC request failed",
				zap.String("method", info.FullMethod),
				logger.Duration(duration),
				logger.DurationMs(duration),
				zap.String("status", statusCode.String()),
				zap.Error(err),
			)
		} else {
			logger.Info(ctx, "gRPC request completed",
				zap.String("method", info.FullMethod),
				logger.Duration(duration),
				logger.DurationMs(duration),
				zap.String("status", statusCode.String()),
			)
		}

		return resp, err
	}
}

// StreamLoggingInterceptor는 gRPC stream 요청을 로깅합니다
func StreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()

		// Request ID 추출 또는 생성
		requestID := extractOrGenerateRequestID(ctx)

		// Context에 request ID 추가
		ctx = logger.WithFields(ctx,
			logger.RequestID(requestID),
		)

		// Wrapped stream
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// 요청 로깅
		logger.Info(ctx, "incoming gRPC stream",
			zap.String("method", info.FullMethod),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
		)

		// 스트림 처리
		err := handler(srv, wrappedStream)

		// 응답 로깅
		duration := time.Since(start)
		statusCode := status.Code(err)

		if err != nil {
			logger.Error(ctx, "gRPC stream failed",
				zap.String("method", info.FullMethod),
				logger.Duration(duration),
				zap.String("status", statusCode.String()),
				zap.Error(err),
			)
		} else {
			logger.Info(ctx, "gRPC stream completed",
				zap.String("method", info.FullMethod),
				logger.Duration(duration),
				zap.String("status", statusCode.String()),
			)
		}

		return err
	}
}

// extractOrGenerateRequestID는 metadata에서 request ID를 추출하거나 생성합니다
func extractOrGenerateRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if ids := md.Get(RequestIDMetadataKey); len(ids) > 0 {
			return ids[0]
		}
	}
	return uuid.New().String()
}

// wrappedServerStream은 context를 커스터마이즈한 ServerStream입니다
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
