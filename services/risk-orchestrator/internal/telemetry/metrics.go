package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	RiskEvaluationsTotal   metric.Int64Counter
	FailOpenTotal          metric.Int64Counter
	RulesEngineErrorsTotal metric.Int64Counter
	OrchestratorLatency    metric.Float64Histogram
	KafkaPublishTotal      metric.Int64Counter
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
	meter := mp.provider.Meter("risk-orchestrator")

	evalTotal, err := meter.Int64Counter("risk_evaluations_total",
		metric.WithDescription("Total risk evaluations processed"))
	if err != nil {
		return nil, fmt.Errorf("risk_evaluations_total: %w", err)
	}

	failOpen, err := meter.Int64Counter("fail_open_total",
		metric.WithDescription("Total evaluations that applied fail-open due to Rules Engine unavailability"))
	if err != nil {
		return nil, fmt.Errorf("fail_open_total: %w", err)
	}

	reErrors, err := meter.Int64Counter("rules_engine_errors_total",
		metric.WithDescription("Total errors calling the Rules Engine"))
	if err != nil {
		return nil, fmt.Errorf("rules_engine_errors_total: %w", err)
	}

	latency, err := meter.Float64Histogram("orchestrator_latency_seconds",
		metric.WithDescription("End-to-end orchestration latency"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("orchestrator_latency_seconds: %w", err)
	}

	kafkaTotal, err := meter.Int64Counter("kafka_publish_total",
		metric.WithDescription("Total Kafka messages published"))
	if err != nil {
		return nil, fmt.Errorf("kafka_publish_total: %w", err)
	}

	return &Metrics{
		RiskEvaluationsTotal:   evalTotal,
		FailOpenTotal:          failOpen,
		RulesEngineErrorsTotal: reErrors,
		OrchestratorLatency:    latency,
		KafkaPublishTotal:      kafkaTotal,
	}, nil
}
