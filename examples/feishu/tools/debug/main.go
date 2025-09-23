package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// 直接实现一个简单的飞书发送器来测试真实的网络请求
type DirectFeishuSender struct {
	webhookURL string
	secret     string
	client     *http.Client
}

type FeishuMessage struct {
	MsgType   string      `json:"msg_type"`
	Content   interface{} `json:"content"`
	Timestamp string      `json:"timestamp,omitempty"`
	Sign      string      `json:"sign,omitempty"`
}

type TextContent struct {
	Text string `json:"text"`
}

func NewDirectFeishuSender(webhookURL, secret string) *DirectFeishuSender {
	return &DirectFeishuSender{
		webhookURL: webhookURL,
		secret:     secret,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// generateSignature 生成飞书签名
func (d *DirectFeishuSender) generateSignature(timestamp string) string {
	if d.secret == "" {
		return ""
	}

	stringToSign := timestamp + "\n" + d.secret
	h := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (d *DirectFeishuSender) SendText(ctx context.Context, text string) error {
	// 生成时间戳和签名
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sign := d.generateSignature(timestamp)

	// 构建消息
	message := FeishuMessage{
		MsgType: "text",
		Content: TextContent{
			Text: text,
		},
	}

	// 如果有密钥，添加签名信息
	if d.secret != "" {
		message.Timestamp = timestamp
		message.Sign = sign
		fmt.Printf("🔐 添加签名验证:\n")
		fmt.Printf("  时间戳: %s\n", timestamp)
		fmt.Printf("  签名: %s\n\n", sign)
	}

	// 序列化消息
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	fmt.Printf("📤 发送的消息内容:\n%s\n\n", string(payload))

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NotifyHub-Debug/1.0")

	fmt.Printf("🔗 请求 URL: %s\n", d.webhookURL)
	fmt.Printf("📋 请求头:\n")
	for k, v := range req.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Println()

	// 发送请求
	fmt.Printf("⏳ 发送请求...\n")
	start := time.Now()
	resp, err := d.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return fmt.Errorf("发送请求失败 (耗时 %v): %w", duration, err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	fmt.Printf("📥 响应状态: %s (耗时 %v)\n", resp.Status, duration)
	fmt.Printf("📋 响应头:\n")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Printf("📄 响应内容:\n%s\n\n", string(body))

	// 检查响应状态
	if resp.StatusCode != 200 {
		return fmt.Errorf("请求失败: HTTP %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 尝试解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("⚠️  无法解析响应 JSON: %v\n", err)
		return nil
	}

	// 检查响应状态码
	if code, ok := response["code"]; ok {
		if codeInt, ok := code.(float64); ok && codeInt != 0 {
			msg, _ := response["msg"].(string)
			return fmt.Errorf("飞书API返回错误: code=%v, msg=%s", code, msg)
		}
	}

	fmt.Printf("✅ 消息发送成功!\n")
	return nil
}

func main() {
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	secret := os.Getenv("FEISHU_SECRET")

	if webhookURL == "" {
		fmt.Println("❌ 请设置环境变量 FEISHU_WEBHOOK_URL")
		fmt.Println("示例:")
		fmt.Println(`export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your-token"`)
		os.Exit(1)
	}

	fmt.Println("🧪 飞书直接发送测试")
	fmt.Println("====================")
	fmt.Printf("Webhook URL: %s\n", webhookURL)
	fmt.Printf("Secret: %s\n", secret)
	fmt.Println()

	sender := NewDirectFeishuSender(webhookURL, secret)
	ctx := context.Background()

	testMessage := fmt.Sprintf("🧪 NotifyHub 直接测试\n时间: %s\n这是一条通过 HTTP 直接发送的测试消息。",
		time.Now().Format("2006-01-02 15:04:05"))

	fmt.Println("📝 准备发送测试消息...")
	fmt.Printf("内容: %s\n\n", testMessage)

	if err := sender.SendText(ctx, testMessage); err != nil {
		fmt.Printf("❌ 发送失败: %v\n\n", err)

		fmt.Println("🔧 故障排除建议:")
		fmt.Println("1. 检查 Webhook URL 是否正确和有效")
		fmt.Println("2. 确认机器人已添加到目标群聊")
		fmt.Println("3. 检查网络连接是否正常")
		fmt.Println("4. 验证飞书服务是否可访问")
		fmt.Println("5. 查看上述响应内容了解具体错误")

		os.Exit(1)
	}

	fmt.Println("🎉 测试完成!")
	fmt.Println("\n如果你在飞书群聊中看到了消息,说明发送成功!")
	fmt.Println("如果没有看到消息,请检查:")
	fmt.Println("- 机器人是否已正确添加到群聊")
	fmt.Println("- Webhook URL 是否对应正确的群聊")
	fmt.Println("- 群聊设置是否允许机器人发送消息")
}
