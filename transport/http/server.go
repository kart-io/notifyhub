package http

import (
	"context"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/api"
)

// HTTPServer implements HTTP transport layer as specified in the proposal
type HTTPServer struct {
	hub        *api.NotifyHub
	server     *http.Server
	port       string
	middleware []Middleware
}

// Config holds HTTP server configuration
type Config struct {
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	EnableCORS     bool
}

// Engine represents different HTTP frameworks
type Engine interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Handle(method, path string, handler HandlerFunc)
	Use(middleware Middleware)
}

// HandlerFunc represents an HTTP handler function
type HandlerFunc func(Context) error

// Context represents an HTTP request/response context
type Context interface {
	Request() *http.Request
	Response() http.ResponseWriter
	Param(key string) string
	Query(key string) string
	JSON(code int, obj interface{}) error
	Bind(obj interface{}) error
}

// Middleware represents HTTP middleware
type Middleware func(HandlerFunc) HandlerFunc

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(hub *api.NotifyHub, config *Config) *HTTPServer {
	if config.Port == "" {
		config.Port = ":8080"
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}

	return &HTTPServer{
		hub:        hub,
		port:       config.Port,
		middleware: make([]Middleware, 0),
	}
}

// AddMiddleware adds middleware to the server
func (s *HTTPServer) AddMiddleware(middleware Middleware) {
	s.middleware = append(s.middleware, middleware)
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	// Register routes
	s.registerRoutes(mux)

	s.server = &http.Server{
		Addr:           s.port,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// registerRoutes registers HTTP routes
func (s *HTTPServer) registerRoutes(mux *http.ServeMux) {
	// Create handlers
	sendHandler := &sendHandler{hub: s.hub}
	batchHandler := &batchHandler{hub: s.hub}
	healthHandler := &healthHandler{hub: s.hub}

	// Register routes
	mux.HandleFunc("/send", sendHandler.Handle)
	mux.HandleFunc("/batch", batchHandler.Handle)
	mux.HandleFunc("/health", healthHandler.Handle)
	mux.HandleFunc("/metrics", s.handleMetrics)
}

// handleMetrics handles metrics endpoint
func (s *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.hub.GetMetrics()

	// Simple JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// This is a simplified metrics response
	// In a real implementation, you'd use a proper JSON encoder
	response := `{
		"total_sent": ` + string(rune(metrics.TotalSent)) + `,
		"total_failed": ` + string(rune(metrics.TotalFailed)) + `,
		"success_rate": ` + string(rune(int(metrics.SuccessRate*100))) + `
	}`

	_, _ = w.Write([]byte(response))
}
