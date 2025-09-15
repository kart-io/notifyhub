package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kart-io/notifyhub/client"
	notifyConfig "github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/examples/http-service/internal/config"
	"github.com/kart-io/notifyhub/examples/http-service/internal/handlers"
	"github.com/kart-io/notifyhub/examples/http-service/internal/middleware"
	"github.com/kart-io/notifyhub/logger"
)

func main() {
	// 1. 加载配置 - 统一配置管理
	configManager := config.NewManager()
	cfg, err := configManager.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. 初始化NotifyHub - 简化的生命周期管理
	hub, err := initializeNotifyHub(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize NotifyHub: %v", err)
	}
	defer hub.Stop()

	// 3. 创建HTTP服务器 - 使用优化的handler和middleware
	server := createServer(cfg, hub)

	// 4. 启动服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Starting HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 5. 优雅停机
	gracefulShutdown(server)
}

// initializeNotifyHub 初始化NotifyHub实例
func initializeNotifyHub(cfg *config.Config) (*client.Hub, error) {
	// 解析日志级别
	logLevel := parseLogLevel(cfg.NotifyHub.LogLevel)

	// 构建NotifyHub配置选项
	var options []notifyConfig.Option

	if cfg.NotifyHub.UseDefaults {
		options = append(options, notifyConfig.WithDefaults())
	}

	options = append(options, notifyConfig.WithDefaultLogger(logLevel))

	// 如果是mock模式，添加mock notifier
	if cfg.NotifyHub.MockMode {
		options = append(options, notifyConfig.WithMockNotifier("http-service-mock"))
	}

	// 使用NewAndStart简化生命周期管理
	ctx := context.Background()
	hub, err := client.NewAndStart(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create and start NotifyHub: %v", err)
	}

	// 获取logger并记录启动成功
	hubLogger := hub.GetLogger()
	hubLogger.Info(ctx, "NotifyHub started successfully with optimized configuration")

	return hub, nil
}

// createServer 创建HTTP服务器
func createServer(cfg *config.Config, hub *client.Hub) *http.Server {
	// 获取logger
	hubLogger := hub.GetLogger()

	// 创建handler
	optimizedHandler := handlers.NewOptimizedNotificationHandler(hub, hubLogger)
	standardHandler := handlers.NewNotificationHandler(hub, hubLogger)
	demoHandler := handlers.NewDemoHandler(hub, hubLogger)

	// 创建中间件链构建器
	chainBuilder := middleware.NewChainBuilder(hubLogger)

	// 创建路由多路复用器
	mux := http.NewServeMux()

	// 设置路由
	setupOptimizedRoutes(mux, optimizedHandler, standardHandler, demoHandler, chainBuilder, cfg)

	// 创建服务器
	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

// setupOptimizedRoutes 设置优化的路由
func setupOptimizedRoutes(
	mux *http.ServeMux,
	optimizedHandler *handlers.OptimizedNotificationHandler,
	standardHandler *handlers.NotificationHandler,
	demoHandler *handlers.DemoHandler,
	chainBuilder *middleware.ChainBuilder,
	cfg *config.Config,
) {
	// 公共路由 - 无需认证
	publicChain := chainBuilder.PublicChain()
	mux.Handle("/health", publicChain.Then(http.HandlerFunc(standardHandler.GetHealth)))
	mux.Handle("/metrics", publicChain.Then(http.HandlerFunc(standardHandler.GetMetrics)))

	// 监控路由 - 带指标收集
	monitoringChain := chainBuilder.MonitoringChain()
	mux.Handle("/monitoring/health", monitoringChain.Then(http.HandlerFunc(standardHandler.GetHealth)))

	// 受保护的API路由
	var protectedChain *middleware.MiddlewareChain
	if cfg.Auth.Required {
		protectedChain = chainBuilder.ProtectedChain(
			cfg.Auth.APIKey,
			cfg.Server.RateLimitPerMinute,
			cfg.Server.MaxRequestSize,
		)
	} else {
		// 无需认证的保护链
		protectedChain = chainBuilder.PublicChain().
			Add(middleware.RateLimitMiddleware(cfg.Server.RateLimitPerMinute)).
			Add(middleware.RequestSizeLimit(cfg.Server.MaxRequestSize)).
			Add(middleware.ValidateContentType).
			Add(middleware.LoggingMiddleware(chainBuilder.logger))
	}

	// 优化后的API端点
	mux.Handle("/api/v2/notifications", protectedChain.Then(http.HandlerFunc(optimizedHandler.SendNotification)))
	mux.Handle("/api/v2/notifications/quick", protectedChain.Then(http.HandlerFunc(optimizedHandler.SendQuickNotification)))
	mux.Handle("/api/v2/notifications/bulk", protectedChain.Then(http.HandlerFunc(optimizedHandler.SendBulkNotifications)))

	// 兼容性端点 (v1 API)
	mux.Handle("/api/v1/notifications", protectedChain.Then(http.HandlerFunc(standardHandler.SendNotification)))
	mux.Handle("/api/v1/notifications/text", protectedChain.Then(http.HandlerFunc(standardHandler.SendTextNotification)))

	// 演示端点
	mux.Handle("/api/v1/demo/builder", protectedChain.Then(http.HandlerFunc(demoHandler.DemoBuilderAPI)))
	mux.Handle("/api/v1/demo/convenience", protectedChain.Then(http.HandlerFunc(demoHandler.DemoConvenienceFunctions)))
	mux.Handle("/api/v1/demo/http-utils", protectedChain.Then(http.HandlerFunc(demoHandler.DemoHTTPUtilities)))

	// API文档端点
	mux.HandleFunc("/api/docs", createAPIDocsHandler(cfg))
}

// createAPIDocsHandler 创建API文档处理器
func createAPIDocsHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docs := map[string]interface{}{
			"name":        "NotifyHub HTTP Service (Optimized)",
			"version":     "2.0.0",
			"description": "Optimized HTTP service for sending notifications through NotifyHub",
			"features": []string{
				"Unified request parsing",
				"Standardized response format",
				"Advanced error handling",
				"Configurable middleware chains",
				"Builder pattern API",
				"Async/sync notification support",
				"Bulk operations",
			},
			"endpoints": map[string]interface{}{
				// 公共端点
				"GET  /health":           "Service health check",
				"GET  /metrics":          "Service metrics",
				"GET  /monitoring/health": "Health check with monitoring",

				// V2 优化API
				"POST /api/v2/notifications":       "Send notification (optimized)",
				"GET  /api/v2/notifications/quick": "Quick notification via query params",
				"POST /api/v2/notifications/bulk": "Bulk notifications (use ?bulk=true)",

				// V1 兼容API
				"POST /api/v1/notifications":      "Send notification (legacy)",
				"GET  /api/v1/notifications/text": "Text notification (legacy)",

				// 演示端点
				"POST /api/v1/demo/builder":     "Demonstrate Builder API",
				"POST /api/v1/demo/convenience": "Demonstrate convenience functions",
				"POST /api/v1/demo/http-utils":  "Demonstrate HTTP utilities",

				// 文档
				"GET  /api/docs": "API documentation",
			},
			"improvements": map[string]interface{}{
				"code_reduction":     "75% less code for message creation",
				"error_handling":     "Structured validation errors",
				"configuration":      "Unified config management",
				"middleware":         "Composable middleware chains",
				"response_format":    "Standardized HTTP responses",
				"testing_friendly":   "One-line test configuration",
			},
			"authentication": getAuthInfo(cfg),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(docs)
	}
}

// getAuthInfo 获取认证信息
func getAuthInfo(cfg *config.Config) interface{} {
	if cfg.Auth.Required {
		return "Bearer token in Authorization header"
	}
	return "No authentication required"
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "debug":
		return logger.Debug
	case "info":
		return logger.Info
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

// gracefulShutdown 优雅停机
func gracefulShutdown(server *http.Server) {
	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// 创建带超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅停机
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}