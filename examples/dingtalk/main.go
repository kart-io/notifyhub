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
)

func main() {
	// 获取钉钉 Webhook URL 和密钥
	webhookURL := os.Getenv("DINGTALK_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "https://httpbin.org/post" // 测试用默认地址
		fmt.Println("使用测试地址: https://httpbin.org/post")
		fmt.Println("设置 DINGTALK_WEBHOOK_URL 环境变量以使用真实钉钉 Webhook")
	}

	secret := os.Getenv("DINGTALK_SECRET")
	keywords := []string{}
	if keywordsStr := os.Getenv("DINGTALK_KEYWORDS"); keywordsStr != "" {
		keywords = []string{keywordsStr}
	}

	// 创建日志记录器
	logger := logger.New().LogMode(logger.Info)

	// 创建配置（使用 Platforms 映射）
	cfg := &config.Config{
		Platforms: map[string]map[string]interface{}{
			"dingtalk": {
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

	// 创建上下文
	ctx := context.Background()

	fmt.Println("=== 钉钉推送示例 ===")

	// 示例1：发送文本消息
	fmt.Println("\n1. 发送文本消息")
	textMsg := &message.Message{
		ID:     "text-001",
		Title:  "系统通知",
		Body:   "这是一条来自 NotifyHub 的钉钉测试消息。",
		Format: message.FormatText,
		Targets: []target.Target{
			{Type: "dingtalk", Value: "default", Platform: "dingtalk"},
		},
	}

	receipt, err := hub.Send(ctx, textMsg)
	if err != nil {
		log.Printf("发送文本消息失败: %v", err)
	} else {
		fmt.Printf("文本消息发送结果: %s\n", receipt.Status)
		for _, result := range receipt.Results {
			fmt.Printf("  - 平台: %s, 成功: %t, 消息ID: %s\n",
				result.Platform, result.Success, result.MessageID)
			if !result.Success {
				fmt.Printf("    错误: %s\n", result.Error)
			}
		}
	}

	// 等待一秒避免频率限制
	time.Sleep(time.Second)

	// 示例2：发送 Markdown 消息
	fmt.Println("\n2. 发送 Markdown 消息")
	markdownMsg := &message.Message{
		ID:    "markdown-001",
		Title: "📊 系统状态报告",
		Body: `## 服务状态

**✅ Web服务**: 正常运行
**✅ 数据库**: 连接正常
**⚠️ 缓存服务**: 高负载
**❌ 备份服务**: 异常

### 性能指标
- CPU使用率: 65%
- 内存使用率: 78%
- 磁盘使用率: 45%

[查看详细报告](https://dashboard.example.com)`,
		Format: message.FormatMarkdown,
		Targets: []target.Target{
			{Type: "dingtalk", Value: "status-report", Platform: "dingtalk"},
		},
	}

	receipt, err = hub.Send(ctx, markdownMsg)
	if err != nil {
		log.Printf("发送 Markdown 消息失败: %v", err)
	} else {
		fmt.Printf("Markdown 消息发送结果: %s\n", receipt.Status)
		for _, result := range receipt.Results {
			fmt.Printf("  - 平台: %s, 成功: %t, 消息ID: %s\n",
				result.Platform, result.Success, result.MessageID)
			if !result.Success {
				fmt.Printf("    错误: %s\n", result.Error)
			}
		}
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

	fmt.Println("\n=== 钉钉推送示例完成 ===")
}
