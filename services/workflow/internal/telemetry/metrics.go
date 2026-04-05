package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Metrics exposes all instrumentation counters and histograms for the workflow service.
type Metrics struct {
	TemplatesCreatedTotal metric.Int64Counter
	TemplatesUpdatedTotal metric.Int64Counter
	TransitionEvalTotal   metric.Int64Counter
	CacheHitsTotal        metric.Int64Counter
	CacheMissesTotal      metric.Int64Counter
	WorkflowRPCDuration   metric.Float64Histogram
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

// NewMetrics registers all workflow instruments.
func NewMetrics(mp *MeterProvider) (*Metrics, error) {
	meter := mp.provider.Meter("workflow")

	created, err := meter.Int64Counter("workflow_templates_created_total",
		metric.WithDescription("Total workflow templates created"))
	if err != nil {
		return nil, fmt.Errorf("workflow_templates_created_total: %w", err)
	}

	updated, err := meter.Int64Counter("workflow_templates_updated_total",
		metric.WithDescription("Total workflow templates updated"))
	if err != nil {
		return nil, fmt.Errorf("workflow_templates_updated_total: %w", err)
	}

	evalTotal, err := meter.Int64Counter("workflow_transition_evals_total",
		metric.WithDescription("Total EvaluateTransition calls"))
	if err != nil {
		return nil, fmt.Errorf("workflow_transition_evals_total: %w", err)
	}

	cacheHits, err := meter.Int64Counter("workflow_cache_hits_total",
		metric.WithDescription("Total cache hits for workflow templates"))
	if err != nil {
		return nil, fmt.Errorf("workflow_cache_hits_total: %w", err)
	}

	cacheMisses, err := meter.Int64Counter("workflow_cache_misses_total",
		metric.WithDescription("Total cache misses for workflow templates"))
	if err != nil {
		return nil, fmt.Errorf("workflow_cache_misses_total: %w", err)
	}

	rpcDuration, err := meter.Float64Histogram("workflow_rpc_duration_seconds",
		metric.WithDescription("gRPC handler duration for workflow service"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("workflow_rpc_duration_seconds: %w", err)
	}

	return &Metrics{
		TemplatesCreatedTotal: created,
		TemplatesUpdatedTotal: updated,
		TransitionEvalTotal:   evalTotal,
		CacheHitsTotal:        cacheHits,
		CacheMissesTotal:      cacheMisses,
		WorkflowRPCDuration:   rpcDuration,
	}, nil
}
