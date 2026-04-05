package telemetry

import (
	"fmt"

	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	QueryTotal      metric.Int64Counter
	MutationTotal   metric.Int64Counter
	CacheHits       metric.Int64Counter
	CacheMisses     metric.Int64Counter
	RequestDuration metric.Float64Histogram
}

func NewMeterProvider() (*sdkmetric.MeterProvider, error) {
	exporter, err := prometheusexporter.New()
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	return mp, nil
}

func NewMetrics(mp *sdkmetric.MeterProvider) (*Metrics, error) {
	meter := mp.Meter("graphql-bff")

	queryTotal, err := meter.Int64Counter("bff_graphql_queries_total",
		metric.WithDescription("Total number of GraphQL query operations"))
	if err != nil {
		return nil, err
	}

	mutationTotal, err := meter.Int64Counter("bff_graphql_mutations_total",
		metric.WithDescription("Total number of GraphQL mutation operations"))
	if err != nil {
		return nil, err
	}

	cacheHits, err := meter.Int64Counter("bff_cache_hits_total",
		metric.WithDescription("Total Redis cache hits"))
	if err != nil {
		return nil, err
	}

	cacheMisses, err := meter.Int64Counter("bff_cache_misses_total",
		metric.WithDescription("Total Redis cache misses"))
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram("bff_request_duration_seconds",
		metric.WithDescription("GraphQL request duration in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	return &Metrics{
		QueryTotal:      queryTotal,
		MutationTotal:   mutationTotal,
		CacheHits:       cacheHits,
		CacheMisses:     cacheMisses,
		RequestDuration: requestDuration,
	}, nil
}
