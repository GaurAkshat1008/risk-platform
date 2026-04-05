package telemetry

import (
    "fmt"

    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/metric"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
    RuleEvaluationsTotal metric.Int64Counter
    RulesMatchedTotal    metric.Int64Counter
    EvaluationDuration   metric.Float64Histogram
    KafkaPublishTotal    metric.Int64Counter
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
    meter := mp.provider.Meter("rules-engine")

    evalTotal, err := meter.Int64Counter("rule_evaluations_total",
        metric.WithDescription("Total rule EvaluateRules calls"))
    if err != nil {
        return nil, fmt.Errorf("rule_evaluations_total: %w", err)
    }

    matchedTotal, err := meter.Int64Counter("rules_matched_total",
        metric.WithDescription("Total individual rules that matched"))
    if err != nil {
        return nil, fmt.Errorf("rules_matched_total: %w", err)
    }

    evalDuration, err := meter.Float64Histogram("rule_evaluation_duration_seconds",
        metric.WithDescription("Rule evaluation latency"),
        metric.WithUnit("s"))
    if err != nil {
        return nil, fmt.Errorf("rule_evaluation_duration_seconds: %w", err)
    }

    kafkaTotal, err := meter.Int64Counter("kafka_publish_total",
        metric.WithDescription("Total Kafka messages published"))
    if err != nil {
        return nil, fmt.Errorf("kafka_publish_total: %w", err)
    }

    return &Metrics{
        RuleEvaluationsTotal: evalTotal,
        RulesMatchedTotal:    matchedTotal,
        EvaluationDuration:   evalDuration,
        KafkaPublishTotal:    kafkaTotal,
    }, nil
}