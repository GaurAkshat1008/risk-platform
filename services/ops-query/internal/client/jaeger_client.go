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

// JaegerClient queries the Jaeger HTTP REST API.
type JaegerClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewJaegerClient creates a new Jaeger API client.
func NewJaegerClient(addr string) *JaegerClient {
	return &JaegerClient{
		baseURL:    addr,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// JaegerSpan represents a span returned from the Jaeger API.
type JaegerSpan struct {
	TraceID       string    `json:"traceID"`
	SpanID        string    `json:"spanID"`
	ParentSpanID  string    // derived from references
	OperationName string    `json:"operationName"`
	ServiceName   string    // derived from process
	DurationMs    int64     // derived from duration (microseconds → ms)
	Status        string    // derived from tags
	StartTime     time.Time // derived from startTime (unix microseconds)
}

// jaegerSpanRaw is the minimal JSON shape of a Jaeger trace API response span.
type jaegerSpanRaw struct {
	TraceID       string `json:"traceID"`
	SpanID        string `json:"spanID"`
	OperationName string `json:"operationName"`
	StartTime     int64  `json:"startTime"` // unix microseconds
	Duration      int64  `json:"duration"`  // microseconds
	ProcessID     string `json:"processID"`
	References    []struct {
		RefType string `json:"refType"`
		SpanID  string `json:"spanID"`
	} `json:"references"`
	Tags []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Type  string `json:"type"`
	} `json:"tags"`
}

type jaegerTraceResponse struct {
	Data []struct {
		TraceID   string            `json:"traceID"`
		Spans     []jaegerSpanRaw   `json:"spans"`
		Processes map[string]struct {
			ServiceName string `json:"serviceName"`
		} `json:"processes"`
	} `json:"data"`
	Errors []struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	} `json:"errors"`
}

// GetTrace retrieves all spans for a specific trace ID.
func (c *JaegerClient) GetTrace(ctx context.Context, traceID string) ([]JaegerSpan, error) {
	endpoint := fmt.Sprintf("%s/api/traces/%s", c.baseURL, url.PathEscape(traceID))
	return c.fetchTrace(ctx, endpoint)
}

// SearchTraces returns spans matching the given criteria.
func (c *JaegerClient) SearchTraces(ctx context.Context, service string, from, to time.Time, limit int) ([]JaegerSpan, error) {
	params := url.Values{}
	params.Set("service", service)
	params.Set("start", strconv.FormatInt(from.UnixMicro(), 10))
	params.Set("end", strconv.FormatInt(to.UnixMicro(), 10))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	endpoint := c.baseURL + "/api/traces?" + params.Encode()
	return c.fetchTrace(ctx, endpoint)
}

func (c *JaegerClient) fetchTrace(ctx context.Context, endpoint string) ([]JaegerSpan, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build jaeger request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jaeger http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read jaeger body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jaeger HTTP %d: %s", resp.StatusCode, body)
	}

	var result jaegerTraceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode jaeger response: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("jaeger error %d: %s", result.Errors[0].Code, result.Errors[0].Msg)
	}

	var spans []JaegerSpan
	for _, trace := range result.Data {
		for _, raw := range trace.Spans {
			span := JaegerSpan{
				TraceID:       raw.TraceID,
				SpanID:        raw.SpanID,
				OperationName: raw.OperationName,
				DurationMs:    raw.Duration / 1000,
				StartTime:     time.UnixMicro(raw.StartTime).UTC(),
			}

			// Resolve service name from process map.
			if proc, ok := trace.Processes[raw.ProcessID]; ok {
				span.ServiceName = proc.ServiceName
			}

			// Resolve parent span from references.
			for _, ref := range raw.References {
				if ref.RefType == "CHILD_OF" {
					span.ParentSpanID = ref.SpanID
					break
				}
			}

			// Extract status from tags.
			span.Status = "OK"
			for _, tag := range raw.Tags {
				if tag.Key == "error" && tag.Value == "true" {
					span.Status = "ERROR"
					break
				}
				if tag.Key == "otel.status_code" {
					span.Status = tag.Value
				}
			}

			spans = append(spans, span)
		}
	}

	return spans, nil
}
