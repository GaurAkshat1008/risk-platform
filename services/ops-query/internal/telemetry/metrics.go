package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Metrics holds all prometheus/OTel instruments for ops-query.
type Metrics struct {
	QueryLogsTotal    metric.Int64Counter
	QueryTracesTotal  metric.Int64Counter
	GetSLOStatusTotal metric.Int64Counter
	ListAlertsTotal   metric.Int64Counter
	OpsRPCDuration    metric.Float64Histogram
}

// MeterProvider wraps the OTel SDK meter provider.
type MeterProvider struct {
	provider *sdkmetric.MeterProvider
}

// NewMeterProvider creates a Prometheus-backed OTel meter provider.
func NewMeterProvider() (*MeterProvider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	return &MeterProvider{provider: provider}, nil
}

// NewMetrics registers all instruments against the given MeterProvider.
func NewMetrics(mp *MeterProvider) (*Metrics, error) {
	meter := mp.provider.Meter("ops-query")

	queryLogs, err := meter.Int64Counter("ops_query_logs_total",
		metric.WithDescription("Total QueryLogs RPC calls"))
	if err != nil {
		return nil, fmt.Errorf("ops_query_logs_total: %w", err)
	}

	queryTraces, err := meter.Int64Counter("ops_query_traces_total",
		metric.WithDescription("Total QueryTraces RPC calls"))
	if err != nil {
		return nil, fmt.Errorf("ops_query_traces_total: %w", err)
	}

	getSLO, err := meter.Int64Counter("ops_get_slo_status_total",
		metric.WithDescription("Total GetSLOStatus RPC calls"))
	if err != nil {
		return nil, fmt.Errorf("ops_get_slo_status_total: %w", err)
	}

	listAlerts, err := meter.Int64Counter("ops_list_alerts_total",
		metric.WithDescription("Total ListAlerts RPC calls"))
	if err != nil {
		return nil, fmt.Errorf("ops_list_alerts_total: %w", err)
	}

	rpcDuration, err := meter.Float64Histogram("ops_rpc_duration_seconds",
		metric.WithDescription("gRPC handler duration in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("ops_rpc_duration_seconds: %w", err)
	}

	return &Metrics{
		QueryLogsTotal:    queryLogs,
		QueryTracesTotal:  queryTraces,
		GetSLOStatusTotal: getSLO,
		ListAlertsTotal:   listAlerts,
		OpsRPCDuration:    rpcDuration,
	}, nil
}
