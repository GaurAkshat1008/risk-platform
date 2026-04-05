package telemetry

import (
    "context"
    "fmt"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
    "go.opentelemetry.io/otel/trace"
)

type TracerProvider struct {
    provider *sdktrace.TracerProvider
}

type TracerConfig struct {
    ServiceName       string
    ServiceVersion    string
    Environment       string
    CollectorEndpoint string
}

func NewTracerProvider(ctx context.Context, cfg TracerConfig) (*TracerProvider, error) {
    res, err := resource.Merge(
        resource.Default(),
        resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(cfg.ServiceName),
            semconv.ServiceVersion(cfg.ServiceVersion),
            semconv.DeploymentEnvironmentName(cfg.Environment),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("build otel resource: %w", err)
    }

    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(cfg.CollectorEndpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("create otlp exporter: %w", err)
    }

    provider := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )
    otel.SetTracerProvider(provider)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))
    return &TracerProvider{provider: provider}, nil
}

func (tp *TracerProvider) Shutdown(ctx context.Context) error {
    shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    return tp.provider.Shutdown(shutdownCtx)
}

func Tracer(name string) trace.Tracer {
    return otel.Tracer(name)
}