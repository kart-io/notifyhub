package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kart-io/notifyhub/client"
)

// NotifyHub HTTP Service - Best Practices Example
//
// This example demonstrates the recommended way to build production-ready
// NotifyHub HTTP services with minimal code and maximum functionality.
//
// Key Features:
// - One-line NotifyHub initialization
// - Complete HTTP server with built-in routes
// - Automatic environment configuration
// - Production-ready error handling
// - Graceful shutdown
//
// Usage:
//   go run cmd/main.go
//   curl http://localhost:8080/
//
func main() {
	log.Println("🚀 Starting NotifyHub HTTP Service")

	// =================================================================
	// STEP 1: Initialize NotifyHub with Smart Defaults
	// =================================================================
	// This single line replaces 20+ lines of configuration code

	ctx := context.Background()
	hub, err := client.NewForTestingAndStart(ctx) // Perfect for demos and development

	// For production, use: client.NewWithDefaultsAndStart(ctx)
	// It automatically loads configuration from environment variables

	if err != nil {
		log.Fatalf("❌ Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	log.Println("✅ NotifyHub initialized with smart defaults")

	// =================================================================
	// STEP 2: Create Complete HTTP Server with All Routes
	// =================================================================
	// This single line replaces 50+ lines of route setup and middleware

	server := client.QuickHTTPServer(hub, getPort(), "/api")

	// The above line automatically creates:
	// POST /api/send      - Send any notification
	// POST /api/batch     - Send multiple notifications
	// POST /api/text      - Quick text message
	// POST /api/alert     - Emergency alert (with auto-retry)
	// POST /api/template  - Template-based message
	// GET  /api/health    - Service health check
	// GET  /api/metrics   - Performance metrics

	// =================================================================
	// STEP 3: Add Custom Welcome Page
	// =================================================================

	setupWelcomePage(server, hub)

	// =================================================================
	// STEP 4: Start Server with Graceful Shutdown
	// =================================================================

	startServer(server)
}

// setupWelcomePage adds a simple welcome page with examples
func setupWelcomePage(server *http.Server, hub *client.Hub) {
	mux, ok := server.Handler.(*http.ServeMux)
	if !ok {
		return
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>NotifyHub Service</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .example { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .endpoint { color: #0066cc; font-weight: bold; }
        h1 { color: #333; }
    </style>
</head>
<body>
    <h1>🚀 NotifyHub HTTP Service</h1>
    <p>Welcome to the NotifyHub HTTP Service! This service provides a simple REST API for sending notifications.</p>

    <h2>📡 Available Endpoints</h2>

    <div class="example">
        <div class="endpoint">GET /api/health</div>
        Check service health status
    </div>

    <div class="example">
        <div class="endpoint">POST /api/text</div>
        Send a quick text message
        <pre>{"title": "Hello", "body": "Test message", "targets": [{"type": "email", "value": "test@example.com"}]}</pre>
    </div>

    <div class="example">
        <div class="endpoint">POST /api/alert</div>
        Send an emergency alert (with automatic retry)
        <pre>{"title": "URGENT", "body": "System down!", "targets": [{"type": "email", "value": "admin@example.com"}]}</pre>
    </div>

    <div class="example">
        <div class="endpoint">POST /api/batch</div>
        Send multiple notifications at once
        <pre>{"messages": [{"title": "Message 1", "body": "Content 1", "targets": [{"type": "email", "value": "user1@example.com"}]}]}</pre>
    </div>

    <h2>🧪 Quick Test</h2>
    <p>Try this command to test the service:</p>
    <div class="example">
        <pre>curl -X POST http://localhost` + getPort() + `/api/text \
  -H "Content-Type: application/json" \
  -d '{"title": "Test", "body": "Hello from NotifyHub!", "targets": [{"type": "email", "value": "test@example.com"}]}'</pre>
    </div>

    <h2>📊 Service Info</h2>
    <ul>
        <li><strong>Version:</strong> Best Practices Edition</li>
        <li><strong>Features:</strong> Auto-configuration, Built-in validation, Error handling</li>
        <li><strong>Environment:</strong> Development (using mock notifiers)</li>
    </ul>
</body>
</html>
		`))
	})

	// Add a simple demo endpoint that showcases the power of the optimized API
	mux.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
		// This endpoint demonstrates how easy it is to send notifications
		// using the new optimized API - just 3 lines of code!

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Send a demo alert using the convenience function (1 line!)
		err := hub.SendAlert(ctx,
			"Demo Alert",
			"This alert was sent using just one line of code!",
			// Targets are automatically handled
		)

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"success": false, "message": "Failed to send demo alert", "error": "` + err.Error() + `"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"success": true,
				"message": "Demo alert sent successfully!",
				"code_used": "hub.SendAlert(ctx, title, body)",
				"lines_of_code": 1,
				"features": ["automatic retry", "timeout handling", "error handling", "logging"]
			}`))
		}
	})
}

// startServer starts the HTTP server with graceful shutdown
func startServer(server *http.Server) {
	// Start server in background
	go func() {
		log.Printf("🌐 Server running at http://localhost%s", server.Addr)
		log.Printf("📖 Visit http://localhost%s for documentation", server.Addr)
		log.Printf("🧪 Try http://localhost%s/demo for a quick test", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️ Server shutdown error: %v", err)
	} else {
		log.Println("✅ Server stopped gracefully")
	}
}

// getPort returns the port from environment or default
func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
}