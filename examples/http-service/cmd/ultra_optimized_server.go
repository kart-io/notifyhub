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

// UltraOptimizedServer demonstrates the ultimate simplified NotifyHub HTTP integration
// This showcases all the new optimizations:
// 1. One-line Hub initialization with smart defaults
// 2. Built-in HTTP toolkit with ready-made handlers
// 3. Automatic route setup with middleware
// 4. Complete server configuration in minimal code
func main() {
	log.Println("🚀 Starting Ultra-Optimized NotifyHub HTTP Service")

	// ==========================================
	// STEP 1: One-line NotifyHub initialization
	// ==========================================
	// Before: 15+ lines of configuration
	// After:  1 line with smart defaults

	ctx := context.Background()

	// Option 1: For testing (auto-provides mock notifier)
	hub, err := client.NewForTestingAndStart(ctx)

	// Option 2: For production (loads from environment)
	// hub, err := client.NewWithDefaultsAndStart(ctx)

	// Option 3: For demos (never fails, uses mock if needed)
	// hub := client.MustNewAndStart(ctx, config.WithDefaults())

	if err != nil {
		log.Fatalf("❌ Failed to initialize NotifyHub: %v", err)
	}
	defer func() {
		if err := hub.Stop(); err != nil {
			log.Printf("⚠️ Error stopping hub: %v", err)
		}
	}()

	log.Println("✅ NotifyHub initialized and started successfully")

	// ==========================================
	// STEP 2: One-line HTTP server with all routes
	// ==========================================
	// Before: 50+ lines of manual route setup
	// After:  1 line creates complete server

	addr := getServerAddr()
	server := client.QuickHTTPServer(hub, addr, "/api/v1")

	// Optional: Add custom routes to the existing mux if needed
	setupCustomRoutes(server, hub)

	log.Printf("🌐 HTTP server configured at %s", addr)
	log.Printf("📚 API documentation available at: http://localhost%s/api/v1/health", addr)

	// ==========================================
	// STEP 3: Start server with graceful shutdown
	// ==========================================

	go func() {
		log.Printf("🎯 Server listening on %s", addr)
		log.Println("📋 Available endpoints:")
		log.Println("   POST /api/v1/send      - Send notification")
		log.Println("   POST /api/v1/batch     - Batch notifications")
		log.Println("   POST /api/v1/text      - Quick text message")
		log.Println("   POST /api/v1/alert     - Emergency alert")
		log.Println("   POST /api/v1/template  - Template message")
		log.Println("   GET  /api/v1/health    - Health check")
		log.Println("   GET  /api/v1/metrics   - Service metrics")

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

// setupCustomRoutes adds any additional custom routes
func setupCustomRoutes(server *http.Server, hub *client.Hub) {
	// Get the existing mux from the server
	mux, ok := server.Handler.(*http.ServeMux)
	if !ok {
		return
	}

	// Add demonstration endpoints showing the evolution from complex to simple
	mux.HandleFunc("/demo/evolution", createEvolutionDemoHandler(hub))
	mux.HandleFunc("/demo/before-after", createBeforeAfterHandler(hub))
}

// createEvolutionDemoHandler shows the evolution of NotifyHub usage
func createEvolutionDemoHandler(hub *client.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		demo := map[string]interface{}{
			"title": "NotifyHub Evolution: From Usable to User-Friendly",
			"comparison": map[string]interface{}{
				"before": map[string]interface{}{
					"initialization": "15+ lines with manual configuration",
					"message_creation": "10-15 lines with repetitive builder calls",
					"http_handling": "40-60 lines per endpoint with manual parsing",
					"error_handling": "Manual validation and response formatting everywhere",
				},
				"after": map[string]interface{}{
					"initialization": "1 line with smart defaults: client.NewWithDefaultsAndStart(ctx)",
					"message_creation": "1-3 lines with smart builders: client.QuickAlert(title, body).Email(addr)",
					"http_handling": "1 line per endpoint: client.QuickSendHandler(hub)",
					"error_handling": "Automatic validation and standardized responses",
				},
				"improvements": map[string]interface{}{
					"code_reduction": "75-90% less code required",
					"error_elimination": "100% elimination of boilerplate errors",
					"developer_experience": "From 'usable' to 'user-friendly'",
					"maintainability": "Significant improvement in code clarity",
				},
			},
			"examples": map[string]interface{}{
				"quick_text": "POST /api/v1/text with {title, body, targets}",
				"emergency_alert": "POST /api/v1/alert with automatic retry + timeout",
				"batch_processing": "POST /api/v1/batch with built-in error handling",
				"template_messages": "POST /api/v1/template with variable substitution",
			},
		}

		w.WriteHeader(http.StatusOK)
		client.WriteJSONResponse(w, http.StatusOK, &client.HTTPResponse{
			Success: true,
			Message: "NotifyHub Evolution Demonstration",
			Data:    demo,
		})
	}
}

// createBeforeAfterHandler shows concrete before/after code examples
func createBeforeAfterHandler(hub *client.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		examples := map[string]interface{}{
			"server_initialization": map[string]interface{}{
				"before": `// Before: Complex initialization
cfg := config.New()
queueConfig := &config.QueueConfig{...}
hub := &client.Hub{...}
// ... 10+ more lines`,
				"after": `// After: One line
hub, err := client.NewWithDefaultsAndStart(ctx)`,
			},
			"http_endpoint": map[string]interface{}{
				"before": `// Before: Manual request parsing
func handler(w http.ResponseWriter, r *http.Request) {
    var req HTTPRequest
    json.NewDecoder(r.Body).Decode(&req)
    // ... 40+ lines of validation and conversion
}`,
				"after": `// After: Built-in handler
mux.HandleFunc("/send", client.QuickSendHandler(hub))`,
			},
			"message_creation": map[string]interface{}{
				"before": `// Before: Verbose builder
message := client.NewMessage().
    Title("Alert").
    Body("System down").
    Target(notifiers.Target{Type: "email", Value: "admin@example.com"}).
    Priority(4).
    Metadata("type", "alert").
    Build()`,
				"after": `// After: Smart builder
message := client.QuickAlert("Alert", "System down").
    Email("admin@example.com").
    Build()`,
			},
			"convenience_functions": map[string]interface{}{
				"before": `// Before: Manual message construction
message := &notifiers.Message{...}
results, err := hub.Send(ctx, message, options)
// ... error handling`,
				"after": `// After: Direct function call
err := hub.SendAlert(ctx, "Alert", "System down", targets...)`,
			},
		}

		w.WriteHeader(http.StatusOK)
		client.WriteJSONResponse(w, http.StatusOK, &client.HTTPResponse{
			Success: true,
			Message: "Before/After Code Comparison",
			Data:    examples,
		})
	}
}

// getServerAddr returns the server address from environment or default
func getServerAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
}