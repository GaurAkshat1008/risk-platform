package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Metrics exposes all instrumentation counters and histograms for the explanation service.
type Metrics struct {
	ExplanationsGeneratedTotal metric.Int64Counter
	ExplanationCacheHitsTotal  metric.Int64Counter
	DecisionFetchErrorsTotal   metric.Int64Counter
	ExplanationRPCDuration     metric.Float64Histogram
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

// NewMetrics registers all explanation instruments.
func NewMetrics(mp *MeterProvider) (*Metrics, error) {
	meter := mp.provider.Meter("explanation")

	generated, err := meter.Int64Counter("explanation_generated_total",
		metric.WithDescription("Total explanations generated (fresh or cached)"))
	if err != nil {
		return nil, fmt.Errorf("explanation_generated_total: %w", err)
	}

	cacheHits, err := meter.Int64Counter("explanation_cache_hits_total",
		metric.WithDescription("Total explanations served from the DB cache without re-generation"))
	if err != nil {
		return nil, fmt.Errorf("explanation_cache_hits_total: %w", err)
	}

	fetchErrors, err := meter.Int64Counter("explanation_decision_fetch_errors_total",
		metric.WithDescription("Total errors fetching decision or rules for explanation generation"))
	if err != nil {
		return nil, fmt.Errorf("explanation_decision_fetch_errors_total: %w", err)
	}

	rpcDur, err := meter.Float64Histogram("explanation_rpc_duration_seconds",
		metric.WithDescription("Duration of explanation gRPC calls in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("explanation_rpc_duration_seconds: %w", err)
	}

	return &Metrics{
		ExplanationsGeneratedTotal: generated,
		ExplanationCacheHitsTotal:  cacheHits,
		DecisionFetchErrorsTotal:   fetchErrors,
		ExplanationRPCDuration:     rpcDur,
	}, nil
}
