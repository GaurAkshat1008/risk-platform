package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Metrics exposes all instrumentation counters and histograms for case-management.
type Metrics struct {
	CasesCreatedTotal   metric.Int64Counter
	CasesResolvedTotal  metric.Int64Counter
	CasesEscalatedTotal metric.Int64Counter
	CasesAssignedTotal  metric.Int64Counter
	KafkaConsumeTotal   metric.Int64Counter
	KafkaPublishTotal   metric.Int64Counter
	CaseRPCDuration     metric.Float64Histogram
}

// MeterProvider wraps the OTel SDK MeterProvider backed by Prometheus.
type MeterProvider struct {
	provider *sdkmetric.MeterProvider
}

// NewMeterProvider creates a Prometheus-backed OTel MeterProvider.
func NewMeterProvider() (*MeterProvider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	return &MeterProvider{provider: provider}, nil
}

// NewMetrics registers all case-management instruments.
func NewMetrics(mp *MeterProvider) (*Metrics, error) {
	meter := mp.provider.Meter("case-management")

	casesCreated, err := meter.Int64Counter("cases_created_total",
		metric.WithDescription("Total cases created from decision.made events"))
	if err != nil {
		return nil, fmt.Errorf("cases_created_total: %w", err)
	}

	casesResolved, err := meter.Int64Counter("cases_resolved_total",
		metric.WithDescription("Total cases resolved by analysts"))
	if err != nil {
		return nil, fmt.Errorf("cases_resolved_total: %w", err)
	}

	casesEscalated, err := meter.Int64Counter("cases_escalated_total",
		metric.WithDescription("Total cases escalated (SLA breach or manual)"))
	if err != nil {
		return nil, fmt.Errorf("cases_escalated_total: %w", err)
	}

	casesAssigned, err := meter.Int64Counter("cases_assigned_total",
		metric.WithDescription("Total case assignments made"))
	if err != nil {
		return nil, fmt.Errorf("cases_assigned_total: %w", err)
	}

	kafkaConsume, err := meter.Int64Counter("kafka_consume_total",
		metric.WithDescription("Total Kafka messages consumed"))
	if err != nil {
		return nil, fmt.Errorf("kafka_consume_total: %w", err)
	}

	kafkaPublish, err := meter.Int64Counter("kafka_publish_total",
		metric.WithDescription("Total Kafka messages published"))
	if err != nil {
		return nil, fmt.Errorf("kafka_publish_total: %w", err)
	}

	rpcDuration, err := meter.Float64Histogram("case_rpc_duration_seconds",
		metric.WithDescription("gRPC handler duration for case-management"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("case_rpc_duration_seconds: %w", err)
	}

	return &Metrics{
		CasesCreatedTotal:   casesCreated,
		CasesResolvedTotal:  casesResolved,
		CasesEscalatedTotal: casesEscalated,
		CasesAssignedTotal:  casesAssigned,
		KafkaConsumeTotal:   kafkaConsume,
		KafkaPublishTotal:   kafkaPublish,
		CaseRPCDuration:     rpcDuration,
	}, nil
}
