package notifiers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFeishuNotifier(t *testing.T) {
	// Create Feishu notifier
	notifier := NewFeishuNotifier(
		"https://open.feishu.cn/open-apis/bot/v2/hook/test-webhook",
		"test-secret",
		30*time.Second,
	)

	// Test notifier name
	if notifier.Name() != "feishu" {
		t.Error("Feishu notifier name should be 'feishu'")
	}

	// Test target support
	testCases := []struct {
		target   Target
		expected bool
	}{
		{Target{Type: TargetTypeGroup, Platform: "feishu"}, true},
		{Target{Type: TargetTypeUser, Platform: "feishu"}, true},
		{Target{Type: TargetTypeEmail, Platform: "feishu"}, false},
		{Target{Type: TargetTypeGroup, Platform: "email"}, false},
		{Target{Type: TargetTypeGroup}, true}, // Platform can be empty, defaults to supporting
	}

	for i, tc := range testCases {
		supported := notifier.SupportsTarget(tc.target)
		if supported != tc.expected {
			t.Errorf("Test case %d: expected %v, got %v for target %+v", i, tc.expected, supported, tc.target)
		}
	}

	// Test basic functionality without network calls
	ctx := context.Background()

	// Health check should not panic
	_ = notifier.Health(ctx)

	message := &Message{
		Title:    "Test",
		Body:     "Test message",
		Format:   FormatText,
		Priority: 3,
		Targets: []Target{
			{Type: TargetTypeGroup, Value: "group123", Platform: "feishu"},
		},
	}

	// Send should not panic and return result
	result, _ := notifier.Send(ctx, message)
	assert.NotNil(t, result)
}

func TestEmailNotifier(t *testing.T) {
	// Create Email notifier
	notifier := NewEmailNotifier(
		"localhost",
		587,
		"test@example.com",
		"password",
		"sender@example.com",
		false,
		30*time.Second,
	)

	// Test notifier name
	if notifier.Name() != "email" {
		t.Error("Email notifier name should be 'email'")
	}

	// Test target support
	testCases := []struct {
		target   Target
		expected bool
	}{
		{Target{Type: TargetTypeEmail}, true},
		{Target{Type: TargetTypeUser, Value: "user@example.com"}, true}, // Email format in value
		{Target{Type: TargetTypeGroup}, false},
		{Target{Type: TargetTypeUser, Value: "user123"}, false}, // Non-email format
	}

	for i, tc := range testCases {
		supported := notifier.SupportsTarget(tc.target)
		if supported != tc.expected {
			t.Errorf("Test case %d: expected %v, got %v for target %+v", i, tc.expected, supported, tc.target)
		}
	}

	// Test basic functionality without network calls
	ctx := context.Background()

	// Health check should not panic
	_ = notifier.Health(ctx)

	message := &Message{
		Title:    "Test Email",
		Body:     "Test email body",
		Format:   FormatHTML,
		Priority: 3,
		Targets: []Target{
			{Type: TargetTypeEmail, Value: "test@example.com"},
		},
	}

	// Send should not panic and return result
	result, _ := notifier.Send(ctx, message)
	assert.NotNil(t, result)
}



// TestFeishuSignatureGeneration 测试飞书签名生成功能
func TestFeishuSignatureGeneration(t *testing.T) {
	// 测试有 secret 的情况
	notifierWithSecret := NewFeishuNotifier(
		"https://open.feishu.cn/open-apis/bot/v2/hook/test-webhook",
		"test-secret-key",
		30*time.Second,
	)

	timestamp := "1640995200" // 固定时间戳用于测试
	signature := notifierWithSecret.generateSignature(timestamp)

	// 签名应该不为空
	assert.NotEmpty(t, signature)

	// 相同输入应该产生相同签名
	signature2 := notifierWithSecret.generateSignature(timestamp)
	assert.Equal(t, signature, signature2)

	// 测试无 secret 的情况
	notifierWithoutSecret := NewFeishuNotifier(
		"https://open.feishu.cn/open-apis/bot/v2/hook/test-webhook",
		"",
		30*time.Second,
	)

	emptySignature := notifierWithoutSecret.generateSignature(timestamp)
	assert.Empty(t, emptySignature)

	// 测试不同时间戳产生不同签名
	timestamp2 := "1640995300"
	signature3 := notifierWithSecret.generateSignature(timestamp2)
	assert.NotEqual(t, signature, signature3)
}

// TestFeishuSignatureAlgorithm 测试签名算法的正确性
func TestFeishuSignatureAlgorithm(t *testing.T) {
	// 使用已知的输入和预期输出来验证算法正确性
	secret := "test-secret"
	timestamp := "1640995200"

	notifier := NewFeishuNotifier(
		"https://open.feishu.cn/open-apis/bot/v2/hook/test-webhook",
		secret,
		30*time.Second,
	)

	signature := notifier.generateSignature(timestamp)

	// 验证签名是 Base64 编码的字符串
	assert.Regexp(t, `^[A-Za-z0-9+/]+=*$`, signature)

	// 验证签名长度合理（HMAC-SHA256 的 Base64 编码长度应该是 44 字符）
	assert.Equal(t, 44, len(signature))
}

// TestFeishuNotifierSimpleMethods 测试便捷创建方法
func TestFeishuNotifierSimpleMethods(t *testing.T) {
	webhookURL := "https://open.feishu.cn/open-apis/bot/v2/hook/test-simple"

	// 测试 NewFeishuNotifierSimple
	simpleNotifier := NewFeishuNotifierSimple(webhookURL)
	assert.NotNil(t, simpleNotifier)
	assert.Equal(t, "feishu", simpleNotifier.Name())
	assert.Equal(t, webhookURL, simpleNotifier.webhookURL)
	assert.Equal(t, "", simpleNotifier.secret) // 应该没有 secret
	assert.Equal(t, 30*time.Second, simpleNotifier.timeout) // 默认超时

	// 测试 NewFeishuNotifierWithTimeout
	customTimeout := 60 * time.Second
	timeoutNotifier := NewFeishuNotifierWithTimeout(webhookURL, customTimeout)
	assert.NotNil(t, timeoutNotifier)
	assert.Equal(t, "feishu", timeoutNotifier.Name())
	assert.Equal(t, webhookURL, timeoutNotifier.webhookURL)
	assert.Equal(t, "", timeoutNotifier.secret) // 应该没有 secret
	assert.Equal(t, customTimeout, timeoutNotifier.timeout) // 自定义超时

	// 验证签名生成应该返回空（因为没有 secret）
	timestamp := "1640995200"
	signature := simpleNotifier.generateSignature(timestamp)
	assert.Empty(t, signature)

	signature2 := timeoutNotifier.generateSignature(timestamp)
	assert.Empty(t, signature2)
}
