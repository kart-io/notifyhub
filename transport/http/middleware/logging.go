package middleware

import (
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/logger"
)

// LoggingMiddleware provides request logging for HTTP transport
type LoggingMiddleware struct {
	logger logger.Interface
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger logger.Interface) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Middleware returns the HTTP middleware function
func (l *LoggingMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call next handler
		next(wrapper, r)

		// Log the request
		duration := time.Since(start)
		l.logger.Info(r.Context(), "HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapper.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write ensures status code is set
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
