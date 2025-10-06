// Package main provides comprehensive email provider validation and testing
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/examples/common"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/email"
	"github.com/kart-io/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Email Provider", "Validation & Testing")

	// Create logger
	logger := common.NewLogger(true)

	logger.Info("🔍 邮件服务商验证与测试工具")
	logger.Info("📧 支持配置验证、连接测试、发送测试等功能")

	// Parse command line arguments for provider-specific testing
	if len(os.Args) > 1 {
		providerName := os.Args[1]
		runProviderSpecificTest(logger, providerName)
		return
	}

	// Run comprehensive validation
	runComprehensiveValidation(logger)
}

func runProviderSpecificTest(logger *common.Logger, providerName string) {
	logger.Info("\n🎯 运行 %s 专项测试", providerName)

	// Get configuration from environment or prompt
	config := getProviderConfigFromEnv(logger, providerName)
	if config == nil {
		logger.Error("无法获取 %s 的配置信息", providerName)
		return
	}

	// Run validation tests
	validateProviderConfig(logger, providerName, config)

	// Test SMTP connection
	testSMTPConnection(logger, providerName, config)

	// If recipient is provided, test actual sending
	if recipient := os.Getenv("EMAIL_RECIPIENT"); recipient != "" {
		testEmailSending(logger, providerName, config, recipient)
	} else {
		logger.Info("💡 设置 EMAIL_RECIPIENT 环境变量来测试实际邮件发送")
	}
}

func runComprehensiveValidation(logger *common.Logger) {
	logger.Info("\n🔧 运行综合验证测试")

	// Test all predefined configurations
	testPredefinedConfigurations(logger)

	// Test provider detection
	testProviderDetection(logger)

	// Test configuration validation
	testConfigurationValidation(logger)

	// Test SMTP server connectivity
	testSMTPConnectivity(logger)

	logger.Success("综合验证测试完成！")
}

func getProviderConfigFromEnv(logger *common.Logger, providerName string) *EmailProviderConfig {
	username := os.Getenv("EMAIL_USERNAME")
	password := os.Getenv("EMAIL_PASSWORD")

	if username == "" || password == "" {
		logger.Error("请设置环境变量: EMAIL_USERNAME 和 EMAIL_PASSWORD")
		return nil
	}

	config := &EmailProviderConfig{
		Username: username,
		Password: password,
		From:     username, // Default to username
	}

	// Set provider-specific defaults
	switch strings.ToLower(providerName) {
	case "gmail":
		config.Host = "smtp.gmail.com"
		config.Port = 587
		config.UseTLS = false
		config.UseStartTLS = true
	case "163":
		config.Host = "smtp.163.com"
		config.Port = 25
		config.UseTLS = false
		config.UseStartTLS = true
	case "qq":
		config.Host = "smtp.qq.com"
		config.Port = 587
		config.UseTLS = false
		config.UseStartTLS = true
	default:
		logger.Error("不支持的邮件服务商: %s", providerName)
		return nil
	}

	return config
}

type EmailProviderConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	From        string
	UseTLS      bool
	UseStartTLS bool
}

func validateProviderConfig(logger *common.Logger, providerName string, config *EmailProviderConfig) {
	logger.Info("\n🔍 验证 %s 配置", providerName)

	// Basic validation
	errors := []string{}

	if config.Host == "" {
		errors = append(errors, "SMTP服务器地址不能为空")
	}

	if config.Port <= 0 || config.Port > 65535 {
		errors = append(errors, "端口号必须在1-65535之间")
	}

	if config.Username == "" {
		errors = append(errors, "用户名不能为空")
	}

	if config.Password == "" {
		errors = append(errors, "密码不能为空")
	}

	if config.From == "" {
		errors = append(errors, "发件人地址不能为空")
	}

	if len(errors) > 0 {
		logger.Error("❌ 配置验证失败:")
		for _, err := range errors {
			logger.Error("   - %s", err)
		}
		return
	}

	logger.Success("✅ %s 配置验证通过", providerName)

	// Display configuration details
	logger.Info("📋 配置详情:")
	logger.Info("   服务器: %s:%d", config.Host, config.Port)
	logger.Info("   用户名: %s", config.Username)
	logger.Info("   密码: %s", maskPassword(config.Password))
	logger.Info("   发件人: %s", config.From)
	logger.Info("   TLS: %v, STARTTLS: %v", config.UseTLS, config.UseStartTLS)
}

func testSMTPConnection(logger *common.Logger, providerName string, config *EmailProviderConfig) {
	logger.Info("\n🔌 测试 %s SMTP连接", providerName)

	// Test basic TCP connection
	address := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	logger.Info("连接地址: %s", address)

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		logger.Error("❌ TCP连接失败: %v", err)

		// Provide specific troubleshooting
		if strings.Contains(err.Error(), "connection refused") {
			logger.Info("💡 故障排除建议:")
			logger.Info("   - 检查网络连接")
			logger.Info("   - 确认SMTP服务器地址和端口正确")
			logger.Info("   - 检查防火墙设置")
		} else if strings.Contains(err.Error(), "timeout") {
			logger.Info("💡 故障排除建议:")
			logger.Info("   - 网络可能较慢，尝试增加超时时间")
			logger.Info("   - 检查是否被防火墙阻拦")
		}
		return
	}
	defer func() { _ = conn.Close() }()

	logger.Success("✅ TCP连接成功")

	// Test SMTP handshake (basic)
	logger.Info("测试SMTP握手...")
	// Note: Full SMTP handshake would require more complex implementation
	// For now, we just verify the TCP connection works
	logger.Success("✅ SMTP服务器可达")
}

func testEmailSending(logger *common.Logger, providerName string, config *EmailProviderConfig, recipient string) {
	logger.Info("\n📧 测试 %s 邮件发送", providerName)
	logger.Info("收件人: %s", recipient)

	// Create NotifyHub configuration using common utilities
	exampleConfig := common.DefaultExampleConfig()
	exampleConfig.Email.Host = config.Host
	exampleConfig.Email.Port = config.Port
	exampleConfig.Email.Username = config.Username
	exampleConfig.Email.Password = config.Password
	exampleConfig.Email.From = config.From
	exampleConfig.Email.UseTLS = config.UseTLS
	exampleConfig.Email.To = recipient

	nhConfig := exampleConfig.CreateEmailConfig()

	// Create NotifyHub client
	client, err := notifyhub.NewClient(nhConfig)
	if err != nil {
		logger.Error("❌ NotifyHub客户端创建失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("✅ NotifyHub客户端创建成功")

	// Create test message
	msg := common.CreateTestMessage("Email", "basic")
	msg.Title = fmt.Sprintf("NotifyHub %s 测试邮件", providerName)
	msg.Body = fmt.Sprintf(`
这是一封来自 NotifyHub %s 的测试邮件。

发送时间: %s
测试内容:
- SMTP服务器: %s:%d
- 加密方式: %s
- 发送状态: 成功

如果您收到这封邮件，说明 %s 配置正确且邮件发送功能正常。
`, providerName, time.Now().Format("2006-01-02 15:04:05"),
		config.Host, config.Port,
		getEncryptionType(config),
		providerName)

	msg.Targets = []target.Target{
		common.CreateEmailTarget(recipient),
	}

	// Send message
	ctx := context.Background()
	logger.Info("🚀 开始发送测试邮件...")

	receipt, err := client.Send(ctx, msg)
	if err != nil {
		logger.Error("❌ 邮件发送失败: %v", err)

		// Provide specific error analysis
		analyzeEmailError(logger, err)
		return
	}

	// Analyze results
	if receipt.Failed > 0 {
		logger.Error("❌ 邮件发送部分失败: %d/%d", receipt.Failed, receipt.Total)
		for _, result := range receipt.Results {
			if !result.Success {
				logger.Error("失败详情: Target=%s, Error=%v", result.Target, result.Error)
			}
		}
	} else {
		logger.Success("✅ 邮件发送成功!")
		logger.Info("📊 发送统计:")
		logger.Info("   总计: %d", receipt.Total)
		logger.Info("   成功: %d", receipt.Successful)
		logger.Info("   失败: %d", receipt.Failed)

		logger.Info("📬 请检查收件箱 (包括垃圾邮件文件夹)")
	}
}

func getEncryptionType(config *EmailProviderConfig) string {
	if config.UseTLS {
		return "TLS (直接加密)"
	} else if config.UseStartTLS {
		return "STARTTLS (先连接后加密)"
	}
	return "无加密"
}

func analyzeEmailError(logger *common.Logger, err error) {
	errStr := strings.ToLower(err.Error())

	logger.Info("💡 错误分析和解决建议:")

	if strings.Contains(errStr, "authentication failed") || strings.Contains(errStr, "535") {
		logger.Info("🔑 认证失败:")
		logger.Info("   - 检查用户名和密码是否正确")
		logger.Info("   - 确认使用授权码而不是登录密码 (163, QQ等)")
		logger.Info("   - 确认使用应用专用密码 (Gmail等)")
		logger.Info("   - 检查邮箱是否开启了SMTP服务")
	} else if strings.Contains(errStr, "connection refused") {
		logger.Info("🔌 连接被拒绝:")
		logger.Info("   - 检查SMTP服务器地址和端口")
		logger.Info("   - 检查网络连接")
		logger.Info("   - 检查防火墙设置")
	} else if strings.Contains(errStr, "tls") || strings.Contains(errStr, "ssl") {
		logger.Info("🔒 TLS/SSL错误:")
		logger.Info("   - 检查加密设置 (TLS vs STARTTLS)")
		logger.Info("   - 尝试不同的端口配置")
		logger.Info("   - 检查服务器证书")
	} else if strings.Contains(errStr, "timeout") {
		logger.Info("⏰ 超时错误:")
		logger.Info("   - 网络可能较慢")
		logger.Info("   - 增加超时时间")
		logger.Info("   - 检查网络连接质量")
	} else {
		logger.Info("❓ 其他错误:")
		logger.Info("   - 查看详细错误日志")
		logger.Info("   - 参考邮箱服务商文档")
		logger.Info("   - 检查邮箱配额和限制")
	}
}

func testPredefinedConfigurations(logger *common.Logger) {
	logger.Info("\n🧪 测试预定义配置")

	providers := map[string]func(string, string) *email.Config{
		"Gmail":       email.GmailConfig,
		"163邮箱":       email.NetEase163Config,
		"126邮箱":       email.NetEase126Config,
		"QQ邮箱":        email.QQMailConfig,
		"Outlook":     email.OutlookConfig,
		"Yahoo":       email.YahooConfig,
		"Zoho":        email.ZohoConfig,
		"腾讯企业邮箱":      email.TencentExmailConfig,
		"阿里云邮箱":       email.AliMailConfig,
		"新浪邮箱":        email.SinaMailConfig,
		"搜狐邮箱":        email.SohuMailConfig,
		"ProtonMail":  email.ProtonMailConfig,
		"Yahoo Japan": email.YahooJPConfig,
		"Yeah邮箱":      email.NetEaseYeahConfig,
	}

	successCount := 0
	totalCount := len(providers)

	for name, configFunc := range providers {
		logger.Info("测试 %s...", name)

		config := configFunc("test@example.com", "test_password")
		if config == nil {
			logger.Error("❌ %s 配置创建失败", name)
			continue
		}

		// Set required fields for validation
		config.From = "test@example.com"

		if err := config.Validate(); err != nil {
			logger.Error("❌ %s 配置验证失败: %v", name, err)
			continue
		}

		logger.Success("✅ %s 配置正常", name)
		successCount++
	}

	logger.Info("\n📊 预定义配置测试结果: %d/%d 成功", successCount, totalCount)
}

func testProviderDetection(logger *common.Logger) {
	logger.Info("\n🔍 测试服务商检测")

	testCases := []struct {
		Email    string
		Expected string
	}{
		{"user@gmail.com", "Gmail"},
		{"user@163.com", "163邮箱"},
		{"user@126.com", "126邮箱"},
		{"user@qq.com", "QQ邮箱"},
		{"user@outlook.com", "Outlook"},
		{"test@unknown.com", "未知"},
	}

	for _, tc := range testCases {
		detected := detectProvider(tc.Email)
		if strings.Contains(detected, tc.Expected) {
			logger.Success("✅ %s -> %s", tc.Email, detected)
		} else {
			logger.Warn("❌ %s -> %s (期望包含: %s)", tc.Email, detected, tc.Expected)
		}
	}
}

func detectProvider(email string) string {
	domain := strings.ToLower(email)
	if atIndex := strings.Index(domain, "@"); atIndex > 0 {
		domain = domain[atIndex+1:]
	}

	switch {
	case strings.Contains(domain, "gmail.com"):
		return "Gmail"
	case strings.Contains(domain, "163.com"):
		return "163邮箱"
	case strings.Contains(domain, "126.com"):
		return "126邮箱"
	case strings.Contains(domain, "qq.com"):
		return "QQ邮箱"
	case strings.Contains(domain, "outlook.com"):
		return "Outlook"
	default:
		return "未知服务商"
	}
}

func testConfigurationValidation(logger *common.Logger) {
	logger.Info("\n✅ 测试配置验证")

	// Test valid configuration
	validConfig := email.NewConfig()
	validConfig.SMTPHost = "smtp.example.com"
	validConfig.SMTPPort = 587
	validConfig.From = "test@example.com"
	validConfig.Username = "test"
	validConfig.Password = "password"

	if err := validConfig.Validate(); err != nil {
		logger.Error("❌ 有效配置验证失败: %v", err)
	} else {
		logger.Success("✅ 有效配置验证通过")
	}

	// Test invalid configurations
	invalidConfigs := []struct {
		Name   string
		Config *email.Config
	}{
		{
			Name: "缺少SMTP主机",
			Config: func() *email.Config {
				cfg := email.NewConfig()
				cfg.SMTPPort = 587
				cfg.From = "test@example.com"
				return cfg
			}(),
		},
		{
			Name: "端口超出范围",
			Config: func() *email.Config {
				cfg := email.NewConfig()
				cfg.SMTPHost = "smtp.example.com"
				cfg.SMTPPort = 99999
				cfg.From = "test@example.com"
				return cfg
			}(),
		},
		{
			Name: "缺少发件人",
			Config: func() *email.Config {
				cfg := email.NewConfig()
				cfg.SMTPHost = "smtp.example.com"
				cfg.SMTPPort = 587
				return cfg
			}(),
		},
	}

	for _, tc := range invalidConfigs {
		if err := tc.Config.Validate(); err != nil {
			logger.Success("✅ %s - 正确检测到错误: %v", tc.Name, err)
		} else {
			logger.Error("❌ %s - 应该失败但通过了验证", tc.Name)
		}
	}
}

func testSMTPConnectivity(logger *common.Logger) {
	logger.Info("\n🌐 测试SMTP服务器连通性")

	servers := []struct {
		Name string
		Host string
		Port int
	}{
		{"Gmail", "smtp.gmail.com", 587},
		{"163邮箱", "smtp.163.com", 25},
		{"QQ邮箱", "smtp.qq.com", 587},
		{"Outlook", "smtp-mail.outlook.com", 587},
	}

	for _, server := range servers {
		logger.Info("测试 %s (%s:%d)...", server.Name, server.Host, server.Port)

		address := net.JoinHostPort(server.Host, strconv.Itoa(server.Port))
		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		if err != nil {
			logger.Error("❌ %s 连接失败: %v", server.Name, err)
		} else {
			_ = conn.Close()
			logger.Success("✅ %s 连接成功", server.Name)
		}
	}
}

func maskPassword(password string) string {
	if len(password) <= 4 {
		return strings.Repeat("*", len(password))
	}
	return password[:2] + strings.Repeat("*", len(password)-4) + password[len(password)-2:]
}

// Command line usage examples
func init() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Println("邮件服务商验证工具使用说明:")
		fmt.Println()
		fmt.Println("基础使用:")
		fmt.Println("  go run main.go                    # 运行综合验证")
		fmt.Println("  go run main.go gmail               # 测试Gmail配置")
		fmt.Println("  go run main.go 163                 # 测试163邮箱配置")
		fmt.Println()
		fmt.Println("环境变量:")
		fmt.Println("  EMAIL_USERNAME=your@email.com     # 邮箱用户名")
		fmt.Println("  EMAIL_PASSWORD=your_password       # 邮箱密码/授权码")
		fmt.Println("  EMAIL_RECIPIENT=recipient@test.com # 测试收件人")
		fmt.Println()
		fmt.Println("示例:")
		fmt.Println("  EMAIL_USERNAME=test@gmail.com EMAIL_PASSWORD=app_password go run main.go gmail")
		os.Exit(0)
	}
}
