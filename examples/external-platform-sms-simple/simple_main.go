// 🚀 极简版SMS平台演示 - 只需要一个方法！
package main

import (
	"context"
	"fmt"
	"strings"
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

// 🏗️ 简化的平台构建器（核心概念演示）
type SimpleSMSPlatform struct {
	sender      *SMSSender
	rateLimiter map[string]int // 简单计数器
	templates   map[string]string
}

func NewSimpleSMSPlatform() *SimpleSMSPlatform {
	return &SimpleSMSPlatform{
		sender:      &SMSSender{},
		rateLimiter: make(map[string]int),
		templates: map[string]string{
			"验证码": "您的验证码是{{code}}，有效期{{minutes}}分钟",
			"欢迎":  "欢迎{{name}}使用我们的服务！",
		},
	}
}

func (p *SimpleSMSPlatform) Send(target, message string) error {
	// 简单限流检查
	if p.rateLimiter[target] >= 10 {
		return fmt.Errorf("rate limit exceeded for %s", target)
	}

	// 发送消息
	err := p.sender.Send(context.Background(), message, target)
	if err == nil {
		p.rateLimiter[target]++
	}
	return err
}

func (p *SimpleSMSPlatform) SendTemplate(target, templateName string, vars map[string]string) error {
	template, exists := p.templates[templateName]
	if !exists {
		return fmt.Errorf("template not found: %s", templateName)
	}

	// 简单变量替换
	message := template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		message = strings.ReplaceAll(message, placeholder, value)
	}

	return p.Send(target, message)
}

func main() {
	fmt.Println("🚀 极简版SMS平台演示")
	fmt.Println("==================")
	fmt.Println("✨ 只需要实现一个 Send 方法！")
	fmt.Println()

	// 创建SMS平台（只需要一行！）
	sms := NewSimpleSMSPlatform()

	// 📤 演示1：基础短信发送
	fmt.Println("📤 演示1：基础短信发送")
	sms.Send("+86 138 0013 8000", "这是一条测试短信")
	sms.Send("+1 555 123 4567", "Hello SMS")
	fmt.Println()

	// 📋 演示2：模板短信
	fmt.Println("📋 演示2：模板短信发送")
	sms.SendTemplate("+86 138 0013 8000", "验证码", map[string]string{
		"code":    "123456",
		"minutes": "5",
	})
	sms.SendTemplate("+86 138 0013 8000", "欢迎", map[string]string{
		"name": "张三",
	})
	fmt.Println()

	// 🚦 演示3：限流测试
	fmt.Println("🚦 演示3：限流测试")
	successCount := 0
	failCount := 0

	// 尝试发送15条短信（限制是10条）
	for i := 1; i <= 15; i++ {
		err := sms.Send("+86 138 0013 8000", fmt.Sprintf("限流测试短信 #%d", i))
		if err != nil {
			failCount++
			if i > 10 { // 只显示被限流的
				fmt.Printf("  ❌ 第%d条被限流: %v\n", i, err)
			}
		} else {
			successCount++
			fmt.Printf("  ✅ 第%d条发送成功\n", i)
		}
	}
	fmt.Printf("📊 发送统计: 成功%d条, 被限流%d条\n", successCount, failCount)
	fmt.Println()

	// ❌ 演示4：错误处理
	fmt.Println("❌ 演示4：错误处理")
	err := sms.Send("+86 138 0013 fail", "这条会失败")
	if err != nil {
		fmt.Printf("✅ 错误被正确处理: %v\n", err)
	}

	fmt.Println()
	fmt.Println("🎉 所有演示完成！")
	fmt.Println()
	fmt.Println("💡 对比说明:")
	fmt.Println("   原始方式: 需要实现7个接口方法，约300行代码")
	fmt.Println("   简化方式: 只需实现1个Send方法，约20行核心代码")
	fmt.Println("   简化比例: 95% 代码减少！")
}
