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
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
)

// ==========================================
// 最简化的 NotifyHub HTTP 服务示例
// 展示最佳实践和推荐用法
// ==========================================

func main() {
	// ✅ 最佳实践 1: 零配置启动
	// 自动从环境变量加载配置
	hub, err := client.NewAndStart(context.Background(), config.WithDefaults())
	if err != nil {
		log.Fatalf("Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	// ✅ 最佳实践 2: 使用内置 HTTP 服务器
	// 一行代码启动完整的 HTTP API
	server := client.QuickHTTPServer(hub, ":8080", "/notify")

	// 添加自定义业务路由（可选）
	mux := server.Handler.(*http.ServeMux)
	setupBusinessRoutes(mux, hub)

	// 启动服务器
	go func() {
		log.Println("🚀 Server starting on :8080")
		log.Println("📋 Available endpoints:")
		log.Println("   POST /notify/send     - Send notification")
		log.Println("   POST /notify/batch    - Batch notifications")
		log.Println("   POST /notify/text     - Quick text message")
		log.Println("   POST /notify/alert    - Emergency alert")
		log.Println("   POST /notify/template - Template message")
		log.Println("   GET  /notify/health   - Health check")
		log.Println("   GET  /notify/metrics  - Service metrics")
		log.Println("")
		log.Println("   POST /quick/email     - Quick email example")
		log.Println("   POST /quick/alert     - Quick alert example")
		log.Println("   POST /quick/broadcast - Broadcast example")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 优雅关闭
	waitForShutdown(server)
}

// ==========================================
// 业务路由示例 - 展示各种优化后的用法
// ==========================================

func setupBusinessRoutes(mux *http.ServeMux, hub *client.Hub) {
	// ✅ 示例 1: 最简单的快速发送
	mux.HandleFunc("/quick/email", func(w http.ResponseWriter, r *http.Request) {
		// 一行代码发送邮件
		err := hub.SendText(r.Context(),
			"System Alert",
			"Database backup completed successfully",
			notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "admin@example.com"})

		if err != nil {
			client.WriteJSONResponse(w, http.StatusInternalServerError,
				client.CreateErrorResponse("Send failed", err.Error()))
			return
		}

		client.WriteJSONResponse(w, http.StatusOK,
			client.CreateSuccessResponse("Email sent", nil))
	})

	// ✅ 示例 2: 使用 Builder API 的紧急告警
	mux.HandleFunc("/quick/alert", func(w http.ResponseWriter, r *http.Request) {
		// 链式构建复杂消息
		message := client.QuickAlert("CRITICAL: Service Down", "Payment service is not responding").
			Email("oncall@example.com").
			ToSlack("#incidents").
			ToFeishu("ops-team").
			Metadata("severity", "critical").
			Metadata("service", "payment").
			Build()

		// 发送并自动重试
		_, err := hub.Send(r.Context(), message, &client.Options{
			Retry:    true,
			MaxRetry: 3,
			Timeout:  30 * time.Second,
		})

		if err != nil {
			client.WriteJSONResponse(w, http.StatusInternalServerError,
				client.CreateErrorResponse("Alert failed", err.Error()))
			return
		}

		client.WriteJSONResponse(w, http.StatusOK,
			client.CreateSuccessResponse("Alert sent to multiple platforms", nil))
	})

	// ✅ 示例 3: 批量广播
	mux.HandleFunc("/quick/broadcast", func(w http.ResponseWriter, r *http.Request) {
		// 使用便捷的批量邮件构建器
		message := client.BatchEmail(
			"Weekly Newsletter",
			"Check out our latest updates and features!",
			"user1@example.com",
			"user2@example.com",
			"user3@example.com",
		).AsHTML().Build() // 支持 HTML 格式

		// 异步发送
		taskID, err := hub.SendAsync(r.Context(), message, nil)

		if err != nil {
			client.WriteJSONResponse(w, http.StatusInternalServerError,
				client.CreateErrorResponse("Broadcast failed", err.Error()))
			return
		}

		client.WriteJSONResponse(w, http.StatusAccepted,
			client.CreateAsyncSuccessResponse(taskID))
	})

	// ✅ 示例 4: 条件消息构建
	mux.HandleFunc("/quick/conditional", func(w http.ResponseWriter, r *http.Request) {
		isProduction := os.Getenv("ENV") == "production"

		// 根据环境动态构建消息
		message := client.NewNotice("Deployment Complete", "Version 2.0.0 deployed").
			Email("team@example.com").
			If(isProduction, func(b *client.MessageBuilder) *client.MessageBuilder {
				// 生产环境：紧急+更多接收者
				return b.Urgent().
					Email("cto@example.com").
					ToSlack("#prod-alerts")
			}).
			Unless(isProduction, func(b *client.MessageBuilder) *client.MessageBuilder {
				// 非生产环境：低优先级
				return b.Low().Metadata("env", "staging")
			}).
			Build()

		_, err := hub.Send(r.Context(), message, nil)

		if err != nil {
			client.WriteJSONResponse(w, http.StatusInternalServerError,
				client.CreateErrorResponse("Send failed", err.Error()))
			return
		}

		client.WriteJSONResponse(w, http.StatusOK,
			client.CreateSuccessResponse("Conditional message sent", map[string]interface{}{
				"production": isProduction,
				"priority":   message.Priority,
			}))
	})

	// ✅ 示例 5: 模板消息
	mux.HandleFunc("/quick/template", func(w http.ResponseWriter, r *http.Request) {
		// 使用模板快速构建
		message := client.TemplatedMessage("daily_report", map[string]interface{}{
			"date":     time.Now().Format("2006-01-02"),
			"users":    1250,
			"revenue":  "$45,678",
			"growth":   "+12%",
		}).Email("reports@example.com").Build()

		_, err := hub.Send(r.Context(), message, nil)

		if err != nil {
			client.WriteJSONResponse(w, http.StatusInternalServerError,
				client.CreateErrorResponse("Template send failed", err.Error()))
			return
		}

		client.WriteJSONResponse(w, http.StatusOK,
			client.CreateSuccessResponse("Template message sent", nil))
	})
}

// ==========================================
// 辅助函数
// ==========================================

func waitForShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}