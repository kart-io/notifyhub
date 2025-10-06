// 🚀 简化版钉钉平台演示 - 只需要实现一个方法！
package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/external"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/platform"
	"github.com/kart/notifyhub/pkg/target"
)

// 🎯 核心实现：只需要这一个方法！
type DingTalkSender struct {
	webhookURL string
	secret     string
	client     *http.Client
}

func NewDingTalkSender(webhookURL, secret string) *DingTalkSender {
	return &DingTalkSender{
		webhookURL: webhookURL,
		secret:     secret,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// 钉钉消息结构
type DingTalkMessage struct {
	MsgType  string    `json:"msgtype"`
	Text     *Text     `json:"text,omitempty"`
	Markdown *Markdown `json:"markdown,omitempty"`
	At       *At       `json:"at,omitempty"`
}

type Text struct {
	Content string `json:"content"`
}

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type At struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	AtUserIds []string `json:"atUserIds,omitempty"`
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

	// 构建请求URL（包含签名）
	requestURL := d.buildURL()

	// 发送HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(msgBytes))
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

	return nil
}

// 构建带签名的URL
func (d *DingTalkSender) buildURL() string {
	if d.secret == "" {
		return d.webhookURL
	}

	timestamp := time.Now().UnixMilli()
	sign := d.generateSign(timestamp)

	return fmt.Sprintf("%s&timestamp=%d&sign=%s", d.webhookURL, timestamp, url.QueryEscape(sign))
}

// 生成钉钉签名
func (d *DingTalkSender) generateSign(timestamp int64) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, d.secret)
	hash := hmac.New(sha256.New, []byte(d.secret))
	hash.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// 📱 钉钉目标验证器
func validateDingTalkTarget(target string) error {
	// 支持 webhook、@all、@手机号等格式
	if target == "@all" {
		return nil
	}
	if strings.HasPrefix(target, "@") {
		// 验证@手机号格式
		mobiles := strings.Split(strings.TrimPrefix(target, "@"), ",")
		for _, mobile := range mobiles {
			if len(mobile) < 11 {
				return fmt.Errorf("invalid mobile number: %s", mobile)
			}
		}
		return nil
	}
	if strings.HasPrefix(target, "webhook") {
		return nil
	}
	return fmt.Errorf("unsupported target format: %s", target)
}

// 📝 钉钉消息格式化器
func formatDingTalkMessage(msg *message.Message) string {
	content := msg.Body
	if msg.Title != "" {
		content = fmt.Sprintf("**%s**\n\n%s", msg.Title, msg.Body)
	}
	return content
}

func main() {
	fmt.Println("🚀 简化版钉钉平台演示")
	fmt.Println("==================")
	fmt.Println("✨ 只需要实现一个 Send 方法！")
	fmt.Println()

	// 配置钉钉webhook（这里使用示例URL）
	webhookURL := "https://oapi.dingtalk.com/robot/send?access_token=your_token"
	secret := "your_secret"

	// ✨ 创建钉钉发送器（核心代码）
	dingTalkSender := NewDingTalkSender(webhookURL, secret)

	// ✨ 使用简化的构建器创建钉钉平台 - 仅需一行！
	platform := external.NewPlatform("dingtalk", dingTalkSender).
		WithTargetTypes("webhook", "group", "channel").
		WithFormats("text", "markdown").
		WithMaxMessageSize(1000).
		WithTemplates(map[string]string{
			"通知": "📢 **{{title}}**\n\n{{content}}\n\n⏰ 时间：{{time}}",
			"告警": "🚨 **{{level}}告警**\n\n**项目**: {{project}}\n**详情**: {{details}}\n\n⏰ {{time}}",
			"日报": "📊 **{{date}}工作日报**\n\n✅ **已完成**:\n{{completed}}\n\n🔄 **进行中**:\n{{ongoing}}\n\n📅 **明日计划**:\n{{tomorrow}}",
		}).
		WithTargetValidator(validateDingTalkTarget).
		WithMessageFormatter(formatDingTalkMessage).
		Build()

	fmt.Printf("✅ 钉钉平台创建成功: %s\n", platform.Name())

	// 🔍 显示平台能力
	caps := platform.GetCapabilities()
	fmt.Printf("📋 支持的目标类型: %v\n", caps.SupportedTargetTypes)
	fmt.Printf("📋 支持的格式: %v\n", caps.SupportedFormats)
	fmt.Printf("📋 最大消息长度: %d字符\n", caps.MaxMessageSize)

	ctx := context.Background()

	// 📤 演示1：基础消息发送
	fmt.Println("\n📤 演示1：基础消息发送")
	testBasicMessage(ctx, platform)

	// 📋 演示2：模板消息发送
	fmt.Println("\n📋 演示2：模板消息发送")
	testTemplateMessage(ctx, platform)

	// 📢 演示3：@功能测试
	fmt.Println("\n📢 演示3：@功能测试")
	testAtFunction(ctx, platform)

	// ❌ 演示4：错误处理
	fmt.Println("\n❌ 演示4：错误处理")
	testErrorHandling(ctx, platform)

	fmt.Println("\n🎉 所有演示完成！")
	fmt.Println("\n💡 对比说明:")
	fmt.Println("   原始方式: 需要实现7个接口方法，约200行代码")
	fmt.Println("   简化方式: 只需实现1个Send方法，约50行核心代码")
	fmt.Println("   简化比例: 75% 代码减少！")
}

func testBasicMessage(ctx context.Context, platform platform.Platform) {
	msg := message.New()
	msg.Title = "NotifyHub"
	msg.Body = "这是一条来自简化版钉钉平台的测试消息"

	targets := []target.Target{
		external.CreateTarget("webhook", "webhook1"),
	}

	results, err := platform.Send(ctx, msg, targets)
	if err != nil {
		fmt.Printf("  ❌ 发送失败: %v\n", err)
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

func testTemplateMessage(ctx context.Context, platform platform.Platform) {
	// 通知消息
	msg1 := message.New()
	msg1.Variables = map[string]interface{}{
		"title":   "系统维护通知",
		"content": "系统将于今晚22:00-24:00进行维护，请提前保存工作。",
		"time":    time.Now().Format("2006-01-02 15:04:05"),
	}
	msg1.Metadata = map[string]interface{}{
		"template": "通知",
	}

	targets := []target.Target{
		external.CreateTarget("webhook", "webhook1"),
	}

	results, _ := platform.Send(ctx, msg1, targets)
	if len(results) > 0 && results[0].Success {
		fmt.Println("  ✅ 通知消息发送成功")
	}

	// 告警消息
	msg2 := message.New()
	msg2.Variables = map[string]interface{}{
		"level":   "严重",
		"project": "NotifyHub",
		"details": "数据库连接异常，请立即检查",
		"time":    time.Now().Format("2006-01-02 15:04:05"),
	}
	msg2.Metadata = map[string]interface{}{
		"template": "告警",
	}

	results, _ = platform.Send(ctx, msg2, targets)
	if len(results) > 0 && results[0].Success {
		fmt.Println("  ✅ 告警消息发送成功")
	}

	// 日报消息
	msg3 := message.New()
	msg3.Variables = map[string]interface{}{
		"date":      time.Now().Format("2006-01-02"),
		"completed": "- 完成钉钉平台简化实现\n- 更新文档和示例",
		"ongoing":   "- 优化用户体验\n- 性能测试",
		"tomorrow":  "- 发布新版本\n- 用户培训",
	}
	msg3.Metadata = map[string]interface{}{
		"template": "日报",
	}

	results, _ = platform.Send(ctx, msg3, targets)
	if len(results) > 0 && results[0].Success {
		fmt.Println("  ✅ 日报消息发送成功")
	}
}

func testAtFunction(ctx context.Context, platform platform.Platform) {
	msg := message.New()
	msg.Body = "这是一条@测试消息"

	// @所有人
	target1 := external.CreateTarget("group", "@all")
	results, _ := platform.Send(ctx, msg, []target.Target{target1})
	if len(results) > 0 && results[0].Success {
		fmt.Println("  ✅ @所有人消息发送成功")
	}

	// @指定手机号
	target2 := external.CreateTarget("group", "@13800138000,13800138001")
	results, _ = platform.Send(ctx, msg, []target.Target{target2})
	if len(results) > 0 && results[0].Success {
		fmt.Println("  ✅ @指定用户消息发送成功")
	}
}

func testErrorHandling(ctx context.Context, platform platform.Platform) {
	msg := message.New()
	msg.Body = "错误测试"

	// 测试无效目标格式
	invalidTarget := external.CreateTarget("webhook", "invalid_format")
	results, _ := platform.Send(ctx, msg, []target.Target{invalidTarget})
	if len(results) > 0 && results[0].Error != nil {
		fmt.Printf("  ✅ 无效目标被正确拒绝: %v\n", results[0].Error)
	}

	// 测试无效手机号
	invalidMobileTarget := external.CreateTarget("group", "@123")
	results, _ = platform.Send(ctx, msg, []target.Target{invalidMobileTarget})
	if len(results) > 0 && results[0].Error != nil {
		fmt.Printf("  ✅ 无效手机号被正确拒绝: %v\n", results[0].Error)
	}
}
