package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
)

// NotificationService 封装了 NotifyHub 的核心功能
type NotificationService struct {
	hub *client.Hub
}

// NewNotificationService 创建通知服务实例 - 展示最佳实践的初始化
func NewNotificationService() (*NotificationService, error) {
	// 优化：使用统一的环境配置，减少样板代码
	hub, err := client.New(
		config.WithFeishuFromEnv(),     // 自动从环境变量加载 Feishu 配置
		config.WithEmailFromEnv(),      // 自动从环境变量加载 Email 配置
		config.WithQueueFromEnv(),      // 自动从环境变量加载队列配置
		config.WithTelemetryFromEnv(),  // 自动从环境变量加载遥测配置
		config.WithMockNotifier("dev"), // 开发环境自动启用 Mock
	)
	if err != nil {
		return nil, err
	}

	return &NotificationService{hub: hub}, nil
}

// Start 启动通知服务
func (ns *NotificationService) Start(ctx context.Context) error {
	return ns.hub.Start(ctx)
}

// Stop 停止通知服务
func (ns *NotificationService) Stop() {
	ns.hub.Stop()
}

// =====================
// HTTP 请求/响应结构
// =====================

// NotificationRequest 统一的通知请求结构
type NotificationRequest struct {
	Type        string                 `json:"type" binding:"required"`        // alert, notice, report
	Title       string                 `json:"title" binding:"required"`       // 标题
	Message     string                 `json:"message" binding:"required"`     // 消息内容
	Priority    int                    `json:"priority,omitempty"`             // 优先级 (1-5)
	Targets     []Target               `json:"targets" binding:"required"`     // 目标列表
	Variables   map[string]interface{} `json:"variables,omitempty"`            // 模板变量
	Metadata    map[string]string      `json:"metadata,omitempty"`             // 元数据
	Template    string                 `json:"template,omitempty"`             // 模板名称
	Async       bool                   `json:"async,omitempty"`                // 是否异步发送
	RetryCount  int                    `json:"retry_count,omitempty"`          // 重试次数
	Timeout     int                    `json:"timeout_seconds,omitempty"`      // 超时时间(秒)
}

// Target 目标配置
type Target struct {
	Type     string `json:"type" binding:"required"`      // email, group, user
	Value    string `json:"value" binding:"required"`     // 目标值
	Platform string `json:"platform,omitempty"`          // 平台 (feishu, email)
}

// NotificationResponse 通知响应
type NotificationResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message,omitempty"`
	TaskID    string                 `json:"task_id,omitempty"`    // 异步任务ID
	Results   []PlatformResult       `json:"results,omitempty"`    // 同步发送结果
	Metadata  map[string]interface{} `json:"metadata,omitempty"`   // 响应元数据
}

// PlatformResult 平台发送结果
type PlatformResult struct {
	Platform string        `json:"platform"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string             `json:"status"`                  // ok, warning, error
	Uptime    string             `json:"uptime"`                  // 运行时间
	Metrics   map[string]interface{} `json:"metrics"`             // 指标信息
	Platforms map[string]bool    `json:"platforms"`               // 平台健康状态
	Version   string             `json:"version,omitempty"`       // 版本信息
}

// =====================
// HTTP 处理器
// =====================

// sendNotification 发送通知的优化实现
func (ns *NotificationService) sendNotification(c *gin.Context) {
	var req NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 优化：使用流式构建器模式，减少样板代码
	builder := ns.createMessageBuilder(req)
	message := builder.Build()

	// 优化：根据请求自动选择同步/异步模式
	if req.Async {
		ns.handleAsyncSend(c, message, req)
	} else {
		ns.handleSyncSend(c, message, req)
	}
}

// createMessageBuilder 创建消息构建器 - 展示优化的构建器模式
func (ns *NotificationService) createMessageBuilder(req NotificationRequest) *client.MessageBuilder {
	var builder *client.MessageBuilder

	// 优化：根据类型自动选择构建器，减少条件判断
	switch req.Type {
	case "alert":
		builder = client.NewAlert(req.Title, req.Message)
	case "notice":
		builder = client.NewNotice(req.Title, req.Message)
	case "report":
		builder = client.NewReport(req.Title, req.Message)
	default:
		builder = client.NewMessage().Title(req.Title).Body(req.Message)
	}

	// 优化：链式调用减少样板代码
	if req.Priority > 0 {
		builder = builder.Priority(req.Priority)
	}

	if req.Template != "" {
		builder = builder.Template(req.Template)
	}

	// 批量添加变量和元数据
	for k, v := range req.Variables {
		builder = builder.Variable(k, v)
	}

	for k, v := range req.Metadata {
		builder = builder.Metadata(k, v)
	}

	// 批量添加目标 - 优化的目标处理
	for _, target := range req.Targets {
		switch target.Type {
		case "email":
			builder = builder.Email(target.Value)
		case "group":
			if target.Platform == "feishu" {
				builder = builder.FeishuGroup(target.Value)
			}
		case "user":
			if target.Platform == "feishu" {
				builder = builder.FeishuUser(target.Value)
			}
		}
	}

	return builder
}

// handleAsyncSend 处理异步发送 - 展示异步最佳实践
func (ns *NotificationService) handleAsyncSend(c *gin.Context, message *client.Message, req NotificationRequest) {
	opts := client.NewAsyncOptions()
	if req.Timeout > 0 {
		opts = opts.WithTimeout(time.Duration(req.Timeout) * time.Second)
	}

	taskID, err := ns.hub.SendAsync(c.Request.Context(), message, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue message: " + err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, NotificationResponse{
		Success: true,
		Message: "Notification queued successfully",
		TaskID:  taskID,
		Metadata: map[string]interface{}{
			"queued_at": time.Now().UTC(),
			"async":     true,
		},
	})
}

// handleSyncSend 处理同步发送 - 展示同步最佳实践
func (ns *NotificationService) handleSyncSend(c *gin.Context, message *client.Message, req NotificationRequest) {
	// 优化：动态配置重试选项
	retryOpts := client.NewRetryOptions(3) // 默认重试3次
	if req.RetryCount > 0 {
		retryOpts = client.NewRetryOptions(req.RetryCount)
	}
	if req.Timeout > 0 {
		retryOpts = retryOpts.WithTimeout(time.Duration(req.Timeout) * time.Second)
	}

	results, err := ns.hub.Send(c.Request.Context(), message, retryOpts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send notification: " + err.Error()})
		return
	}

	// 转换结果格式
	platformResults := make([]PlatformResult, len(results))
	allSuccess := true
	for i, result := range results {
		platformResults[i] = PlatformResult{
			Platform: result.Platform,
			Success:  result.Success,
			Duration: result.Duration,
		}
		if result.Error != nil {
			platformResults[i].Error = result.Error.Error()
			allSuccess = false
		}
	}

	status := http.StatusOK
	if !allSuccess {
		status = http.StatusPartialContent // 部分成功
	}

	c.JSON(status, NotificationResponse{
		Success: allSuccess,
		Message: "Notification sent",
		Results: platformResults,
		Metadata: map[string]interface{}{
			"sent_at":       time.Now().UTC(),
			"async":         false,
			"total_targets": len(req.Targets),
		},
	})
}

// healthCheck 健康检查 - 展示监控最佳实践
func (ns *NotificationService) healthCheck(c *gin.Context) {
	health := ns.hub.GetHealth(c.Request.Context())
	metrics := ns.hub.GetMetrics()

	status := "ok"
	if !health.Healthy {
		status = "error"
	}

	response := HealthResponse{
		Status:    status,
		Uptime:    health.Uptime.String(),
		Metrics:   metrics,
		Platforms: health.Platforms,
		Version:   "1.2.0", // 可从构建时注入
	}

	httpStatus := http.StatusOK
	if !health.Healthy {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, response)
}

// getMetrics 获取详细指标
func (ns *NotificationService) getMetrics(c *gin.Context) {
	metrics := ns.hub.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"metrics":   metrics,
		"timestamp": time.Now().UTC(),
	})
}

// =====================
// 服务器设置和中间件
// =====================

// setupRouter 设置路由 - 展示 RESTful API 最佳实践
func (ns *NotificationService) setupRouter() *gin.Engine {
	// 优化：生产环境配置
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 中间件配置
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())      // CORS 支持
	router.Use(rateLimitMiddleware()) // 限流中间件

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 通知相关
		api.POST("/notifications", ns.sendNotification)
		api.POST("/notifications/send", ns.sendNotification) // 兼容性别名

		// 快捷方法 - 减少客户端代码复杂度
		api.POST("/alert", ns.sendAlert)
		api.POST("/notice", ns.sendNotice)
		api.POST("/report", ns.sendReport)

		// 监控和健康检查
		api.GET("/health", ns.healthCheck)
		api.GET("/metrics", ns.getMetrics)
		api.GET("/status", ns.healthCheck) // 兼容性别名
	}

	// 根路径
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    "NotifyHub HTTP Service",
			"version": "1.2.0",
			"status":  "running",
			"docs":    "/api/v1/health",
		})
	})

	return router
}

// 快捷发送方法 - 减少客户端复杂度

func (ns *NotificationService) sendAlert(c *gin.Context) {
	ns.sendQuickNotification(c, "alert")
}

func (ns *NotificationService) sendNotice(c *gin.Context) {
	ns.sendQuickNotification(c, "notice")
}

func (ns *NotificationService) sendReport(c *gin.Context) {
	ns.sendQuickNotification(c, "report")
}

// sendQuickNotification 快速发送的简化实现
func (ns *NotificationService) sendQuickNotification(c *gin.Context, notificationType string) {
	var req NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 自动设置类型
	req.Type = notificationType
	ns.sendNotification(c)
}

// =====================
// 中间件
// =====================

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// rateLimitMiddleware 简单的限流中间件
func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 简单的基于 IP 的限流（生产环境建议使用 Redis）
		// 这里只是示例，实际应用中需要更复杂的限流逻辑
		c.Next()
	}
}

// =====================
// 主函数和生命周期管理
// =====================

func main() {
	// 创建通知服务
	service, err := NewNotificationService()
	if err != nil {
		log.Fatalf("Failed to create notification service: %v", err)
	}

	// 启动通知服务
	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		log.Fatalf("Failed to start notification service: %v", err)
	}

	// 设置路由
	router := service.setupRouter()

	// 服务器配置
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅启动
	go func() {
		log.Printf("🚀 NotifyHub HTTP Service starting on port %s", port)
		log.Printf("📖 Health check: http://localhost:%s/api/v1/health", port)
		log.Printf("📊 Metrics: http://localhost:%s/api/v1/metrics", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 优雅停止
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// 给服务器 30 秒时间完成现有请求
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// 停止通知服务
	service.Stop()
	log.Println("✅ Server gracefully stopped")
}

// =====================
// 工具函数
// =====================

// getEnv 获取环境变量，如果不存在则使用默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整型环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}