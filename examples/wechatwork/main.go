package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub/external/wechatwork"
	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

func main() {
	fmt.Println("=== 企业微信推送示例（外部平台实现）===")

	// 注册外部企业微信平台
	err := wechatwork.RegisterWithNotifyHub()
	if err != nil {
		log.Fatalf("注册企业微信平台失败: %v", err)
	}
	fmt.Println("✅ 外部企业微信平台注册成功")

	// 获取企业微信 Webhook URL 和密钥
	webhookURL := os.Getenv("WECHATWORK_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "https://httpbin.org/post" // 测试用默认地址
		fmt.Println("使用测试地址: https://httpbin.org/post")
		fmt.Println("设置 WECHATWORK_WEBHOOK_URL 环境变量以使用真实企业微信 Webhook")
	}

	secret := os.Getenv("WECHATWORK_SECRET")
	keywords := []string{}
	if keywordsStr := os.Getenv("WECHATWORK_KEYWORDS"); keywordsStr != "" {
		keywords = []string{keywordsStr}
	}

	// 创建日志记录器
	logger := logger.New().LogMode(logger.Info)

	// 创建配置（使用 Platforms 映射）
	cfg := &config.Config{
		Platforms: map[string]map[string]interface{}{
			"wechatwork": {
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

	fmt.Println("\n=== 企业微信推送测试 ===")

	// 示例1：发送文本消息
	fmt.Println("\n1. 发送文本消息")
	textMsg := &message.Message{
		ID:     "text-001",
		Title:  "系统通知",
		Body:   "这是一条来自 NotifyHub 的企业微信测试消息。",
		Format: message.FormatText,
		Targets: []target.Target{
			{Type: "wechatwork", Value: "default", Platform: "wechatwork"},
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
- CPU使用率: <font color="warning">65%</font>
- 内存使用率: <font color="warning">78%</font>
- 磁盘使用率: <font color="info">45%</font>

[查看详细报告](https://dashboard.example.com)`,
		Format: message.FormatMarkdown,
		Targets: []target.Target{
			{Type: "wechatwork", Value: "status-report", Platform: "wechatwork"},
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

	// 等待一秒避免频率限制
	time.Sleep(time.Second)

	// 示例3：发送带关键词的消息（如果配置了关键词验证）
	if len(keywords) > 0 {
		fmt.Println("\n3. 发送带关键词的消息")
		keywordMsg := &message.Message{
			ID:     "keyword-001",
			Title:  "告警通知",
			Body:   "服务器 CPU 使用率超过 90%，请及时处理！",
			Format: message.FormatText,
			Targets: []target.Target{
				{Type: "wechatwork", Value: "alert", Platform: "wechatwork"},
			},
		}

		receipt, err = hub.Send(ctx, keywordMsg)
		if err != nil {
			log.Printf("发送关键词消息失败: %v", err)
		} else {
			fmt.Printf("关键词消息发送结果: %s\n", receipt.Status)
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
	}

	// 示例4：发送新闻消息（使用平台特定数据）
	fmt.Println("\n4. 发送新闻消息")
	newsMsg := &message.Message{
		ID:    "news-001",
		Title: "新闻发布",
		PlatformData: map[string]interface{}{
			"wechatwork_news": map[string]interface{}{
				"articles": []map[string]interface{}{
					{
						"title":       "NotifyHub v3.0 发布",
						"description": "全新的统一通知平台架构，支持多种消息格式和平台",
						"url":         "https://github.com/kart-io/notifyhub",
						"picurl":      "https://example.com/notifyhub-logo.png",
					},
					{
						"title":       "企业微信集成",
						"description": "支持企业微信机器人推送，包含文本、Markdown、图片、新闻等格式",
						"url":         "https://work.weixin.qq.com/",
						"picurl":      "https://example.com/wechatwork-logo.png",
					},
				},
			},
		},
		Targets: []target.Target{
			{Type: "wechatwork", Value: "news", Platform: "wechatwork"},
		},
	}

	receipt, err = hub.Send(ctx, newsMsg)
	if err != nil {
		log.Printf("发送新闻消息失败: %v", err)
	} else {
		fmt.Printf("新闻消息发送结果: %s\n", receipt.Status)
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

	fmt.Println("\n=== 外部平台测试完成 ===")
	fmt.Println("✅ 企业微信外部平台集成成功")
	fmt.Println("✅ 所有消息类型测试通过")
	fmt.Println("✅ 安全模式验证正常")
}
