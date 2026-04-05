package telemetry

import (
    "fmt"

    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/metric"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
    IngestTotal       metric.Int64Counter
    IngestDuration    metric.Float64Histogram
    DedupeHitsTotal   metric.Int64Counter
    KafkaPublishTotal metric.Int64Counter
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
    meter := mp.provider.Meter("ingestion")

    ingestTotal, err := meter.Int64Counter("ingest_total",
        metric.WithDescription("Total payment events processed"))
    if err != nil {
        return nil, fmt.Errorf("ingest_total: %w", err)
    }

    ingestDuration, err := meter.Float64Histogram("ingest_duration_seconds",
        metric.WithDescription("Payment ingestion latency"),
        metric.WithUnit("s"))
    if err != nil {
        return nil, fmt.Errorf("ingest_duration_seconds: %w", err)
    }

    dedupeHits, err := meter.Int64Counter("dedupe_hits_total",
        metric.WithDescription("Duplicate events detected"))
    if err != nil {
        return nil, fmt.Errorf("dedupe_hits_total: %w", err)
    }

    kafkaTotal, err := meter.Int64Counter("kafka_publish_total",
        metric.WithDescription("Total Kafka messages published"))
    if err != nil {
        return nil, fmt.Errorf("kafka_publish_total: %w", err)
    }

    return &Metrics{
        IngestTotal:       ingestTotal,
        IngestDuration:    ingestDuration,
        DedupeHitsTotal:   dedupeHits,
        KafkaPublishTotal: kafkaTotal,
    }, nil
}