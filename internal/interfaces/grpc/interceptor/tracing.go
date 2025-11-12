package interceptor

import (
	"context"

	"github.com/YouSangSon/database-service/internal/pkg/logger"
	"github.com/YouSangSon/database-service/internal/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryTracingInterceptor는 gRPC unary 요청에 분산 추적을 추가합니다
func UnaryTracingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Span 시작
		ctx, span := tracing.StartSpan(ctx, info.FullMethod)
		defer span.End()

		// Span attributes 추가
		span.SetAttributes(
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.service", extractServiceName(info.FullMethod)),
		)

		// Context에 trace 정보 추가
		ctx = logger.WithFields(ctx,
			logger.TraceID(tracing.GetTraceID(ctx)),
			logger.SpanID(tracing.GetSpanID(ctx)),
		)

		// 요청 처리
		resp, err := handler(ctx, req)

		// 상태 코드 기록
		st := status.Convert(err)
		span.SetAttributes(
			attribute.String("rpc.grpc.status_code", st.Code().String()),
		)

		// 에러가 있으면 span에 기록
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, st.Message())
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}

// StreamTracingInterceptor는 gRPC stream 요청에 분산 추적을 추가합니다
func StreamTracingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// Span 시작
		ctx, span := tracing.StartSpan(ctx, info.FullMethod)
		defer span.End()

		// Span attributes 추가
		span.SetAttributes(
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.service", extractServiceName(info.FullMethod)),
			attribute.Bool("rpc.grpc.is_client_stream", info.IsClientStream),
			attribute.Bool("rpc.grpc.is_server_stream", info.IsServerStream),
		)

		// Context에 trace 정보 추가
		ctx = logger.WithFields(ctx,
			logger.TraceID(tracing.GetTraceID(ctx)),
			logger.SpanID(tracing.GetSpanID(ctx)),
		)

		// Wrapped stream
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// 스트림 처리
		err := handler(srv, wrappedStream)

		// 상태 코드 기록
		st := status.Convert(err)
		span.SetAttributes(
			attribute.String("rpc.grpc.status_code", st.Code().String()),
		)

		// 에러가 있으면 span에 기록
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, st.Message())
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// extractServiceName은 full method에서 서비스 이름을 추출합니다
// 예: "/database.DatabaseService/Create" -> "database.DatabaseService"
func extractServiceName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return ""
	}
	// Remove leading slash
	if fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	// Find last slash
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[:i]
		}
	}
	return fullMethod
}
