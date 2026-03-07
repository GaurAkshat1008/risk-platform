package telemetry

import (
    "context"
    "log/slog"
    "time"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
		"go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
    "google.golang.org/grpc"
    "google.golang.org/grpc/status"
)

func UnaryServerInterceptor(metrics *Metrics) grpc.UnaryServerInterceptor {
    tracer := Tracer("grpc.server")

    return func(
        ctx context.Context,
        req any,
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (any, error) {
        start := time.Now()

        ctx, span := tracer.Start(ctx, info.FullMethod,
            trace.WithAttributes(
                attribute.String("rpc.method", info.FullMethod),
                attribute.String("rpc.system", "grpc"),
            ),
        )
        defer span.End()

        traceID := span.SpanContext().TraceID().String()
        spanID := span.SpanContext().SpanID().String()

        logger := slog.Default().With(
            "trace_id", traceID,
            "span_id", spanID,
            "grpc_method", info.FullMethod,
        )
        logger.Info("gRPC request started")

        resp, err := handler(ctx, req)

        duration := time.Since(start).Seconds()
        grpcStatus, _ := status.FromError(err)

        if err != nil {
            span.SetStatus(codes.Error, err.Error())
            span.SetAttributes(
                attribute.String("grpc.status_code", grpcStatus.Code().String()),
            )
        } else {
            span.SetStatus(codes.Ok, "")
        }

        statusLabel := attribute.String("status", grpcStatus.Code().String())
        methodLabel := attribute.String("method", info.FullMethod)

        metrics.AuthValidationDuration.Record(ctx, duration,
            metric.WithAttributes(statusLabel, methodLabel),
        )

        logger.Info("gRPC request completed",
            "duration_ms", time.Since(start).Milliseconds(),
            "grpc_status", grpcStatus.Code().String(),
        )

        return resp, err
    }
}