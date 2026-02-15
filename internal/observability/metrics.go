package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fluxlens_http_requests_total",
			Help: "Total HTTP requests handled by FluxLens gateways.",
		},
		[]string{"method", "path", "code"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fluxlens_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	auditChainLength = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "fluxlens_audit_chain_length",
			Help: "Number of records in the active audit chain.",
		},
	)
	auditChainValid = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "fluxlens_audit_chain_valid",
			Help: "1 if audit chain verifies, else 0.",
		},
	)
)

// Handler returns the Prometheus scrape endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// Instrument wraps an HTTP handler with request metrics.
func Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rw, r)
		path := normalizePath(r.URL.Path)
		httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(rw.code)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	})
}

// SetAuditMetrics updates gauges from the current chain state.
func SetAuditMetrics(length int, valid bool) {
	auditChainLength.Set(float64(length))
	if valid {
		auditChainValid.Set(1)
	} else {
		auditChainValid.Set(0)
	}
}

func normalizePath(p string) string {
	switch p {
	case "/api/v1/health", "/api/v1/events", "/api/v1/digest", "/api/v1/audit",
		"/api/v1/alerts", "/api/v1/operator/suggest", "/api/v1/operator/resolve",
		"/api/v1/operator/export", "/metrics", "/api/openapi.yaml":
		return p
	default:
		return "other"
	}
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
