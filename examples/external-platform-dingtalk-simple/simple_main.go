// 🚀 极简版钉钉平台演示 - 只需要一个方法！
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// 🎯 核心实现：只需要这一个方法！
type DingTalkSender struct {
	webhookURL string
	client     *http.Client
}

func NewDingTalkSender(webhookURL string) *DingTalkSender {
	return &DingTalkSender{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// 钉钉消息结构
type DingTalkMessage struct {
	MsgType string `json:"msgtype"`
	Text    *Text  `json:"text,omitempty"`
	At      *At    `json:"at,omitempty"`
}

type Text struct {
	Content string `json:"content"`
}

type At struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

// 只需要实现这一个方法！
func (d *DingTalkSender) Send(ctx context.Context, message, target string) error {
	// 构建钉钉消息
	dingMsg := &DingTalkMessage{
		MsgType: "text",
		Text: &Text{
			Content: message,
		},
	}

	// 处理@功能
	if strings.Contains(target, "@") {
		if target == "@all" {
			dingMsg.At = &At{IsAtAll: true}
		} else {
			// 支持@手机号
			mobiles := strings.Split(strings.TrimPrefix(target, "@"), ",")
			dingMsg.At = &At{AtMobiles: mobiles}
		}
	}

	// 序列化消息
	msgBytes, err := json.Marshal(dingMsg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	// 发送HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, bytes.NewBuffer(msgBytes))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	fmt.Printf("📱 钉钉消息发送成功: %s -> %s\n", target, message)
	return nil
}

// 🏗️ 简化的钉钉平台（核心概念演示）
type SimpleDingTalkPlatform struct {
	sender    *DingTalkSender
	templates map[string]string
}

func NewSimpleDingTalkPlatform(webhookURL string) *SimpleDingTalkPlatform {
	return &SimpleDingTalkPlatform{
		sender: NewDingTalkSender(webhookURL),
		templates: map[string]string{
			"通知": "📢 **{{title}}**\n\n{{content}}\n\n⏰ 时间：{{time}}",
			"告警": "🚨 **{{level}}告警**\n\n**项目**: {{project}}\n**详情**: {{details}}",
			"日报": "📊 **{{date}}工作日报**\n\n✅ 已完成:\n{{completed}}\n\n🔄 进行中:\n{{ongoing}}",
		},
	}
}

func (p *SimpleDingTalkPlatform) Send(target, message string) error {
	return p.sender.Send(context.Background(), message, target)
}

func (p *SimpleDingTalkPlatform) SendTemplate(target, templateName string, vars map[string]string) error {
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
	fmt.Println("🚀 极简版钉钉平台演示")
	fmt.Println("==================")
	fmt.Println("✨ 只需要实现一个 Send 方法！")
	fmt.Println()

	// 配置钉钉webhook（这里使用示例URL，实际使用时替换为真实URL）
	webhookURL := "https://oapi.dingtalk.com/robot/send?access_token=your_token"

	// 创建钉钉平台（只需要一行！）
	dingtalk := NewSimpleDingTalkPlatform(webhookURL)

	// 📤 演示1：基础消息发送
	fmt.Println("📤 演示1：基础消息发送")
	fmt.Println("模拟发送消息（实际需要真实webhook URL）:")
	simulateSend("webhook", "这是一条来自简化版钉钉平台的测试消息")
	simulateSend("@all", "通知所有人的重要消息")
	fmt.Println()

	// 📋 演示2：模板消息
	fmt.Println("📋 演示2：模板消息发送")
	fmt.Println("模拟发送通知模板:")
	notifyMessage := strings.ReplaceAll(dingtalk.templates["通知"], "{{title}}", "系统维护通知")
	notifyMessage = strings.ReplaceAll(notifyMessage, "{{content}}", "系统将于今晚22:00-24:00进行维护")
	notifyMessage = strings.ReplaceAll(notifyMessage, "{{time}}", time.Now().Format("2006-01-02 15:04:05"))
	simulateSend("webhook", notifyMessage)

	fmt.Println("模拟发送告警模板:")
	alertMessage := strings.ReplaceAll(dingtalk.templates["告警"], "{{level}}", "严重")
	alertMessage = strings.ReplaceAll(alertMessage, "{{project}}", "NotifyHub")
	alertMessage = strings.ReplaceAll(alertMessage, "{{details}}", "数据库连接异常")
	simulateSend("@13800138000", alertMessage)

	fmt.Println("模拟发送日报模板:")
	reportMessage := strings.ReplaceAll(dingtalk.templates["日报"], "{{date}}", time.Now().Format("2006-01-02"))
	reportMessage = strings.ReplaceAll(reportMessage, "{{completed}}", "- 完成钉钉平台简化实现")
	reportMessage = strings.ReplaceAll(reportMessage, "{{ongoing}}", "- 优化用户体验")
	simulateSend("webhook", reportMessage)
	fmt.Println()

	// 📢 演示3：@功能测试
	fmt.Println("📢 演示3：@功能测试")
	simulateSend("@all", "这是一条@所有人的消息")
	simulateSend("@13800138000,13800138001", "这是一条@指定用户的消息")
	fmt.Println()

	// 🎯 演示4：实际功能展示（如果有真实webhook URL）
	fmt.Println("🎯 演示4：实际功能展示")
	fmt.Println("⚠️  注意：以下演示需要真实的钉钉webhook URL才能实际发送")
	fmt.Println("如果您有真实的webhook URL，可以替换上面的 webhookURL 变量")
	fmt.Println()

	fmt.Println("🎉 所有演示完成！")
	fmt.Println()
	fmt.Println("💡 对比说明:")
	fmt.Println("   原始方式: 需要实现7个接口方法，约200行代码")
	fmt.Println("   简化方式: 只需实现1个Send方法，约50行核心代码")
	fmt.Println("   简化比例: 75% 代码减少！")
	fmt.Println()
	fmt.Println("🔧 使用方法:")
	fmt.Println("1. 替换 webhookURL 为您的真实钉钉机器人webhook地址")
	fmt.Println("2. 调用 dingtalk.Send(target, message) 发送消息")
	fmt.Println("3. 调用 dingtalk.SendTemplate(target, template, vars) 发送模板消息")
}

// 模拟发送函数（用于演示）
func simulateSend(target, message string) {
	fmt.Printf("📱 模拟发送到 %s:\n", target)
	// 限制显示长度，避免输出过长
	displayMessage := message
	if len(displayMessage) > 100 {
		displayMessage = displayMessage[:100] + "..."
	}
	fmt.Printf("   内容: %s\n", displayMessage)
	fmt.Printf("   状态: ✅ 模拟发送成功\n")
	fmt.Println()
}
