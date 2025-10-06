// 简化版SMS平台演示 - 只需要10行核心代码！
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/kart-io/notifyhub/pkg/external"
	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/platform"
	"github.com/kart-io/notifyhub/pkg/target"
)

// 🎯 核心实现：只需要这一个方法！
type SMSSender struct{}

func (s *SMSSender) Send(ctx context.Context, message, target string) error {
	// 这里是实际的SMS发送逻辑
	fmt.Printf("📱 发送短信到 %s: %s\n", target, message)

	// 模拟失败场景
	if strings.Contains(target, "fail") {
		return fmt.Errorf("SMS发送失败")
	}

	return nil
}

// 📞 手机号验证器（可选）
func validatePhone(phone string) error {
	if len(phone) < 10 {
		return fmt.Errorf("手机号太短")
	}
	return nil
}

// 📝 消息格式化器（可选）
func formatMessage(msg *message.Message) string {
	if msg.Title != "" {
		return fmt.Sprintf("【%s】%s", msg.Title, msg.Body)
	}
	return msg.Body
}

func main() {
	fmt.Println("🚀 简化版SMS平台演示")
	fmt.Println("==================")

	// ✨ 使用简化的构建器创建SMS平台 - 仅需一行！
	platform := external.NewPlatform("sms", &SMSSender{}).
		WithTargetTypes("phone", "mobile").
		WithMaxMessageSize(70).
		WithRateLimit(10, 100).
		WithTemplates(map[string]string{
			"验证码": "您的验证码是{{code}}，有效期{{minutes}}分钟",
			"欢迎":  "欢迎{{name}}使用我们的服务！",
		}).
		WithTargetValidator(validatePhone).
		WithMessageFormatter(formatMessage).
		Build()

	fmt.Printf("✅ SMS平台创建成功: %s\n", platform.Name())

	// 🔍 显示平台能力
	caps := platform.GetCapabilities()
	fmt.Printf("📋 支持的目标类型: %v\n", caps.SupportedTargetTypes)
	fmt.Printf("📋 最大消息长度: %d字符\n", caps.MaxMessageSize)

	ctx := context.Background()

	// 📤 演示1：基础短信发送
	fmt.Println("\n📤 演示1：基础短信发送")
	testBasicSMS(ctx, platform)

	// 📋 演示2：模板短信发送
	fmt.Println("\n📋 演示2：模板短信发送")
	testTemplateSMS(ctx, platform)

	// 🚦 演示3：限流测试
	fmt.Println("\n🚦 演示3：限流测试")
	testRateLimit(ctx, platform)

	// ❌ 演示4：错误处理
	fmt.Println("\n❌ 演示4：错误处理")
	testErrorHandling(ctx, platform)

	fmt.Println("\n🎉 所有演示完成！")
}

func testBasicSMS(ctx context.Context, platform platform.Platform) {
	msg := message.New()
	msg.Title = "NotifyHub"
	msg.Body = "这是一条测试短信"

	targets := []target.Target{
		external.CreateTarget("phone", "+86 138 0013 8000"),
		external.CreateTarget("mobile", "+1 555 123 4567"),
	}

	results, err := platform.Send(ctx, msg, targets)
	if err != nil {
		log.Printf("发送失败: %v", err)
		return
	}

	for i, result := range results {
		if result.Success {
			fmt.Printf("  ✅ 目标%d: 发送成功\n", i+1)
		} else {
			fmt.Printf("  ❌ 目标%d: 发送失败 - %v\n", i+1, result.Error)
		}
	}
}

func testTemplateSMS(ctx context.Context, platform platform.Platform) {
	// 验证码短信
	msg1 := message.New()
	msg1.Variables = map[string]interface{}{
		"code":    "123456",
		"minutes": "5",
	}
	msg1.Metadata = map[string]interface{}{
		"template": "验证码",
	}

	targets := []target.Target{
		external.CreateTarget("phone", "+86 138 0013 8000"),
	}

	results, _ := platform.Send(ctx, msg1, targets)
	if len(results) > 0 && results[0].Success {
		fmt.Println("  ✅ 验证码短信发送成功")
	}

	// 欢迎短信
	msg2 := message.New()
	msg2.Variables = map[string]interface{}{
		"name": "张三",
	}
	msg2.Metadata = map[string]interface{}{
		"template": "欢迎",
	}

	results, _ = platform.Send(ctx, msg2, targets)
	if len(results) > 0 && results[0].Success {
		fmt.Println("  ✅ 欢迎短信发送成功")
	}
}

func testRateLimit(ctx context.Context, platform platform.Platform) {
	msg := message.New()
	msg.Body = "限流测试短信"

	target := external.CreateTarget("phone", "+86 138 0013 8000")

	successCount := 0
	failCount := 0

	// 尝试发送15条短信（限制是10条/小时）
	for i := 1; i <= 15; i++ {
		results, _ := platform.Send(ctx, msg, []target.Target{target})
		if len(results) > 0 && results[0].Success {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Printf("  📊 发送统计: 成功%d条, 被限流%d条\n", successCount, failCount)
}

func testErrorHandling(ctx context.Context, platform platform.Platform) {
	msg := message.New()
	msg.Body = "错误测试"

	// 测试无效手机号
	invalidTarget := external.CreateTarget("phone", "123")
	results, _ := platform.Send(ctx, msg, []target.Target{invalidTarget})
	if len(results) > 0 && results[0].Error != nil {
		fmt.Printf("  ✅ 无效手机号被正确拒绝: %v\n", results[0].Error)
	}

	// 测试发送失败
	failTarget := external.CreateTarget("phone", "+86 138 0013 fail")
	results, _ = platform.Send(ctx, msg, []target.Target{failTarget})
	if len(results) > 0 && results[0].Error != nil {
		fmt.Printf("  ✅ 发送失败被正确处理: %v\n", results[0].Error)
	}
}
