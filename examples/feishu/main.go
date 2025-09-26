package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
	"github.com/kart-io/notifyhub/pkg/notifyhub/template"
)

func main() {
	fmt.Println("=== 飞书推送示例（模板集成）===")

	// 获取飞书 Webhook URL 和密钥
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "https://httpbin.org/post" // 测试用默认地址
		fmt.Println("使用测试地址: https://httpbin.org/post")
		fmt.Println("设置 FEISHU_WEBHOOK_URL 环境变量以使用真实飞书 Webhook")
	}

	secret := os.Getenv("FEISHU_SECRET")
	keywords := []string{}
	if keywordsStr := os.Getenv("FEISHU_KEYWORDS"); keywordsStr != "" {
		keywords = []string{keywordsStr}
	}

	// 创建日志记录器
	logger := logger.New().LogMode(logger.Info)

	// 创建配置（使用 Platforms 映射）
	cfg := &config.Config{
		Platforms: map[string]map[string]interface{}{
			"feishu": {
				"webhook_url": webhookURL,
				"secret":      secret,
				"keywords":    keywords,
				"timeout":     "30s",
			},
		},
		Logger: logger,
	}

	// 创建 Hub 实例
	hub, err := core.NewHub(cfg)
	if err != nil {
		log.Fatalf("创建 Hub 失败: %v", err)
	}
	defer hub.Close(context.Background())

	// 创建模板管理器
	templateManager, err := createTemplateManager(logger)
	if err != nil {
		log.Fatalf("创建模板管理器失败: %v", err)
	}
	defer templateManager.Close()

	// 注册模板
	err = registerTemplates(templateManager)
	if err != nil {
		log.Fatalf("注册模板失败: %v", err)
	}

	// 创建上下文
	ctx := context.Background()

	fmt.Println("\n=== 飞书推送测试（使用模板）===")

	// 示例1：使用告警模板发送消息
	fmt.Println("\n1. 发送告警消息（使用模板）")
	err = sendAlertMessage(ctx, hub, templateManager)
	if err != nil {
		log.Printf("发送告警消息失败: %v", err)
	}

	time.Sleep(time.Second)

	// 示例2：使用系统状态模板发送消息
	fmt.Println("\n2. 发送系统状态报告（使用模板）")
	err = sendSystemStatusMessage(ctx, hub, templateManager)
	if err != nil {
		log.Printf("发送系统状态消息失败: %v", err)
	}

	time.Sleep(time.Second)

	// 示例3：使用部署通知模板发送消息
	fmt.Println("\n3. 发送部署通知（使用模板）")
	err = sendDeploymentMessage(ctx, hub, templateManager)
	if err != nil {
		log.Printf("发送部署通知失败: %v", err)
	}

	time.Sleep(time.Second)

	// 示例4：使用用户活动模板发送消息（Mustache 引擎）
	fmt.Println("\n4. 发送用户活动通知（使用 Mustache 模板）")
	err = sendUserActivityMessage(ctx, hub, templateManager)
	if err != nil {
		log.Printf("发送用户活动通知失败: %v", err)
	}

	// 健康检查
	fmt.Println("\n=== 健康检查 ===")
	health, err := hub.Health(ctx)
	if err != nil {
		log.Printf("健康检查失败: %v", err)
	} else {
		fmt.Printf("整体健康状态: %s\n", health.Status)
		for platform, platformHealth := range health.Platforms {
			fmt.Printf("  - %s: %s\n", platform, func() string {
				if platformHealth.Available {
					return "健康"
				}
				return "不健康"
			}())
		}
	}

	fmt.Println("\n=== 飞书推送示例完成 ===")
	fmt.Println("✅ 模板系统集成成功")
	fmt.Println("✅ 多种模板引擎支持")
	fmt.Println("✅ 动态变量替换正常")
}

// createTemplateManager 创建模板管理器
func createTemplateManager(logger logger.Logger) (template.Manager, error) {
	// 创建模板管理器选项
	options := []template.Option{
		template.WithDefaultEngine(template.EngineGo),
		template.WithMemoryCache(5*time.Minute, 1000),
	}

	// 创建模板管理器
	return template.NewManagerWithOptions(logger, options...)
}

// registerTemplates 注册所有模板
func registerTemplates(manager template.Manager) error {
	// 注册告警模板
	alertTemplate, err := os.ReadFile("templates/alert.tmpl")
	if err != nil {
		return fmt.Errorf("读取告警模板失败: %w", err)
	}
	err = manager.RegisterTemplate("alert", string(alertTemplate), template.EngineGo)
	if err != nil {
		return fmt.Errorf("注册告警模板失败: %w", err)
	}

	// 注册系统状态模板
	statusTemplate, err := os.ReadFile("templates/system_status.tmpl")
	if err != nil {
		return fmt.Errorf("读取系统状态模板失败: %w", err)
	}
	err = manager.RegisterTemplate("system_status", string(statusTemplate), template.EngineGo)
	if err != nil {
		return fmt.Errorf("注册系统状态模板失败: %w", err)
	}

	// 注册部署通知模板
	deploymentTemplate, err := os.ReadFile("templates/deployment.tmpl")
	if err != nil {
		return fmt.Errorf("读取部署通知模板失败: %w", err)
	}
	err = manager.RegisterTemplate("deployment", string(deploymentTemplate), template.EngineGo)
	if err != nil {
		return fmt.Errorf("注册部署通知模板失败: %w", err)
	}

	// 注册用户活动模板（Mustache 引擎）
	userActivityTemplate, err := os.ReadFile("templates/user_activity.mustache")
	if err != nil {
		return fmt.Errorf("读取用户活动模板失败: %w", err)
	}
	err = manager.RegisterTemplate("user_activity", string(userActivityTemplate), template.EngineMustache)
	if err != nil {
		return fmt.Errorf("注册用户活动模板失败: %w", err)
	}

	fmt.Printf("✅ 成功注册 %d 个模板\n", len(manager.ListTemplates()))
	return nil
}

// sendAlertMessage 发送告警消息
func sendAlertMessage(ctx context.Context, hub core.Hub, templateManager template.Manager) error {
	// 准备模板变量
	variables := map[string]interface{}{
		"severity":     "critical",
		"service_name": "API Gateway",
		"alert_type":   "High CPU Usage",
		"timestamp":    time.Now().Format("2006-01-02 15:04:05"),
		"duration":     "5分钟",
		"description":  "API Gateway CPU 使用率持续超过 90%，响应时间显著增加",
		"affected_services": []string{
			"用户登录服务",
			"订单处理服务",
			"支付网关",
		},
		"metrics": map[string]interface{}{
			"cpu_usage":    "94",
			"memory_usage": "78",
			"disk_usage":   "45",
		},
		"dashboard_url": "https://monitoring.example.com/dashboard",
		"runbook_url":   "https://docs.example.com/runbooks/api-gateway",
	}

	// 渲染模板
	content, err := templateManager.RenderTemplate(ctx, "alert", variables)
	if err != nil {
		return fmt.Errorf("渲染告警模板失败: %w", err)
	}

	// 创建消息
	msg := &message.Message{
		ID:     "alert-001",
		Title:  "🚨 系统告警",
		Body:   content,
		Format: message.FormatMarkdown,
		Targets: []target.Target{
			{Type: "feishu", Value: "alert", Platform: "feishu"},
		},
	}

	// 发送消息
	receipt, err := hub.Send(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Printf("告警消息发送结果: %s\n", receipt.Status)
	for _, result := range receipt.Results {
		fmt.Printf("  - 平台: %s, 成功: %t, 消息ID: %s\n",
			result.Platform, result.Success, result.MessageID)
		if !result.Success {
			fmt.Printf("    错误: %s\n", result.Error)
		}
	}

	return nil
}

// sendSystemStatusMessage 发送系统状态消息
func sendSystemStatusMessage(ctx context.Context, hub core.Hub, templateManager template.Manager) error {
	variables := map[string]interface{}{
		"report_date": time.Now().Format("2006-01-02"),
		"services": []map[string]interface{}{
			{
				"name":          "Web 前端",
				"status":        "healthy",
				"response_time": 120,
				"error_rate":    0.1,
				"uptime":        "99.9%",
			},
			{
				"name":          "API 服务",
				"status":        "warning",
				"response_time": 350,
				"error_rate":    2.5,
				"uptime":        "98.5%",
			},
			{
				"name":          "数据库",
				"status":        "healthy",
				"response_time": 45,
				"error_rate":    0.0,
				"uptime":        "100%",
			},
		},
		"server": map[string]interface{}{
			"name":         "prod-server-01",
			"cpu_usage":    65,
			"memory_usage": 72,
			"memory_used":  "5.8GB",
			"memory_total": "8GB",
			"disk_usage":   45,
			"disk_used":    "180GB",
			"disk_total":   "400GB",
			"network_tx":   "150MB/s",
			"network_rx":   "80MB/s",
		},
		"metrics": []map[string]interface{}{
			{
				"name":   "请求数量",
				"value":  15420,
				"unit":   "/分钟",
				"trend":  "↑",
				"change": "+5.2%",
			},
			{
				"name":   "活跃用户",
				"value":  1250,
				"unit":   "",
				"trend":  "↓",
				"change": "-2.1%",
			},
		},
		"generated_at":   time.Now().Format("2006-01-02 15:04:05"),
		"monitoring_url": "https://monitoring.example.com",
		"trends_url":     "https://monitoring.example.com/trends",
	}

	content, err := templateManager.RenderTemplate(ctx, "system_status", variables)
	if err != nil {
		return fmt.Errorf("渲染系统状态模板失败: %w", err)
	}

	msg := &message.Message{
		ID:     "status-001",
		Title:  "📊 系统状态报告",
		Body:   content,
		Format: message.FormatMarkdown,
		Targets: []target.Target{
			{Type: "feishu", Value: "status", Platform: "feishu"},
		},
	}

	receipt, err := hub.Send(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Printf("系统状态消息发送结果: %s\n", receipt.Status)
	for _, result := range receipt.Results {
		fmt.Printf("  - 平台: %s, 成功: %t, 消息ID: %s\n",
			result.Platform, result.Success, result.MessageID)
		if !result.Success {
			fmt.Printf("    错误: %s\n", result.Error)
		}
	}

	return nil
}

// sendDeploymentMessage 发送部署通知
func sendDeploymentMessage(ctx context.Context, hub core.Hub, templateManager template.Manager) error {
	variables := map[string]interface{}{
		"project_name":    "NotifyHub",
		"environment":     "production",
		"version":         "v3.1.0",
		"status":          "success",
		"deployer":        "张三",
		"start_time":      "2024-09-26 14:30:00",
		"end_time":        "2024-09-26 14:45:00",
		"actual_duration": "15分钟",
		"old_version":     "v3.0.2",
		"new_version":     "v3.1.0",
		"deployment_url":  "https://app.example.com",
		"changes": []map[string]interface{}{
			{
				"type":        "新功能",
				"description": "添加飞书模板支持",
			},
			{
				"type":        "优化",
				"description": "提升消息发送性能",
			},
			{
				"type":        "修复",
				"description": "修复并发发送时的内存泄漏问题",
			},
		},
		"health_checks": []map[string]interface{}{
			{"name": "数据库连接", "passed": true},
			{"name": "API 响应", "passed": true},
			{"name": "缓存服务", "passed": true},
			{"name": "外部依赖", "passed": true},
		},
		"approver":       "李四",
		"logs_url":       "https://logs.example.com/deployment/v3.1.0",
		"monitoring_url": "https://monitoring.example.com",
	}

	content, err := templateManager.RenderTemplate(ctx, "deployment", variables)
	if err != nil {
		return fmt.Errorf("渲染部署通知模板失败: %w", err)
	}

	msg := &message.Message{
		ID:     "deploy-001",
		Title:  "🚀 部署通知",
		Body:   content,
		Format: message.FormatMarkdown,
		Targets: []target.Target{
			{Type: "feishu", Value: "deployment", Platform: "feishu"},
		},
	}

	receipt, err := hub.Send(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Printf("部署通知发送结果: %s\n", receipt.Status)
	for _, result := range receipt.Results {
		fmt.Printf("  - 平台: %s, 成功: %t, 消息ID: %s\n",
			result.Platform, result.Success, result.MessageID)
		if !result.Success {
			fmt.Printf("    错误: %s\n", result.Error)
		}
	}

	return nil
}

// sendUserActivityMessage 发送用户活动通知（使用 Mustache 模板）
func sendUserActivityMessage(ctx context.Context, hub core.Hub, templateManager template.Manager) error {
	variables := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "王五",
			"email": "wang.wu@example.com",
		},
		"activity": map[string]interface{}{
			"type":      "登录",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
			"login": map[string]interface{}{
				"ip_address": "192.168.1.100",
				"device": map[string]interface{}{
					"type": "Desktop",
					"name": "Windows PC",
				},
				"browser": map[string]interface{}{
					"name":    "Chrome",
					"version": "118.0.0.0",
				},
				"location": map[string]interface{}{
					"city":    "深圳",
					"country": "中国",
				},
			},
		},
		"security_alert": map[string]interface{}{
			"risk_level":     "低",
			"description":    "检测到来自新设备的登录",
			"recommendation": "如果不是本人操作，请立即修改密码",
		},
		"details_url":  "https://security.example.com/activity/12345",
		"security_url": "https://security.example.com/settings",
	}

	content, err := templateManager.RenderTemplate(ctx, "user_activity", variables)
	if err != nil {
		return fmt.Errorf("渲染用户活动模板失败: %w", err)
	}

	msg := &message.Message{
		ID:     "activity-001",
		Title:  "👤 用户活动通知",
		Body:   content,
		Format: message.FormatMarkdown,
		Targets: []target.Target{
			{Type: "feishu", Value: "security", Platform: "feishu"},
		},
	}

	receipt, err := hub.Send(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Printf("用户活动通知发送结果: %s\n", receipt.Status)
	for _, result := range receipt.Results {
		fmt.Printf("  - 平台: %s, 成功: %t, 消息ID: %s\n",
			result.Platform, result.Success, result.MessageID)
		if !result.Success {
			fmt.Printf("    错误: %s\n", result.Error)
		}
	}

	return nil
}
