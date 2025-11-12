package interceptor

import (
	"context"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryMetricsInterceptor는 gRPC unary 요청의 메트릭을 수집합니다
func UnaryMetricsInterceptor() grpc.UnaryServerInterceptor {
	m := metrics.GetMetrics()

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// 요청 처리
		resp, err := handler(ctx, req)

		// 메트릭 기록
		duration := time.Since(start)
		statusCode := status.Code(err).String()

		m.RecordGRPCRequest(info.FullMethod, statusCode, duration)

		return resp, err
	}
}

// StreamMetricsInterceptor는 gRPC stream 요청의 메트릭을 수집합니다
func StreamMetricsInterceptor() grpc.StreamServerInterceptor {
	m := metrics.GetMetrics()

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// 스트림 처리
		err := handler(srv, ss)

		// 메트릭 기록
		duration := time.Since(start)
		statusCode := status.Code(err).String()

		m.RecordGRPCRequest(info.FullMethod, statusCode, duration)

		return err
	}
}
