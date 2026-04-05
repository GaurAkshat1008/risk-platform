package telemetry

import (
	"net/http"
	"time"
)

// RequestTimer wraps an http.Handler and records request duration using the provided Metrics.
func RequestTimer(metrics *Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		metrics.RequestDuration.Record(r.Context(), time.Since(start).Seconds())
	})
}
