package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PrometheusClient queries the Prometheus HTTP API.
type PrometheusClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPrometheusClient creates a Prometheus API client.
func NewPrometheusClient(addr string) *PrometheusClient {
	return &PrometheusClient{
		baseURL:    addr,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SLOMetrics holds aggregated SLO data retrieved from Prometheus.
type SLOMetrics struct {
	ErrorRate    float64
	P50LatencyMs float64
	P95LatencyMs float64
	P99LatencyMs float64
	Availability float64
}

// GetSLOMetrics fetches error rate, latency percentiles, and availability for
// the given service over the given window (e.g. "5m", "1h", "24h").
func (c *PrometheusClient) GetSLOMetrics(ctx context.Context, service, window string) (*SLOMetrics, error) {
	now := time.Now()
	start := now.Add(-parseDuration(window))
	step := "60s"

	errorRate, err := c.scalarQuery(ctx,
		fmt.Sprintf(
			`sum(rate(grpc_server_handled_total{grpc_code!="OK",job=~".*%s.*"}[%s])) / sum(rate(grpc_server_handled_total{job=~".*%s.*"}[%s]))`,
			service, window, service, window,
		),
		start, now, step,
	)
	if err != nil {
		return nil, fmt.Errorf("error rate query: %w", err)
	}

	p50, err := c.scalarQuery(ctx,
		fmt.Sprintf(
			`histogram_quantile(0.50, sum(rate(grpc_server_handling_seconds_bucket{job=~".*%s.*"}[%s])) by (le)) * 1000`,
			service, window,
		),
		start, now, step,
	)
	if err != nil {
		return nil, fmt.Errorf("p50 query: %w", err)
	}

	p95, err := c.scalarQuery(ctx,
		fmt.Sprintf(
			`histogram_quantile(0.95, sum(rate(grpc_server_handling_seconds_bucket{job=~".*%s.*"}[%s])) by (le)) * 1000`,
			service, window,
		),
		start, now, step,
	)
	if err != nil {
		return nil, fmt.Errorf("p95 query: %w", err)
	}

	p99, err := c.scalarQuery(ctx,
		fmt.Sprintf(
			`histogram_quantile(0.99, sum(rate(grpc_server_handling_seconds_bucket{job=~".*%s.*"}[%s])) by (le)) * 1000`,
			service, window,
		),
		start, now, step,
	)
	if err != nil {
		return nil, fmt.Errorf("p99 query: %w", err)
	}

	// availability = 1 - error_rate, clamped to [0,1]
	availability := 1.0 - errorRate
	if availability < 0 {
		availability = 0
	}

	return &SLOMetrics{
		ErrorRate:    errorRate,
		P50LatencyMs: p50,
		P95LatencyMs: p95,
		P99LatencyMs: p99,
		Availability: availability,
	}, nil
}

// prometheusQueryRangeResponse is the minimal response shape from /api/v1/query_range.
type prometheusQueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Values [][]interface{} `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// PrometheusAlert is an alert returned from /api/v1/alerts.
type PrometheusAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"` // "firing" | "pending" | "inactive"
	ActiveAt    time.Time         `json:"activeAt"`
}

// prometheusAlertsResponse is the response shape of /api/v1/alerts.
type prometheusAlertsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Alerts []PrometheusAlert `json:"alerts"`
	} `json:"data"`
}

// GetAlerts retrieves all alerts from Prometheus.
func (c *PrometheusClient) GetAlerts(ctx context.Context) ([]PrometheusAlert, error) {
	endpoint := c.baseURL + "/api/v1/alerts"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build alerts request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get alerts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read alerts body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus alerts HTTP %d: %s", resp.StatusCode, body)
	}

	var result prometheusAlertsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode alerts: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus alerts status: %s", result.Status)
	}

	return result.Data.Alerts, nil
}

// scalarQuery runs a query_range and returns the last available scalar value.
func (c *PrometheusClient) scalarQuery(ctx context.Context, query string, start, end time.Time, step string) (float64, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start.Unix(), 10))
	params.Set("end", strconv.FormatInt(end.Unix(), 10))
	params.Set("step", step)

	endpoint := c.baseURL + "/api/v1/query_range?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("prometheus HTTP %d: %s", resp.StatusCode, body)
	}

	var result prometheusQueryRangeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	if result.Status != "success" {
		return 0, fmt.Errorf("prometheus status: %s", result.Status)
	}

	// Return the most recent non-NaN value.
	for i := len(result.Data.Result) - 1; i >= 0; i-- {
		r := result.Data.Result[i]
		for j := len(r.Values) - 1; j >= 0; j-- {
			if valStr, ok := r.Values[j][1].(string); ok {
				if valStr == "NaN" {
					continue
				}
				f, err := strconv.ParseFloat(valStr, 64)
				if err == nil {
					return f, nil
				}
			}
		}
	}
	return 0, nil
}

// parseDuration converts a Prometheus-style window string to a Go time.Duration.
func parseDuration(window string) time.Duration {
	if len(window) == 0 {
		return time.Hour
	}
	d, err := time.ParseDuration(window)
	if err == nil {
		return d
	}
	// Handle "24h", "7d" patterns more robustly.
	if len(window) >= 2 {
		unit := window[len(window)-1]
		value, err := strconv.Atoi(window[:len(window)-1])
		if err == nil {
			switch unit {
			case 'm':
				return time.Duration(value) * time.Minute
			case 'h':
				return time.Duration(value) * time.Hour
			case 'd':
				return time.Duration(value) * 24 * time.Hour
			case 'w':
				return time.Duration(value) * 7 * 24 * time.Hour
			}
		}
	}
	return time.Hour
}
