package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
	fmt.Println("=== Task 8.2: Platform Configuration Strong-Type Implementation Validation ===")

	// Test 1: Verify FeishuConfig structure completeness
	fmt.Println("\n1. Testing FeishuConfig structure completeness...")
	testFeishuConfigStructure()

	// Test 2: Verify FeishuConfig validation
	fmt.Println("\n2. Testing FeishuConfig validation...")
	testFeishuConfigValidation()

	// Test 3: Verify option functions
	fmt.Println("\n3. Testing platform option functions...")
	testPlatformOptionFunctions()

	// Test 4: Verify JSON serialization
	fmt.Println("\n4. Testing JSON serialization...")
	testJSONSerialization()

	// Test 5: Verify map to strong-typed conversion
	fmt.Println("\n5. Testing map to strong-typed conversion...")
	testMapConversion()

	// Test 6: Verify other platform configs exist
	fmt.Println("\n6. Testing other platform configurations...")
	testOtherPlatformConfigs()

	fmt.Println("\n=== Task 8.2 Validation Complete ===")
}

func testFeishuConfigStructure() {
	cfg := &config.FeishuConfig{}
	cfgType := reflect.TypeOf(cfg).Elem()

	requiredFields := map[string]string{
		"WebhookURL": "string",
		"Secret":     "string",
		"AppID":      "string",
		"AppSecret":  "string",
		"AuthType":   "string",
		"Timeout":    "time.Duration",
		"MaxRetries": "int",
		"RateLimit":  "int",
		"SignVerify": "bool",
		"Keywords":   "[]string",
	}

	fmt.Printf("  FeishuConfig has %d fields\n", cfgType.NumField())

	for fieldName, expectedType := range requiredFields {
		field, found := cfgType.FieldByName(fieldName)
		if !found {
			fmt.Printf("  ❌ Field %s missing\n", fieldName)
			continue
		}

		if !contains(field.Type.String(), expectedType) {
			fmt.Printf("  ❌ Field %s has wrong type: %s (expected: %s)\n", fieldName, field.Type.String(), expectedType)
			continue
		}

		// Check JSON and YAML tags
		jsonTag := field.Tag.Get("json")
		yamlTag := field.Tag.Get("yaml")
		if jsonTag == "" || yamlTag == "" {
			fmt.Printf("  ⚠️  Field %s missing tags (json: %s, yaml: %s)\n", fieldName, jsonTag, yamlTag)
		} else {
			fmt.Printf("  ✓ Field %s: %s with tags\n", fieldName, field.Type.String())
		}
	}
}

func testFeishuConfigValidation() {
	tests := []struct {
		name      string
		config    *config.FeishuConfig
		wantError bool
	}{
		{
			name: "valid webhook config",
			config: &config.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
				Secret:     "test-secret",
				AuthType:   "webhook",
			},
			wantError: false,
		},
		{
			name: "invalid webhook URL",
			config: &config.FeishuConfig{
				WebhookURL: "invalid-url",
				AuthType:   "webhook",
			},
			wantError: true,
		},
		{
			name: "missing webhook URL for webhook auth",
			config: &config.FeishuConfig{
				AuthType: "webhook",
			},
			wantError: true,
		},
		{
			name: "negative timeout",
			config: &config.FeishuConfig{
				WebhookURL: "https://test.com",
				Timeout:    -5 * time.Second,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		err := feishu.ValidateConfig(tt.config)
		if tt.wantError && err == nil {
			fmt.Printf("  ❌ %s: Expected error but got none\n", tt.name)
		} else if !tt.wantError && err != nil {
			fmt.Printf("  ❌ %s: Unexpected error: %v\n", tt.name, err)
		} else {
			fmt.Printf("  ✓ %s: Validation as expected\n", tt.name)
		}
	}
}

func testPlatformOptionFunctions() {
	// Test WithFeishu
	feishuConfig := config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		Secret:     "test-secret",
		AuthType:   "webhook",
	}

	cfg, err := config.NewConfig(config.WithFeishu(feishuConfig))
	if err != nil {
		fmt.Printf("  ❌ WithFeishu failed: %v\n", err)
	} else if cfg.Feishu == nil {
		fmt.Printf("  ❌ WithFeishu: Feishu config not set\n")
	} else if cfg.Feishu.WebhookURL != feishuConfig.WebhookURL {
		fmt.Printf("  ❌ WithFeishu: WebhookURL not preserved\n")
	} else {
		fmt.Printf("  ✓ WithFeishu works correctly\n")
	}

	// Test WithEmail
	emailConfig := config.EmailConfig{
		SMTPHost: "smtp.gmail.com",
		SMTPPort: 587,
		SMTPFrom: "test@example.com",
	}

	cfg2, err := config.NewConfig(config.WithEmail(emailConfig))
	if err != nil {
		fmt.Printf("  ❌ WithEmail failed: %v\n", err)
	} else if cfg2.Email == nil {
		fmt.Printf("  ❌ WithEmail: Email config not set\n")
	} else {
		fmt.Printf("  ✓ WithEmail works correctly\n")
	}

	// Test convenience function WithFeishuWebhook
	cfg3, err := config.NewConfig(config.WithFeishuWebhook("https://test.com", "secret"))
	if err != nil {
		fmt.Printf("  ❌ WithFeishuWebhook failed: %v\n", err)
	} else if cfg3.Feishu == nil || cfg3.Feishu.WebhookURL != "https://test.com" {
		fmt.Printf("  ❌ WithFeishuWebhook: Configuration not correct\n")
	} else {
		fmt.Printf("  ✓ WithFeishuWebhook convenience function works\n")
	}
}

func testJSONSerialization() {
	original := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		Secret:     "test-secret-123",
		AuthType:   "webhook",
		Timeout:    45 * time.Second,
		MaxRetries: 5,
		RateLimit:  60,
		SignVerify: true,
		Keywords:   []string{"alert", "notification", "warning"},
	}

	// JSON serialization
	jsonData, err := json.Marshal(original)
	if err != nil {
		fmt.Printf("  ❌ JSON Marshal failed: %v\n", err)
		return
	}

	// JSON deserialization
	var deserialized config.FeishuConfig
	err = json.Unmarshal(jsonData, &deserialized)
	if err != nil {
		fmt.Printf("  ❌ JSON Unmarshal failed: %v\n", err)
		return
	}

	// Compare
	if original.WebhookURL != deserialized.WebhookURL ||
		original.Secret != deserialized.Secret ||
		original.AuthType != deserialized.AuthType ||
		original.Timeout != deserialized.Timeout ||
		original.MaxRetries != deserialized.MaxRetries ||
		original.SignVerify != deserialized.SignVerify ||
		len(original.Keywords) != len(deserialized.Keywords) {
		fmt.Printf("  ❌ JSON serialization roundtrip failed\n")
	} else {
		fmt.Printf("  ✓ JSON serialization roundtrip successful\n")
	}
}

func testMapConversion() {
	configMap := map[string]interface{}{
		"webhook_url": "https://test.com",
		"secret":      "test-secret",
		"timeout":     "30s",
		"keywords":    []string{"alert", "notification"},
	}

	feishuConfig, err := feishu.NewConfigFromMap(configMap)
	if err != nil {
		fmt.Printf("  ❌ Map conversion failed: %v\n", err)
		return
	}

	if feishuConfig.WebhookURL != "https://test.com" ||
		feishuConfig.Secret != "test-secret" ||
		feishuConfig.Timeout != 30*time.Second ||
		len(feishuConfig.Keywords) != 2 {
		fmt.Printf("  ❌ Map conversion values incorrect\n")
	} else {
		fmt.Printf("  ✓ Map to strong-typed conversion successful\n")
	}
}

func testOtherPlatformConfigs() {
	// Verify other platform configurations exist
	platforms := []interface{}{
		&config.EmailConfig{},
		&config.SMSConfig{},
		&config.SlackConfig{},
		&config.WebhookConfig{},
		&config.DingTalkConfig{},
	}

	for _, platform := range platforms {
		platformType := reflect.TypeOf(platform).Elem()
		if platformType.NumField() == 0 {
			fmt.Printf("  ❌ Platform %s has no fields\n", platformType.Name())
		} else {
			fmt.Printf("  ✓ Platform %s exists with %d fields\n", platformType.Name(), platformType.NumField())
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		   len(s) > len(substr) && s[:len(substr)] == substr ||
		   s == substr
}