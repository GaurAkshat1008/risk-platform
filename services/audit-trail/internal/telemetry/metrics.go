package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"

	sdk "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	AuditEventsAppended    metric.Int64Counter
	ChainVerificationsTotal metric.Int64Counter
	KafkaConsumeTotal      metric.Int64Counter
	AuditRPCDuration       metric.Float64Histogram
}

func NewMetrics(ctx context.Context, reader sdk.Reader) (*Metrics, error) {
	provider := sdk.NewMeterProvider(sdk.WithReader(reader))
	meter := provider.Meter("audit-trail")

	appended, err := meter.Int64Counter("audit_events_appended_total",
		metric.WithDescription("Total audit events appended to the store"))
	if err != nil {
		return nil, fmt.Errorf("audit_events_appended_total: %w", err)
	}

	verifications, err := meter.Int64Counter("audit_chain_verifications_total",
		metric.WithDescription("Total chain integrity verifications performed"))
	if err != nil {
		return nil, fmt.Errorf("audit_chain_verifications_total: %w", err)
	}

	kafkaConsume, err := meter.Int64Counter("audit_kafka_consume_total",
		metric.WithDescription("Total Kafka messages consumed by audit trail"))
	if err != nil {
		return nil, fmt.Errorf("audit_kafka_consume_total: %w", err)
	}

	rpcDuration, err := meter.Float64Histogram("audit_rpc_duration_seconds",
		metric.WithDescription("gRPC handler duration in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("audit_rpc_duration_seconds: %w", err)
	}

	return &Metrics{
		AuditEventsAppended:    appended,
		ChainVerificationsTotal: verifications,
		KafkaConsumeTotal:      kafkaConsume,
		AuditRPCDuration:       rpcDuration,
	}, nil
}
