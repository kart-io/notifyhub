// Package main demonstrates advanced feishu features with NotifyHub
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
	common.PrintHeader("Feishu", "Advanced")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请修改以下配置为您的实际飞书信息
	config.Feishu.WebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxx-xxxx-xxxx-xxxxxxxxxxxxxxxxxxxx"
	config.Feishu.Secret = ""                       // 可选，飞书机器人签名校验
	config.Feishu.Keywords = []string{"111", "222"} // 可选，关键词设置

	// Check configuration
	if !common.CheckConfigurationPrompt("feishu") {
		logger.Info("已取消运行")
		return
	}

	// Validate configuration
	if err := common.ValidateConfig(config, "feishu"); err != nil {
		logger.Error("配置验证失败: %v", err)
		return
	}

	// Create NotifyHub client with async support
	cfg := config.CreateFeishuConfig()
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
		{"批量消息发送", sendBatchMessages},
		{"富文本卡片消息", sendRichCardMessage},
		{"异步消息发送", sendAsyncMessage},
		{"模板化消息", sendTemplatedMessage},
		{"带元数据的消息", sendMessageWithMetadata},
		{"@用户消息", sendMentionMessage},
	}

	for _, example := range examples {
		if err := example.fn(client, config, logger); err != nil {
			logger.Error("%s失败: %v", example.name, err)
		}

		// 添加延迟避免频率限制
		time.Sleep(2 * time.Second)
	}

	logger.Success("所有高级飞书示例运行完成!")
}

// sendBatchMessages demonstrates batch message sending
func sendBatchMessages(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("批量消息发送")

	messages := []*message.Message{
		createAlertMessage(),
		createReportMessage(),
		createReminderMessage(),
	}

	ctx := context.Background()
	receipts, err := client.SendBatch(ctx, messages)
	if err != nil {
		return err
	}

	logger.Success("批量消息发送成功! 共发送 %d 条消息", len(receipts))
	return nil
}

// sendRichCardMessage demonstrates rich card message
func sendRichCardMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送富文本卡片消息")

	msg := message.New()
	msg.Title = "📊 系统监控报告"
	msg.Body = buildRichCardContent()
	msg.Format = message.FormatText
	msg.Priority = message.PriorityNormal

	// Add feishu-specific card data
	msg.PlatformData = map[string]interface{}{
		"feishu": map[string]interface{}{
			"msg_type": "interactive",
			"card": map[string]interface{}{
				"config": map[string]interface{}{
					"wide_screen_mode": true,
				},
				"header": map[string]interface{}{
					"title": map[string]interface{}{
						"tag":     "plain_text",
						"content": "📊 系统监控报告",
					},
					"template": "blue",
				},
				"elements": []map[string]interface{}{
					{
						"tag": "div",
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": "**服务状态**: 🟢 正常运行\n**CPU使用率**: 45%\n**内存使用率**: 62%\n**磁盘使用率**: 78%",
						},
					},
					{
						"tag": "action",
						"actions": []map[string]interface{}{
							{
								"tag": "button",
								"text": map[string]interface{}{
									"tag":     "plain_text",
									"content": "查看详情",
								},
								"type": "primary",
								"url":  "https://monitor.example.com",
							},
						},
					},
				},
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("富文本卡片消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendAsyncMessage demonstrates asynchronous message sending
func sendAsyncMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("异步消息发送")

	msg := common.CreateTestMessage("Feishu", "basic")
	msg.Title = "⚡ 异步消息发送测试"
	msg.Body = "这是一条异步方式发送的飞书消息。\n\n异步功能已经实现，消息将在后台处理。"
	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
	}

	logger.Info("🚀 使用异步模式发送消息")

	ctx := context.Background()
	handle, err := client.SendAsync(ctx, msg)
	if err != nil {
		return err
	}

	logger.Info("异步消息已提交，消息ID: %s", handle.ID())

	// Wait for the result
	receipt, err := handle.Wait(ctx)
	if err != nil {
		logger.Error("异步消息发送失败: %v", err)
		return err
	}

	logger.Success("异步消息发送完成!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendTemplatedMessage demonstrates templated message
func sendTemplatedMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送模板化消息")

	// Template variables
	templateVars := map[string]interface{}{
		"user_name":    "张三",
		"project_name": "NotifyHub",
		"deploy_time":  time.Now().Format("2006-01-02 15:04:05"),
		"version":      "v1.2.0",
		"environment":  "生产环境",
		"success_rate": "99.9%",
	}

	msg := message.New()
	msg.Title = "🚀 部署通知"
	msg.Body = buildDeploymentNotificationTemplate(templateVars)
	msg.Format = message.FormatMarkdown
	msg.Variables = templateVars

	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("模板化消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendMessageWithMetadata demonstrates message with rich metadata
func sendMessageWithMetadata(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送带元数据的消息")

	msg := message.New()
	msg.Title = "📊 带元数据的飞书消息测试"
	msg.Body = `这条消息展示了如何使用丰富的元数据。

元数据用途：
- 消息分类和跟踪
- 统计分析
- 自动化处理
- 业务标识

消息ID: ` + msg.ID
	msg.Format = message.FormatText

	// Rich metadata
	msg.Metadata = map[string]interface{}{
		"campaign_id":    "metadata_demo_001",
		"sender_dept":    "技术部",
		"category":       "测试",
		"message_type":   "notification",
		"priority_score": 85,
		"tags":           []string{"demo", "metadata", "advanced"},
		"business_data": map[string]interface{}{
			"order_id":    "ORD-2023-001",
			"customer_id": "CUST-001",
			"amount":      1299.99,
			"currency":    "CNY",
		},
	}

	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("带元数据的消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	logger.Debug("元数据: %+v", msg.Metadata)
	return nil
}

// sendMentionMessage demonstrates @user mention message
func sendMentionMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送@用户消息")

	msg := message.New()
	msg.Title = "📢 重要通知"
	msg.Body = "<at user_id=\"all\">所有人</at> 请注意：\n\n系统将在今晚23:00进行维护升级，预计耗时2小时。\n\n维护期间服务将暂时不可用，请提前做好准备。\n\n如有紧急问题，请联系值班人员。"
	msg.Format = message.FormatText
	msg.Priority = message.PriorityHigh

	// Add feishu-specific mention data
	msg.PlatformData = map[string]interface{}{
		"feishu": map[string]interface{}{
			"mentions": []map[string]interface{}{
				{
					"key":        "all",
					"id":         "all",
					"id_type":    "user_id",
					"name":       "所有人",
					"tenant_key": "",
				},
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("@用户消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// Helper functions for creating different types of messages

func createAlertMessage() *message.Message {
	msg := message.New()
	msg.Title = "🚨 系统警告"
	msg.Body = "检测到CPU使用率超过90%，请及时处理。"
	msg.Format = message.FormatText
	msg.Priority = message.PriorityUrgent
	msg.Targets = []target.Target{common.CreateFeishuTarget()}
	msg.Metadata = map[string]interface{}{"type": "alert"}
	return msg
}

func createReportMessage() *message.Message {
	msg := message.New()
	msg.Title = "📊 日报"
	msg.Body = `# 今日工作总结

## 完成任务
- 修复了3个bug
- 完成了2个新功能
- 代码审查5个PR

## 明日计划
- 继续开发新模块
- 优化系统性能
- 编写技术文档

---
*由NotifyHub自动生成*`
	msg.Format = message.FormatMarkdown
	msg.Targets = []target.Target{common.CreateFeishuTarget()}
	msg.Metadata = map[string]interface{}{"type": "report"}
	return msg
}

func createReminderMessage() *message.Message {
	msg := message.New()
	msg.Title = "⏰ 会议提醒"
	msg.Body = "您有一个会议将在30分钟后开始：技术评审会议"
	msg.Format = message.FormatText
	msg.Priority = message.PriorityNormal
	msg.Targets = []target.Target{common.CreateFeishuTarget()}
	msg.Metadata = map[string]interface{}{"type": "reminder"}
	return msg
}

func buildRichCardContent() string {
	return `系统状态报告 - ` + time.Now().Format("2006-01-02 15:04:05") + `

🟢 服务状态: 正常运行
📊 CPU使用率: 45%
💾 内存使用率: 62%
💿 磁盘使用率: 78%
🌐 网络延迟: 12ms

详细信息请查看监控面板。`
}

func buildDeploymentNotificationTemplate(vars map[string]interface{}) string {
	return fmt.Sprintf(`# 🚀 部署通知

**项目**: %s
**版本**: %s
**环境**: %s
**部署时间**: %s
**操作人员**: %s

## 部署结果
- ✅ 部署成功
- ✅ 健康检查通过
- ✅ 成功率: %s

## 版本说明
本次发布包含了性能优化和bug修复，预期将提升系统稳定性。

---
*此通知由NotifyHub自动发送*`,
		vars["project_name"], vars["version"], vars["environment"],
		vars["deploy_time"], vars["user_name"], vars["success_rate"])
}
