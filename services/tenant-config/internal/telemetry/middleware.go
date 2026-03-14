package telemetry

import (
    "context"
    "log/slog"
    "time"

    "go.opentelemetry.io/otel/attribute"
    otelcodes "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
    "google.golang.org/grpc"
    "google.golang.org/grpc/status"
)

func UnaryServerInterceptor(metrics *Metrics) grpc.UnaryServerInterceptor {
    tracer := Tracer("grpc.server")

    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        start := time.Now()

        ctx, span := tracer.Start(ctx, info.FullMethod,
            trace.WithAttributes(
                attribute.String("rpc.method", info.FullMethod),
                attribute.String("rpc.system", "grpc"),
            ),
        )
        defer span.End()

        slog.Default().With(
            "trace_id", span.SpanContext().TraceID().String(),
            "span_id", span.SpanContext().SpanID().String(),
            "grpc_method", info.FullMethod,
        ).Info("gRPC request started")

        resp, err := handler(ctx, req)

        duration := time.Since(start).Seconds()
        grpcStatus, _ := status.FromError(err)
        statusLabel := attribute.String("status", grpcStatus.Code().String())
        methodLabel := attribute.String("method", info.FullMethod)

        if err != nil {
            span.SetStatus(otelcodes.Error, err.Error())
            span.SetAttributes(attribute.String("grpc.status_code", grpcStatus.Code().String()))
        } else {
            span.SetStatus(otelcodes.Ok, "")
        }

        metrics.TenantOperationsTotal.Add(ctx, 1, metric.WithAttributes(statusLabel, methodLabel))
        metrics.TenantOperationDuration.Record(ctx, duration, metric.WithAttributes(statusLabel, methodLabel))

        slog.Default().Info("gRPC request completed",
            "duration_ms", time.Since(start).Milliseconds(),
            "grpc_status", grpcStatus.Code().String(),
            "method", info.FullMethod,
        )
        return resp, err
    }
}