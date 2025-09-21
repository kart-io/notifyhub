package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/kart-io/notifyhub"

	"github.com/kart-io/notifyhub/platforms/email"
	"github.com/kart-io/notifyhub/platforms/feishu"
	"github.com/kart-io/notifyhub/platforms/registry"
)

func main() {
	// 解析命令行参数
	var demoType = flag.String("demo", "basic", "演示类型: basic, advanced, all")
	flag.Parse()

	switch *demoType {
	case "basic":
		basicDemo()
	case "advanced":
		advancedDemo()
	case "all":
		log.Println("运行所有演示...")
		basicDemo()
		advancedDemo()
	default:
		log.Printf("未知的演示类型: %s", *demoType)
		log.Println("可用选项: basic, advanced, all")
	}
}

func init() {
	// Register built-in platforms
	if err := registry.Register(feishu.NewFeishuPlatform()); err != nil {
		log.Fatalf("failed to register feishu platform: %v", err)
	}
	if err := registry.Register(email.NewEmailPlatform()); err != nil {
		log.Fatalf("failed to register email platform: %v", err)
	}
}

// 基础配置演示
func basicDemo() {
	log.Println("=== 基础配置演示 ===")
	// 使用环境变量配置，或使用默认测试值
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	secret := os.Getenv("FEISHU_SECRET")

	// 如果没有设置环境变量，使用示例中的测试值
	if webhookURL == "" {
		webhookURL = ""
	}
	if secret == "" {
		secret = ""
	}

	log.Printf("Using webhook URL: %s", webhookURL)
	log.Printf("Using secret: %s", secret)

	client, err := notifyhub.New(
		notifyhub.WithFeishu(webhookURL, secret),
	)
	if err != nil {
		log.Printf("❌ 客户端创建失败: %v", err)
		log.Printf("\n📋 故障排除:")
		log.Printf("1. 确认webhook URL是否有效")
		log.Printf("2. 确认secret是否正确")
		log.Printf("3. 确认飞书机器人是否已启用")
		log.Printf("4. 尝试设置环境变量:")
		log.Printf("   export FEISHU_WEBHOOK_URL=\"your-webhook-url\"")
		log.Printf("   export FEISHU_SECRET=\"your-secret\"")
		return
	}

	ctx := context.Background()
	result, err := client.Send(ctx).
		Title("Hello World").
		Body("This is a test notification").
		ToFeishu("webhook-id").
		Execute()

	if err != nil {
		log.Printf("❌ 发送失败: %v", err)
		return
	}

	log.Printf("✅ 消息发送成功! 结果: %+v", result)
}

// 高级配置演示 - 展示四种安全设置方式
func advancedDemo() {
	webhookURL := ""

	// 运行四种安全模式演示
	demoSignatureMode()
	demoIPWhitelistProxyMode(webhookURL)
	demoKeywordsMode(webhookURL)
	demoNoneMode(webhookURL)
}

// 演示1：签名校验模式
func demoSignatureMode() {

	log.Println("\n=== 高级配置演示（四选一安全设置） ===")

	webhookURL := getEnvOrDefault("FEISHU_WEBHOOK_URL", "")
	secret := getEnvOrDefault("FEISHU_SECRET", "")

	if webhookURL == "" {
		log.Println("跳过高级演示：需要设置 FEISHU_WEBHOOK_URL 环境变量")
		return
	}

	if secret == "" {
		log.Println("\n--- 签名校验模式 ---")
		log.Println("跳过签名校验演示：需要设置 FEISHU_SECRET 环境变量")
		return
	}

	log.Println("\n--- 签名校验模式 ---", webhookURL, secret)
	client, err := notifyhub.New(
		notifyhub.WithFeishuSignature(webhookURL, secret),
	)
	if err != nil {
		log.Printf("创建签名校验客户端失败: %v", err)
	} else {
		testSend(client, "🔐 签名校验", "这是使用签名校验的消息", "signature-test")
	}
}

// 演示2b：IP白名单代理模式
func demoIPWhitelistProxyMode(webhookURL string) {
	log.Println("\n--- IP白名单代理模式 ---")
	log.Println("注意：IP白名单需要在飞书机器人后台预先配置（配置代理服务器的IP）")

	// 从环境变量获取代理配置（支持大小写）
	proxyURL := getHTTPProxy()
	if proxyURL == "" {
		// 如果没有设置代理环境变量，使用默认测试代理
		proxyURL = "http://127.0.0.1:7890"
		log.Printf("未设置代理环境变量，使用默认代理: %s", proxyURL)
	} else {
		log.Printf("使用环境变量代理: %s", proxyURL)
	}

	client, err := notifyhub.New(
		notifyhub.WithFeishuIPWhitelistProxy(webhookURL, proxyURL),
	)
	if err != nil {
		log.Printf("创建IP白名单代理客户端失败: %v", err)
	} else {
		testSend(client, "🌐🔗 IP白名单+代理", "这是通过代理使用IP白名单验证的消息", "ip-whitelist-proxy-test")
	}
}

// 演示3：自定义关键词模式
func demoKeywordsMode(webhookURL string) {
	log.Println("\n--- 自定义关键词模式 ---")
	customKeywords := []string{
		"notification",
	}
	client, err := notifyhub.New(
		notifyhub.WithFeishuKeywords(webhookURL, customKeywords),
	)
	if err != nil {
		log.Printf("创建关键词客户端失败: %v", err)
	} else {
		testSend(client, "🏷️ 关键词模式", "这是使用自定义关键词的消息", "keywords-test")
	}
}

// 演示4：无安全验证模式
func demoNoneMode(webhookURL string) {
	log.Println("\n--- 无安全验证模式 ---")
	client, err := notifyhub.New(
		notifyhub.WithFeishuNone(webhookURL),
	)
	if err != nil {
		log.Printf("创建无验证客户端失败: %v", err)
	} else {
		testSend(client, "🚫 无验证", "这是无安全验证的消息", "none-test")
	}
}

// 辅助测试函数
func testSend(client *notifyhub.Client, title, body, target string) {
	ctx := context.Background()
	result, err := client.Send(ctx).
		Title(title).
		Body(body).
		ToFeishu(target).
		Execute()

	if err != nil {
		log.Printf("❌ %s 发送失败: %v", title, err)
	} else {
		log.Printf("✅ %s 发送成功: %s", title, result.MessageID)
	}
}

// 辅助函数：获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 获取HTTP代理环境变量（支持大小写）
func getHTTPProxy() string {
	// 按标准优先级检查环境变量
	proxies := []string{"HTTP_PROXY", "http_proxy"}
	for _, proxy := range proxies {
		if value := os.Getenv(proxy); value != "" {
			return value
		}
	}
	return ""
}
