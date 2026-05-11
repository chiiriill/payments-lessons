package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests by method, route, and status.",
	}, []string{"method", "route", "status"})

	httpRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds by method and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

func Metrics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		route := "unknown"
		if r := c.Route(); r != nil {
			route = r.Path
		}

		status := strconv.Itoa(c.Response().StatusCode())
		httpRequestsTotal.WithLabelValues(c.Method(), route, status).Inc()
		httpRequestDurationSeconds.WithLabelValues(c.Method(), route).Observe(time.Since(start).Seconds())

		return err
	}
}
