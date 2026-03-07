package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	AuthValidataionTotal metric.Int64Counter
	AuthValidationDuration metric.Float64Histogram

	AuthzDecisionsTotal metric.Int64Counter

	KeycloakAPICallsTotal metric.Int64Counter
	KeycloakAPIDuration metric.Float64Histogram

	TokenCacheHits metric.Int64Counter
	TokenCacheMisses metric.Int64Counter
}

type MeterProvider struct {
	provider *sdkmetric.MeterProvider
}

func NewMeterProvider() (*MeterProvider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter failed: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)

	return &MeterProvider{provider: provider}, nil
}

func NewMetrics(mp *MeterProvider) (*Metrics, error) {
	meter := mp.provider.Meter("identity-access")

	authTotal, err := meter.Int64Counter(
		"auth_validation_total",
		metric.WithDescription("Total number of token validations"),
	)

	if err != nil {
		return nil, fmt.Errorf("create auth_validation_total failed: %w", err)
	}

	authDuration, err := meter.Float64Histogram(
		"auth_validation_duration_seconds",
		metric.WithDescription("Token validation latency in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create auth_validation_duration_seconds: %w", err)
	}

	authzTotal, err := meter.Int64Counter(
		"authz_decisions_total",
		metric.WithDescription("Total authorization decisions"),
	)
	if err != nil {
		return nil, fmt.Errorf("create authz_decsiosions_total failed: %w", err)
	}

	keycloakTotal, err := meter.Int64Counter(
        "keycloak_api_calls_total",
        metric.WithDescription("Total Keycloak Admin API calls"),
    )
    if err != nil {
        return nil, fmt.Errorf("create keycloak_api_calls_total failed: %w", err)
    }

    keycloakDuration, err := meter.Float64Histogram(
        "keycloak_api_duration_seconds",
        metric.WithDescription("Keycloak Admin API call latency"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return nil, fmt.Errorf("create keycloak_api_duration failed: %w", err)
    }

    cacheHits, err := meter.Int64Counter(
        "token_cache_hits_total",
        metric.WithDescription("Service account token cache hits"),
    )
    if err != nil {
        return nil, fmt.Errorf("create token_cache_hits failed: %w", err)
    }

    cacheMisses, err := meter.Int64Counter(
        "token_cache_misses_total",
        metric.WithDescription("Service account token cache misses"),
    )
    if err != nil {
        return nil, fmt.Errorf("create token_cache_misses failed: %w", err)
    }

		return &Metrics{
			AuthValidataionTotal: authTotal,
			AuthValidationDuration: authDuration,
			AuthzDecisionsTotal: authzTotal,
			KeycloakAPICallsTotal: keycloakTotal,
			KeycloakAPIDuration: keycloakDuration,
			TokenCacheHits: cacheHits,
			TokenCacheMisses: cacheMisses,
		}, nil
}

func (mp *MeterProvider) shutdown() error {
	if err := mp.provider.Shutdown(nil); err != nil {
		return fmt.Errorf("shutdown meter provider failed: %w", err)
	}
	return nil
}