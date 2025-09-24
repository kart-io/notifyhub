// 飞书发送调试工具 - Debug Feishu sending
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
	fmt.Println("🔍 飞书发送调试工具")
	fmt.Println("==================")
	fmt.Println()

	// 配置信息
	webhookURL := "https://open.feishu.cn/open-apis/bot/v2/hook/688dc0bf-c74b-41d1-a6b9-8cb660477488"
	secret := "gQURr67BPOsTZlI7jBn0Jh"

	fmt.Printf("📋 配置信息:\n")
	fmt.Printf("   Webhook URL: %s\n", webhookURL)
	fmt.Printf("   Secret: %s\n", secret)
	fmt.Println()

	// 步骤1: 直接HTTP测试飞书webhook
	fmt.Println("🔗 步骤1: 直接HTTP测试")
	fmt.Println("-------------------")
	testDirectHTTP(webhookURL, secret)
	fmt.Println()

	// 步骤2: 使用NotifyHub (不使用签名)
	fmt.Println("🚀 步骤2: NotifyHub测试 (无签名)")
	fmt.Println("-----------------------------")
	testNotifyHubNoAuth(webhookURL)
	fmt.Println()

	// 步骤3: 使用NotifyHub (使用签名)
	fmt.Println("🔒 步骤3: NotifyHub测试 (使用签名)")
	fmt.Println("------------------------------")
	testNotifyHubWithAuth(webhookURL, secret)
	fmt.Println()

	fmt.Println("🔍 调试完成!")
}

// 直接HTTP测试
func testDirectHTTP(webhookURL, secret string) {
	// 构造飞书消息格式
	message := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]interface{}{
			"text": "🔍 直接HTTP测试消息 - 来自NotifyHub调试工具",
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %v\n", err)
		return
	}

	fmt.Printf("📤 发送payload: %s\n", string(payload))

	// 发送HTTP请求
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 发送请求失败: %v\n", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("📨 响应状态: %d\n", resp.StatusCode)
	fmt.Printf("📨 响应内容: %s\n", string(body))

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ 直接HTTP测试成功!\n")
	} else {
		fmt.Printf("❌ 直接HTTP测试失败 - 状态码: %d\n", resp.StatusCode)
	}
}

// NotifyHub测试 (无认证)
func testNotifyHubNoAuth(webhookURL string) {
	hub, err := notifyhub.NewHub(
		feishu.WithFeishu(webhookURL,
			// 不设置签名，使用默认的none模式
			feishu.WithFeishuTimeout(30*time.Second),
		),
	)
	if err != nil {
		fmt.Printf("❌ Hub创建失败: %v\n", err)
		return
	}
	defer func() { _ = hub.Close(context.Background()) }()

	message := notifyhub.NewMessage("🚀 NotifyHub测试 (无签名)").
		WithBody("这是通过NotifyHub发送的测试消息，未使用签名认证").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	ctx := context.Background()
	receipt, err := hub.Send(ctx, message)
	if err != nil {
		fmt.Printf("❌ NotifyHub发送失败: %v\n", err)
		return
	}

	fmt.Printf("📊 发送结果:\n")
	fmt.Printf("   总数: %d, 成功: %d, 失败: %d\n",
		receipt.Total, receipt.Successful, receipt.Failed)

	for _, result := range receipt.Results {
		if result.Success {
			fmt.Printf("   ✅ 平台 %s: 发送成功\n", result.Platform)
			fmt.Printf("      消息ID: %s\n", result.MessageID)
			fmt.Printf("      目标: %s\n", result.Target)
		} else {
			fmt.Printf("   ❌ 平台 %s: 发送失败 - %s\n", result.Platform, result.Error)
		}

		fmt.Printf("      耗时: %dms\n", result.Duration.Milliseconds())
	}
}

// NotifyHub测试 (使用签名)
func testNotifyHubWithAuth(webhookURL, secret string) {
	hub, err := notifyhub.NewHub(
		feishu.WithFeishu(webhookURL,
			feishu.WithFeishuSecret(secret),
			feishu.WithFeishuAuthMode(feishu.AuthModeSignature), // 明确设置签名模式
			feishu.WithFeishuTimeout(30*time.Second),
		),
	)
	if err != nil {
		fmt.Printf("❌ Hub创建失败: %v\n", err)
		return
	}
	defer func() { _ = hub.Close(context.Background()) }()

	message := notifyhub.NewMessage("🔒 NotifyHub测试 (使用签名)").
		WithBody("这是通过NotifyHub发送的测试消息，使用HMAC-SHA256签名认证").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	ctx := context.Background()
	receipt, err := hub.Send(ctx, message)
	if err != nil {
		fmt.Printf("❌ NotifyHub发送失败: %v\n", err)
		return
	}

	fmt.Printf("📊 发送结果:\n")
	fmt.Printf("   总数: %d, 成功: %d, 失败: %d\n",
		receipt.Total, receipt.Successful, receipt.Failed)

	for _, result := range receipt.Results {
		if result.Success {
			fmt.Printf("   ✅ 平台 %s: 发送成功\n", result.Platform)
			fmt.Printf("      消息ID: %s\n", result.MessageID)
			fmt.Printf("      目标: %s\n", result.Target)
		} else {
			fmt.Printf("   ❌ 平台 %s: 发送失败 - %s\n", result.Platform, result.Error)
		}

		fmt.Printf("      耗时: %dms\n", result.Duration.Milliseconds())
	}
}
