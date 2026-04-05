package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Metrics exposes all instrumentation counters and histograms for the notification service.
type Metrics struct {
	NotificationsSentTotal      metric.Int64Counter
	NotificationsDeliveredTotal metric.Int64Counter
	NotificationsFailedTotal    metric.Int64Counter
	KafkaConsumeTotal           metric.Int64Counter
	RateLimitHitsTotal          metric.Int64Counter
	NotificationRPCDuration     metric.Float64Histogram
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

// NewMetrics registers all notification instruments.
func NewMetrics(mp *MeterProvider) (*Metrics, error) {
	meter := mp.provider.Meter("notification")

	sent, err := meter.Int64Counter("notification_sent_total",
		metric.WithDescription("Total notifications created (via gRPC or Kafka)"))
	if err != nil {
		return nil, fmt.Errorf("notification_sent_total: %w", err)
	}

	delivered, err := meter.Int64Counter("notification_delivered_total",
		metric.WithDescription("Total notifications successfully delivered"))
	if err != nil {
		return nil, fmt.Errorf("notification_delivered_total: %w", err)
	}

	failed, err := meter.Int64Counter("notification_failed_total",
		metric.WithDescription("Total notifications that exhausted all retry attempts"))
	if err != nil {
		return nil, fmt.Errorf("notification_failed_total: %w", err)
	}

	kafkaConsume, err := meter.Int64Counter("notification_kafka_consume_total",
		metric.WithDescription("Total Kafka messages consumed by the notification service"))
	if err != nil {
		return nil, fmt.Errorf("notification_kafka_consume_total: %w", err)
	}

	rateLimitHits, err := meter.Int64Counter("notification_rate_limit_hits_total",
		metric.WithDescription("Total notifications dropped by the per-tenant rate limiter"))
	if err != nil {
		return nil, fmt.Errorf("notification_rate_limit_hits_total: %w", err)
	}

	rpcDur, err := meter.Float64Histogram("notification_rpc_duration_seconds",
		metric.WithDescription("Duration of notification gRPC calls in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("notification_rpc_duration_seconds: %w", err)
	}

	return &Metrics{
		NotificationsSentTotal:      sent,
		NotificationsDeliveredTotal: delivered,
		NotificationsFailedTotal:    failed,
		KafkaConsumeTotal:           kafkaConsume,
		RateLimitHitsTotal:          rateLimitHits,
		NotificationRPCDuration:     rpcDur,
	}, nil
}
