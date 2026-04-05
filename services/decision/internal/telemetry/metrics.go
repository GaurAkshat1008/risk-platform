package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	DecisionsTotal    metric.Int64Counter
	OverridesTotal    metric.Int64Counter
	KafkaConsumeTotal metric.Int64Counter
	KafkaPublishTotal metric.Int64Counter
	DecisionLatency   metric.Float64Histogram
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
	meter := mp.provider.Meter("decision")

	decisionsTotal, err := meter.Int64Counter("decisions_total",
		metric.WithDescription("Total decisions recorded"))
	if err != nil {
		return nil, fmt.Errorf("decisions_total: %w", err)
	}

	overridesTotal, err := meter.Int64Counter("decision_overrides_total",
		metric.WithDescription("Total analyst overrides applied"))
	if err != nil {
		return nil, fmt.Errorf("decision_overrides_total: %w", err)
	}

	kafkaConsume, err := meter.Int64Counter("kafka_consume_total",
		metric.WithDescription("Total risk.evaluated events consumed from Kafka"))
	if err != nil {
		return nil, fmt.Errorf("kafka_consume_total: %w", err)
	}

	kafkaPublish, err := meter.Int64Counter("kafka_publish_total",
		metric.WithDescription("Total decision.made events published to Kafka"))
	if err != nil {
		return nil, fmt.Errorf("kafka_publish_total: %w", err)
	}

	decisionLatency, err := meter.Float64Histogram("decision_latency_seconds",
		metric.WithDescription("Time from risk.evaluated consume to decision.made publish"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("decision_latency_seconds: %w", err)
	}

	return &Metrics{
		DecisionsTotal:    decisionsTotal,
		OverridesTotal:    overridesTotal,
		KafkaConsumeTotal: kafkaConsume,
		KafkaPublishTotal: kafkaPublish,
		DecisionLatency:   decisionLatency,
	}, nil
}
