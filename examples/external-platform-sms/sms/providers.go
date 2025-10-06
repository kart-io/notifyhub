// SMS providers implementation
package sms

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// AliyunProvider implements Aliyun SMS service
type AliyunProvider struct {
	accessKeyID     string
	accessKeySecret string
	signName        string
	endpoint        string
}

// NewAliyunProvider creates a new Aliyun SMS provider
func NewAliyunProvider(credentials map[string]string) (SMSProvider, error) {
	accessKeyID, ok := credentials["access_key_id"]
	if !ok {
		return nil, fmt.Errorf("access_key_id is required for Aliyun provider")
	}

	accessKeySecret, ok := credentials["access_key_secret"]
	if !ok {
		return nil, fmt.Errorf("access_key_secret is required for Aliyun provider")
	}

	signName, ok := credentials["sign_name"]
	if !ok {
		return nil, fmt.Errorf("sign_name is required for Aliyun provider")
	}

	return &AliyunProvider{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		signName:        signName,
		endpoint:        credentials["endpoint"], // 可选
	}, nil
}

func (p *AliyunProvider) Name() string {
	return "aliyun"
}

func (p *AliyunProvider) Send(ctx context.Context, phone, content, templateID string) (*SMSResult, error) {
	// 模拟阿里云短信发送
	if strings.Contains(phone, "fail") {
		return nil, fmt.Errorf("阿里云短信发送失败: 手机号无效")
	}

	// 计算短信条数
	parts := calculateSMSParts(content)
	cost := float64(parts) * 0.045 // 阿里云短信价格：0.045元/条

	return &SMSResult{
		MessageID: fmt.Sprintf("aliyun_%d", time.Now().Unix()),
		Status:    "sent",
		Cost:      cost,
		Parts:     parts,
		Metadata: map[string]string{
			"provider":    "aliyun",
			"sign_name":   p.signName,
			"template_id": templateID,
		},
	}, nil
}

func (p *AliyunProvider) ValidateCredentials() error {
	if p.accessKeyID == "invalid" || p.accessKeySecret == "invalid" {
		return fmt.Errorf("invalid Aliyun credentials")
	}
	return nil
}

func (p *AliyunProvider) GetStatus() ProviderStatus {
	return ProviderStatus{
		Available: true,
		Quota: QuotaInfo{
			Remaining: 9500,
			Total:     10000,
			Reset:     int(time.Now().Add(24 * time.Hour).Unix()),
		},
		Metadata: map[string]string{
			"region":    "cn-hangzhou",
			"sign_name": p.signName,
		},
	}
}

func (p *AliyunProvider) Close() error {
	return nil
}

// TencentProvider implements Tencent Cloud SMS service
type TencentProvider struct {
	secretID  string
	secretKey string
	appID     string
	signName  string
}

// NewTencentProvider creates a new Tencent SMS provider
func NewTencentProvider(credentials map[string]string) (SMSProvider, error) {
	secretID, ok := credentials["secret_id"]
	if !ok {
		return nil, fmt.Errorf("secret_id is required for Tencent provider")
	}

	secretKey, ok := credentials["secret_key"]
	if !ok {
		return nil, fmt.Errorf("secret_key is required for Tencent provider")
	}

	appID, ok := credentials["app_id"]
	if !ok {
		return nil, fmt.Errorf("app_id is required for Tencent provider")
	}

	return &TencentProvider{
		secretID:  secretID,
		secretKey: secretKey,
		appID:     appID,
		signName:  credentials["sign_name"],
	}, nil
}

func (p *TencentProvider) Name() string {
	return "tencent"
}

func (p *TencentProvider) Send(ctx context.Context, phone, content, templateID string) (*SMSResult, error) {
	// 模拟腾讯云短信发送
	if strings.Contains(phone, "fail") {
		return nil, fmt.Errorf("腾讯云短信发送失败: 手机号不在白名单")
	}

	parts := calculateSMSParts(content)
	cost := float64(parts) * 0.055 // 腾讯云短信价格：0.055元/条

	return &SMSResult{
		MessageID: fmt.Sprintf("tencent_%d", time.Now().Unix()),
		Status:    "success",
		Cost:      cost,
		Parts:     parts,
		Metadata: map[string]string{
			"provider":    "tencent",
			"app_id":      p.appID,
			"template_id": templateID,
		},
	}, nil
}

func (p *TencentProvider) ValidateCredentials() error {
	if p.secretID == "invalid" || p.secretKey == "invalid" {
		return fmt.Errorf("invalid Tencent credentials")
	}
	return nil
}

func (p *TencentProvider) GetStatus() ProviderStatus {
	return ProviderStatus{
		Available: true,
		Quota: QuotaInfo{
			Remaining: 8800,
			Total:     10000,
			Reset:     int(time.Now().Add(24 * time.Hour).Unix()),
		},
		Metadata: map[string]string{
			"region":  "ap-beijing",
			"app_id":  p.appID,
			"version": "2021-01-11",
		},
	}
}

func (p *TencentProvider) Close() error {
	return nil
}

// TwilioProvider implements Twilio SMS service
type TwilioProvider struct {
	accountSID string
	authToken  string
	fromNumber string
}

// NewTwilioProvider creates a new Twilio SMS provider
func NewTwilioProvider(credentials map[string]string) (SMSProvider, error) {
	accountSID, ok := credentials["account_sid"]
	if !ok {
		return nil, fmt.Errorf("account_sid is required for Twilio provider")
	}

	authToken, ok := credentials["auth_token"]
	if !ok {
		return nil, fmt.Errorf("auth_token is required for Twilio provider")
	}

	fromNumber, ok := credentials["from_number"]
	if !ok {
		return nil, fmt.Errorf("from_number is required for Twilio provider")
	}

	return &TwilioProvider{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
	}, nil
}

func (p *TwilioProvider) Name() string {
	return "twilio"
}

func (p *TwilioProvider) Send(ctx context.Context, phone, content, templateID string) (*SMSResult, error) {
	// 模拟 Twilio 短信发送
	if strings.Contains(phone, "fail") {
		return nil, fmt.Errorf("Twilio SMS failed: Invalid phone number")
	}

	parts := calculateSMSParts(content)
	cost := float64(parts) * 0.0075 // Twilio 价格：$0.0075/SMS

	return &SMSResult{
		MessageID: fmt.Sprintf("twilio_SM%d", time.Now().Unix()),
		Status:    "delivered",
		Cost:      cost,
		Parts:     parts,
		Metadata: map[string]string{
			"provider":    "twilio",
			"from_number": p.fromNumber,
			"account_sid": p.accountSID,
		},
	}, nil
}

func (p *TwilioProvider) ValidateCredentials() error {
	if p.accountSID == "invalid" || p.authToken == "invalid" {
		return fmt.Errorf("invalid Twilio credentials")
	}
	return nil
}

func (p *TwilioProvider) GetStatus() ProviderStatus {
	return ProviderStatus{
		Available: true,
		Quota: QuotaInfo{
			Remaining: -1, // Twilio 按用量计费，无固定配额
			Total:     -1,
			Reset:     0,
		},
		Metadata: map[string]string{
			"region":      "us-east-1",
			"from_number": p.fromNumber,
			"api_version": "2010-04-01",
		},
	}
}

func (p *TwilioProvider) Close() error {
	return nil
}

// NexmoProvider implements Vonage (Nexmo) SMS service
type NexmoProvider struct {
	apiKey    string
	apiSecret string
	fromName  string
}

// NewNexmoProvider creates a new Nexmo SMS provider
func NewNexmoProvider(credentials map[string]string) (SMSProvider, error) {
	apiKey, ok := credentials["api_key"]
	if !ok {
		return nil, fmt.Errorf("api_key is required for Nexmo provider")
	}

	apiSecret, ok := credentials["api_secret"]
	if !ok {
		return nil, fmt.Errorf("api_secret is required for Nexmo provider")
	}

	return &NexmoProvider{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		fromName:  credentials["from_name"], // 可选
	}, nil
}

func (p *NexmoProvider) Name() string {
	return "nexmo"
}

func (p *NexmoProvider) Send(ctx context.Context, phone, content, templateID string) (*SMSResult, error) {
	// 模拟 Nexmo 短信发送
	if strings.Contains(phone, "fail") {
		return nil, fmt.Errorf("Nexmo SMS failed: Rejected by carrier")
	}

	parts := calculateSMSParts(content)
	cost := float64(parts) * 0.0053 // Nexmo 价格：$0.0053/SMS

	return &SMSResult{
		MessageID: fmt.Sprintf("nexmo_%s", generateRandomID()),
		Status:    "delivered",
		Cost:      cost,
		Parts:     parts,
		Metadata: map[string]string{
			"provider":  "nexmo",
			"from_name": p.fromName,
			"network":   "carrier_network",
		},
	}, nil
}

func (p *NexmoProvider) ValidateCredentials() error {
	if p.apiKey == "invalid" || p.apiSecret == "invalid" {
		return fmt.Errorf("invalid Nexmo credentials")
	}
	return nil
}

func (p *NexmoProvider) GetStatus() ProviderStatus {
	return ProviderStatus{
		Available: true,
		Quota: QuotaInfo{
			Remaining: -1, // Nexmo 按用量计费
			Total:     -1,
			Reset:     0,
		},
		Metadata: map[string]string{
			"api_version": "v1",
			"from_name":   p.fromName,
		},
	}
}

func (p *NexmoProvider) Close() error {
	return nil
}

// MockProvider implements a mock SMS provider for testing
type MockProvider struct {
	shouldFail bool
	delay      time.Duration
}

// NewMockProvider creates a new mock SMS provider
func NewMockProvider(credentials map[string]string) (SMSProvider, error) {
	provider := &MockProvider{
		shouldFail: credentials["should_fail"] == "true",
		delay:      1 * time.Second, // 默认延迟1秒
	}

	if delayStr, ok := credentials["delay"]; ok {
		if delay, err := time.ParseDuration(delayStr); err == nil {
			provider.delay = delay
		}
	}

	return provider, nil
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) Send(ctx context.Context, phone, content, templateID string) (*SMSResult, error) {
	// 模拟网络延迟
	time.Sleep(p.delay)

	if p.shouldFail || strings.Contains(phone, "fail") {
		return nil, fmt.Errorf("Mock SMS failed: Simulated failure")
	}

	parts := calculateSMSParts(content)

	return &SMSResult{
		MessageID: fmt.Sprintf("mock_%s", generateRandomID()),
		Status:    "sent",
		Cost:      0.0, // 测试环境免费
		Parts:     parts,
		Metadata: map[string]string{
			"provider":    "mock",
			"template_id": templateID,
			"simulated":   "true",
		},
	}, nil
}

func (p *MockProvider) ValidateCredentials() error {
	return nil // Mock provider 总是有效
}

func (p *MockProvider) GetStatus() ProviderStatus {
	return ProviderStatus{
		Available: !p.shouldFail,
		Quota: QuotaInfo{
			Remaining: 1000,
			Total:     1000,
			Reset:     int(time.Now().Add(24 * time.Hour).Unix()),
		},
		Metadata: map[string]string{
			"provider": "mock",
			"mode":     "testing",
		},
	}
}

func (p *MockProvider) Close() error {
	return nil
}

// Helper functions

// calculateSMSParts calculates how many SMS parts are needed
func calculateSMSParts(content string) int {
	length := len([]rune(content))
	if length <= 70 {
		return 1
	}
	return (length + 66) / 67 // 多条短信时每条67个字符
}

// generateRandomID generates a random ID for mock messages
func generateRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
