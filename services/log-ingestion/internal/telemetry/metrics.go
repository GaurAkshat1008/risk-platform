package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	LogsIngestedTotal   metric.Int64Counter
	KafkaConsumeTotal   metric.Int64Counter
	QueryRequestsTotal  metric.Int64Counter
	LogRPCDuration      metric.Float64Histogram
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
	meter := mp.provider.Meter("log-ingestion")

	ingested, err := meter.Int64Counter("log_ingested_total",
		metric.WithDescription("Total log entries ingested (gRPC + Kafka)"))
	if err != nil {
		return nil, fmt.Errorf("log_ingested_total: %w", err)
	}

	kafkaConsume, err := meter.Int64Counter("log_kafka_consume_total",
		metric.WithDescription("Total log entries consumed from Kafka ops.logs"))
	if err != nil {
		return nil, fmt.Errorf("log_kafka_consume_total: %w", err)
	}

	queryReqs, err := meter.Int64Counter("log_query_requests_total",
		metric.WithDescription("Total QueryLogs RPC calls"))
	if err != nil {
		return nil, fmt.Errorf("log_query_requests_total: %w", err)
	}

	rpcDuration, err := meter.Float64Histogram("log_rpc_duration_seconds",
		metric.WithDescription("gRPC handler duration in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("log_rpc_duration_seconds: %w", err)
	}

	return &Metrics{
		LogsIngestedTotal:  ingested,
		KafkaConsumeTotal:  kafkaConsume,
		QueryRequestsTotal: queryReqs,
		LogRPCDuration:     rpcDuration,
	}, nil
}
