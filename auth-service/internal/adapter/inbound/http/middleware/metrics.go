package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by status code and method.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response latency for HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpInFlightRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_in_flight_requests",
			Help: "Current number of in-flight HTTP requests.",
		},
	)
)

// PrometheusMetrics returns a Gin middleware that records Prometheus metrics.
func PrometheusMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Avoid tracking /metrics, /healthz, /readyz to prevent metric noise
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if path == "/metrics" || path == "/healthz" || path == "/readyz" {
			c.Next()
			return
		}

		httpInFlightRequests.Inc()
		start := time.Now()

		c.Next()

		httpInFlightRequests.Dec()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
