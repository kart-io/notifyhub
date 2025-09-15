package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/examples/http-service/internal/handlers"
	"github.com/kart-io/notifyhub/examples/http-service/internal/middleware"
	"github.com/kart-io/notifyhub/logger"
)

type Config struct {
	Port               int
	APIKey             string
	RateLimitPerMinute int
	MaxRequestSize     int64
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	LogLevel           string
}

func main() {
	log.Println("🚀 Starting NotifyHub HTTP Service (Optimized Version)")

	// ==========================================
	// STEP 1: Simplified configuration and initialization
	// ==========================================

	cfg := loadConfig()
	ctx := context.Background()

	// ✅ AFTER: One-line NotifyHub initialization with smart defaults
	hub, err := initializeHubWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize NotifyHub: %v", err)
	}
	defer func() {
		if err := hub.Stop(); err != nil {
			log.Printf("⚠️ Hub shutdown error: %v", err)
		}
	}()

	log.Println("✅ NotifyHub initialized successfully")

	// ==========================================
	// STEP 2: Optimized HTTP server setup
	// ==========================================

	server := createOptimizedHTTPServer(hub, cfg)
	startServerWithGracefulShutdown(server)
}

// ==========================================
// Optimized Helper Functions
// ==========================================

// initializeHubWithConfig initializes Hub with configuration-aware defaults
func initializeHubWithConfig(ctx context.Context, cfg *Config) (*client.Hub, error) {
	// Parse log level
	logLevel := parseLogLevel(cfg.LogLevel)

	// ✅ AFTER: Simple configuration with environment fallback
	return client.NewAndStart(ctx,
		config.WithDefaults(),           // Load from environment
		config.WithDefaultLogger(logLevel), // Set log level
	)
}

// createOptimizedHTTPServer creates server with built-in NotifyHub routes
func createOptimizedHTTPServer(hub *client.Hub, cfg *Config) *http.Server {
	mux := http.NewServeMux()

	// ==========================================
	// Built-in routes (recommended approach)
	// ==========================================

	// ✅ AFTER: One-line route setup with all standard endpoints
	client.SetupNotifyHubRoutes(mux, hub, "/api/v1")

	// ==========================================
	// Custom handlers for specific business logic
	// ==========================================

	// Create handlers only when needed for custom logic
	notificationHandler := handlers.NewNotificationHandler(hub, hub.GetLogger())
	demoHandler := handlers.NewDemoHandler(hub, hub.GetLogger())

	// Add custom middleware only for specific routes that need it
	setupCustomRoutes(mux, notificationHandler, demoHandler, cfg)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
}

// setupCustomRoutes sets up routes that need custom business logic
func setupCustomRoutes(mux *http.ServeMux, handler *handlers.NotificationHandler, demoHandler *handlers.DemoHandler, cfg *Config) {
	// ✅ AFTER: Use NotifyHub middleware for consistent behavior
	middleware := func(h http.HandlerFunc) http.Handler {
		return client.NotifyHubMiddleware(nil, func(w http.ResponseWriter, r *http.Request, hub *client.Hub) {
			h(w, r)
		})
	}

	// Demo routes showcasing different approaches
	mux.Handle("/api/v1/demo/builder", middleware(demoHandler.DemoBuilderAPI))
	mux.Handle("/api/v1/demo/convenience", middleware(demoHandler.DemoConvenienceFunctions))
	mux.Handle("/api/v1/demo/http-utils", middleware(demoHandler.DemoHTTPUtilities))

	// Example of using built-in handlers vs custom handlers
	mux.HandleFunc("/api/v1/compare/built-in", client.QuickSendHandler(handler.hub))
	mux.HandleFunc("/api/v1/compare/custom", handler.SendNotification)

	// Documentation routes
	mux.HandleFunc("/api/docs", createDocumentationHandler(cfg))
}

// startServerWithGracefulShutdown starts server and handles shutdown
func startServerWithGracefulShutdown(server *http.Server) {
	// Start server
	go func() {
		log.Printf("🎯 Server listening on %s", server.Addr)
		log.Println("📋 Available built-in endpoints:")
		log.Println("   POST /api/v1/send         - Send notification")
		log.Println("   POST /api/v1/batch        - Batch notifications")
		log.Println("   POST /api/v1/text         - Quick text message")
		log.Println("   POST /api/v1/alert        - Emergency alert")
		log.Println("   POST /api/v1/template     - Template message")
		log.Println("   GET  /api/v1/health       - Health check")
		log.Println("   GET  /api/v1/metrics      - Service metrics")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️ Server shutdown error: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}

// parseLogLevel parses string log level to logger.LogLevel
func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "debug":
		return logger.Debug
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	case "silent":
		return logger.Silent
	default:
		return logger.Info
	}
}

// createDocumentationHandler creates API documentation endpoint
func createDocumentationHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docs := map[string]interface{}{
			"service": "NotifyHub HTTP Service",
			"version": "2.0.0 (Optimized)",
			"description": "Production-ready HTTP service with optimized NotifyHub integration",
			"optimizations": map[string]interface{}{
				"initialization": "One-line Hub setup with smart defaults",
				"routing": "Built-in route setup with automatic middleware",
				"handlers": "Ready-made handlers for common patterns",
				"error_handling": "Automatic validation and structured responses",
			},
			"built_in_endpoints": map[string]string{
				"POST /api/v1/send": "Universal message sending",
				"POST /api/v1/batch": "Batch processing",
				"POST /api/v1/text": "Quick text messages",
				"POST /api/v1/alert": "Emergency alerts with retry",
				"POST /api/v1/template": "Template-based messages",
				"GET /api/v1/health": "Service health check",
				"GET /api/v1/metrics": "Performance metrics",
			},
			"demo_endpoints": map[string]string{
				"POST /api/v1/demo/builder": "Builder pattern showcase",
				"POST /api/v1/demo/convenience": "Convenience functions demo",
				"POST /api/v1/demo/http-utils": "HTTP utilities demonstration",
			},
			"configuration": map[string]interface{}{
				"port": cfg.Port,
				"api_key_required": cfg.APIKey != "",
				"rate_limit": cfg.RateLimitPerMinute,
			},
		}

		client.WriteJSONResponse(w, http.StatusOK, &client.HTTPResponse{
			Success: true,
			Message: "NotifyHub Optimized Service Documentation",
			Data:    docs,
		})
	}
}

func loadConfig() *Config {
	cfg := &Config{
		Port:               8080,
		APIKey:             os.Getenv("API_KEY"),
		RateLimitPerMinute: 60,
		MaxRequestSize:     1024 * 1024, // 1MB
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       10 * time.Second,
		LogLevel:           "info",
	}

	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if rateLimit := os.Getenv("RATE_LIMIT_PER_MINUTE"); rateLimit != "" {
		if r, err := strconv.Atoi(rateLimit); err == nil {
			cfg.RateLimitPerMinute = r
		}
	}

	if maxSize := os.Getenv("MAX_REQUEST_SIZE"); maxSize != "" {
		if s, err := strconv.ParseInt(maxSize, 10, 64); err == nil {
			cfg.MaxRequestSize = s
		}
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	return cfg
}