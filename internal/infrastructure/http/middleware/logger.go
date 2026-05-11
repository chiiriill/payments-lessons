package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set("X-Request-ID", requestID)

		requestLogger := logger.With("request_id", requestID)
		c.Locals("logger", requestLogger)

		err := c.Next()

		requestLogger.Info("http request",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
		return err
	}
}

// LoggerFrom returns the request-scoped logger stored by RequestLogger middleware.
// Falls back to slog.Default() if the middleware was not registered.
func LoggerFrom(c *fiber.Ctx) *slog.Logger {
	if log, ok := c.Locals("logger").(*slog.Logger); ok {
		return log
	}
	return slog.Default()
}

func newRequestID() string {
	var b [8]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
