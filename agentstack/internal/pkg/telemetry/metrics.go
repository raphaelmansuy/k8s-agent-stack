package telemetry

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	activeRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_active_requests",
		Help: "Number of active HTTP requests",
	})
)

// MetricsMiddleware returns a middleware that records HTTP metrics.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		activeRequests.Inc()
		defer activeRequests.Dec()

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		status := string(rune(ww.Status()))
		path := r.URL.Path // In production, you might want to use chi.RouteContext(r.Context()).RoutePattern()

		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
	})
}

// MetricsHandler returns the Prometheus metrics handler.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
