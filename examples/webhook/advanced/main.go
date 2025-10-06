// Package main demonstrates advanced webhook features with NotifyHub
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/examples/common"
	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Webhook", "Advanced")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请修改以下配置为您的实际Webhook信息
	config.Webhook.URL = "https://httpbin.org/post" // 测试用的webhook地址
	config.Webhook.Method = "POST"
	config.Webhook.Headers = map[string]string{
		"Content-Type":    "application/json",
		"Authorization":   "Bearer your_token_here",
		"X-Custom-Header": "NotifyHub-Advanced",
		"X-API-Version":   "v1",
	}
	config.Webhook.AuthType = "bearer"
	config.Webhook.Token = "your_token_here"

	// Check configuration
	if !common.CheckConfigurationPrompt("webhook") {
		logger.Info("已取消运行")
		return
	}

	// Validate configuration
	if err := common.ValidateConfig(config, "webhook"); err != nil {
		logger.Error("配置验证失败: %v", err)
		return
	}

	// Create NotifyHub client with async support
	cfg := config.CreateWebhookConfig()
	cfg.Async.Enabled = true
	cfg.Async.Workers = 4
	cfg.Logger.Level = "debug"

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("NotifyHub客户端创建成功 (异步模式)")

	// Advanced examples
	examples := []struct {
		name string
		fn   func(notifyhub.Client, *common.ExampleConfig, *common.Logger) error
	}{
		{"批量Webhook发送", sendBatchWebhooks},
		{"多目标Webhook", sendMultiTargetWebhook},
		{"异步Webhook发送", sendAsyncWebhook},
		{"模板化Webhook", sendTemplatedWebhook},
		{"带重试的Webhook", sendWebhookWithRetry},
		{"事件流Webhook", sendEventStreamWebhook},
		{"签名验证Webhook", sendSignedWebhook},
	}

	for _, example := range examples {
		if err := example.fn(client, config, logger); err != nil {
			logger.Error("%s失败: %v", example.name, err)
		}

		// 添加延迟避免频率限制
		time.Sleep(2 * time.Second)
	}

	logger.Success("所有高级Webhook示例运行完成!")
}

// sendBatchWebhooks demonstrates batch webhook sending
func sendBatchWebhooks(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("批量Webhook发送")

	messages := []*message.Message{
		createOrderWebhook("order_001", "created"),
		createOrderWebhook("order_002", "paid"),
		createOrderWebhook("order_003", "shipped"),
	}

	ctx := context.Background()
	receipts, err := client.SendBatch(ctx, messages)
	if err != nil {
		return err
	}

	logger.Success("批量Webhook发送成功! 共发送 %d 个Webhook", len(receipts))
	return nil
}

// sendMultiTargetWebhook demonstrates sending to multiple webhook targets
func sendMultiTargetWebhook(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送多目标Webhook")

	msg := message.New()
	msg.Title = "🌐 多目标Webhook测试"
	msg.Body = "这条消息将同时发送到多个Webhook端点。"
	msg.Format = message.FormatText

	// Configure multiple webhook targets
	msg.Targets = []target.Target{
		common.CreateWebhookTarget("https://httpbin.org/post"),
		common.CreateWebhookTarget("https://webhook.site/unique-id-1"), // 示例URL
		common.CreateWebhookTarget("https://webhook.site/unique-id-2"), // 示例URL
	}

	// Add multi-target specific data
	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"broadcast":     true,
			"target_count":  len(msg.Targets),
			"fan_out":       true,
			"delivery_mode": "parallel",
		},
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("多目标Webhook发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendAsyncWebhook demonstrates asynchronous webhook sending
func sendAsyncWebhook(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("异步Webhook发送")

	msg := common.CreateTestMessage("Webhook", "basic")
	msg.Title = "⚡ 异步Webhook发送测试"
	msg.Body = "这是一个异步方式发送的Webhook。\n\n异步功能已经实现，Webhook将在后台处理。"
	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	logger.Info("🚀 使用异步模式发送Webhook")

	ctx := context.Background()
	handle, err := client.SendAsync(ctx, msg)
	if err != nil {
		return err
	}

	logger.Info("异步Webhook已提交，消息ID: %s", handle.ID())

	// Wait for the result
	receipt, err := handle.Wait(ctx)
	if err != nil {
		logger.Error("异步Webhook发送失败: %v", err)
		return err
	}

	logger.Success("异步Webhook发送完成!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendTemplatedWebhook demonstrates templated webhook payload
func sendTemplatedWebhook(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送模板化Webhook")

	// Template variables
	templateVars := map[string]interface{}{
		"event_type":   "user_action",
		"user_id":      "user_12345",
		"action":       "purchase_completed",
		"timestamp":    time.Now().Unix(),
		"amount":       299.99,
		"currency":     "USD",
		"order_id":     "ORDER-2023-001",
		"product_name": "NotifyHub Pro License",
	}

	msg := message.New()
	msg.Title = "💰 购买完成事件"
	msg.Body = "用户购买事件Webhook通知"
	msg.Format = message.FormatText
	msg.Variables = templateVars

	// Template-based platform data
	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"event_type": templateVars["event_type"],
			"timestamp":  templateVars["timestamp"],
			"data": map[string]interface{}{
				"user_id":      templateVars["user_id"],
				"action":       templateVars["action"],
				"order_id":     templateVars["order_id"],
				"product_name": templateVars["product_name"],
				"payment": map[string]interface{}{
					"amount":   templateVars["amount"],
					"currency": templateVars["currency"],
					"status":   "completed",
				},
				"metadata": map[string]interface{}{
					"source":    "notifyhub",
					"version":   "1.0",
					"processed": time.Now().Format(time.RFC3339),
				},
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("模板化Webhook发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendWebhookWithRetry demonstrates webhook with retry configuration
func sendWebhookWithRetry(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送带重试的Webhook")

	msg := message.New()
	msg.Title = "🔄 重试机制测试"
	msg.Body = "这个Webhook演示了重试机制的使用。"
	msg.Format = message.FormatText
	msg.Priority = message.PriorityHigh

	// Configure retry behavior
	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"retry_config": map[string]interface{}{
				"max_retries":    3,
				"retry_delay":    "2s",
				"backoff_factor": 2.0,
				"timeout":        "30s",
				"retry_on_codes": []int{429, 500, 502, 503, 504},
			},
			"failure_handling": map[string]interface{}{
				"dead_letter_queue": true,
				"alert_on_failure":  true,
				"fallback_url":      "https://backup-webhook.example.com",
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("带重试的Webhook发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendEventStreamWebhook demonstrates event streaming webhook
func sendEventStreamWebhook(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送事件流Webhook")

	msg := message.New()
	msg.Title = "📊 事件流数据"
	msg.Body = "这是一个事件流Webhook，包含流式数据。"
	msg.Format = message.FormatText

	// Event stream data
	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"stream_type": "event_stream",
			"batch_size":  100,
			"sequence_id": 12345,
			"stream_events": []map[string]interface{}{
				{
					"event_id":   "evt_001",
					"type":       "page_view",
					"timestamp":  time.Now().Unix(),
					"user_id":    "user_001",
					"page":       "/dashboard",
					"session_id": "sess_abc123",
				},
				{
					"event_id":   "evt_002",
					"type":       "button_click",
					"timestamp":  time.Now().Unix() + 5,
					"user_id":    "user_001",
					"element":    "export_button",
					"session_id": "sess_abc123",
				},
				{
					"event_id":  "evt_003",
					"type":      "api_call",
					"timestamp": time.Now().Unix() + 10,
					"user_id":   "user_001",
					"endpoint":  "/api/v1/export",
					"method":    "POST",
					"status":    200,
				},
			},
			"aggregations": map[string]interface{}{
				"total_events": 3,
				"unique_users": 1,
				"event_types":  []string{"page_view", "button_click", "api_call"},
				"time_window":  "15s",
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("事件流Webhook发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendSignedWebhook demonstrates webhook with signature verification
func sendSignedWebhook(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送签名验证Webhook")

	msg := message.New()
	msg.Title = "🔐 签名验证Webhook"
	msg.Body = "这个Webhook包含签名验证信息，确保数据完整性。"
	msg.Format = message.FormatText

	// Signature configuration
	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"signature": map[string]interface{}{
				"algorithm":         "hmac-sha256",
				"secret_key":        "your_webhook_secret",
				"header_name":       "X-Signature-256",
				"include_timestamp": true,
				"timestamp_header":  "X-Timestamp",
			},
			"security": map[string]interface{}{
				"verify_ssl":    true,
				"require_https": true,
				"ip_whitelist":  []string{"192.168.1.0/24", "10.0.0.0/8"},
			},
			"payload": map[string]interface{}{
				"event":     "webhook_test",
				"timestamp": time.Now().Unix(),
				"data":      "sensitive_data_here",
				"checksum":  "calculated_checksum",
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("签名验证Webhook发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// Helper functions for creating different types of webhook messages

func createOrderWebhook(orderID, status string) *message.Message {
	msg := message.New()
	msg.Title = fmt.Sprintf("订单事件: %s", orderID)
	msg.Body = fmt.Sprintf("订单 %s 状态变更为: %s", orderID, status)
	msg.Format = message.FormatText
	msg.Priority = message.PriorityNormal

	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"event_type": "order_status_changed",
			"order_id":   orderID,
			"status":     status,
			"timestamp":  time.Now().Unix(),
			"data": map[string]interface{}{
				"order_id":   orderID,
				"new_status": status,
				"changed_at": time.Now().Format(time.RFC3339),
				"changed_by": "system",
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateWebhookTarget("https://httpbin.org/post"),
	}
	msg.Metadata = map[string]interface{}{
		"type":     "order_event",
		"order_id": orderID,
		"status":   status,
	}
	return msg
}
