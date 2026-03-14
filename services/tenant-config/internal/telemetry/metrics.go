package telemetry

import (
    "fmt"

    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/metric"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
    TenantOperationsTotal    metric.Int64Counter
    TenantOperationDuration  metric.Float64Histogram
    CacheHitsTotal           metric.Int64Counter
    CacheMissesTotal         metric.Int64Counter
    KafkaPublishTotal        metric.Int64Counter
}

type MeterProvider struct {
    provider *sdkmetric.MeterProvider
}

func NewMeterProvider() (*MeterProvider, error) {
    exporter, err := prometheus.New()
    if err != nil {
        return nil, fmt.Errorf("create prometheus exporter: %w", err)
    }
    provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
    return &MeterProvider{provider: provider}, nil
}

func NewMetrics(mp *MeterProvider) (*Metrics, error) {
    meter := mp.provider.Meter("tenant-config")

    opsTotal, err := meter.Int64Counter("tenant_operations_total",
        metric.WithDescription("Total tenant config operations"))
    if err != nil {
        return nil, fmt.Errorf("tenant_operations_total: %w", err)
    }

    opsDuration, err := meter.Float64Histogram("tenant_operation_duration_seconds",
        metric.WithDescription("Tenant config operation latency"),
        metric.WithUnit("s"))
    if err != nil {
        return nil, fmt.Errorf("tenant_operation_duration_seconds: %w", err)
    }

    cacheHits, err := meter.Int64Counter("cache_hits_total",
        metric.WithDescription("Tenant cache hits"))
    if err != nil {
        return nil, fmt.Errorf("cache_hits_total: %w", err)
    }

    cacheMisses, err := meter.Int64Counter("cache_misses_total",
        metric.WithDescription("Tenant cache misses"))
    if err != nil {
        return nil, fmt.Errorf("cache_misses_total: %w", err)
    }

    kafkaTotal, err := meter.Int64Counter("kafka_publish_total",
        metric.WithDescription("Total Kafka messages published"))
    if err != nil {
        return nil, fmt.Errorf("kafka_publish_total: %w", err)
    }

    return &Metrics{
        TenantOperationsTotal:   opsTotal,
        TenantOperationDuration: opsDuration,
        CacheHitsTotal:          cacheHits,
        CacheMissesTotal:        cacheMisses,
        KafkaPublishTotal:       kafkaTotal,
    }, nil
}