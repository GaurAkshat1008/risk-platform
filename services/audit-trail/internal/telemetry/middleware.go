package telemetry

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor records RPC duration and sets span status.
func UnaryServerInterceptor(metrics *Metrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		slog.Info("audit rpc start", "method", info.FullMethod)

		resp, err := handler(ctx, req)

		duration := time.Since(start).Seconds()
		metrics.AuditRPCDuration.Record(ctx, duration)

		span := SpanFromContext(ctx)
		if err != nil {
			st, _ := status.FromError(err)
			span.SetStatus(codes.Error, st.Message())
			slog.Error("audit rpc error", "method", info.FullMethod, "error", err)
		} else {
			span.SetStatus(codes.Ok, "")
			slog.Info("audit rpc complete", "method", info.FullMethod, "duration_ms", time.Since(start).Milliseconds())
		}

		return resp, err
	}
}
