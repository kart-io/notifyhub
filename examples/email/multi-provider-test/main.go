// Package main demonstrates multi-provider email testing with NotifyHub
package main

import (
	"strings"

	"github.com/kart-io/notifyhub/examples/common"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/email"
)

func main() {
	// Print header
	common.PrintHeader("Email Multi-Provider", "Configuration Test")

	// Create logger
	logger := common.NewLogger(true)

	logger.Info("🔍 多邮件服务商配置与测试工具")
	logger.Info("支持的邮件服务商：Gmail, 163, 126, QQ, Outlook, Yahoo, Zoho, ProtonMail, Alibaba, Tencent Enterprise")

	// Test all supported providers
	providers := []EmailProviderTest{
		{
			Provider: EmailProvider{
				Name:        "Gmail",
				Host:        "smtp.gmail.com",
				Port:        587,
				Username:    "your_gmail@gmail.com",
				Password:    "your_app_password",
				Description: "需要开启两步验证并使用应用专用密码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.GmailConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "163邮箱",
				Host:        "smtp.163.com",
				Port:        25,
				Username:    "your_email@163.com",
				Password:    "your_auth_code",
				Description: "需要在163邮箱设置中开启SMTP服务并设置授权码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.NetEase163Config(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "126邮箱",
				Host:        "smtp.126.com",
				Port:        25,
				Username:    "your_email@126.com",
				Password:    "your_auth_code",
				Description: "需要在126邮箱设置中开启SMTP服务并设置授权码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.NetEase126Config(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "Yeah邮箱",
				Host:        "smtp.yeah.net",
				Port:        25,
				Username:    "your_email@yeah.net",
				Password:    "your_auth_code",
				Description: "需要在Yeah邮箱设置中开启SMTP服务并设置授权码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.NetEaseYeahConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "QQ邮箱",
				Host:        "smtp.qq.com",
				Port:        587,
				Username:    "your_email@qq.com",
				Password:    "your_auth_code",
				Description: "需要在QQ邮箱设置中开启SMTP服务并获取授权码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.QQMailConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "腾讯企业邮箱",
				Host:        "smtp.exmail.qq.com",
				Port:        587,
				Username:    "your_email@company.com",
				Password:    "your_password",
				Description: "腾讯企业邮箱，使用企业邮箱账号和密码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.TencentExmailConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "Outlook/Hotmail",
				Host:        "smtp-mail.outlook.com",
				Port:        587,
				Username:    "your_email@outlook.com",
				Password:    "your_password",
				Description: "可以直接使用登录密码，也可以使用应用密码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.OutlookConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "Yahoo Mail",
				Host:        "smtp.mail.yahoo.com",
				Port:        587,
				Username:    "your_email@yahoo.com",
				Password:    "your_app_password",
				Description: "需要使用应用专用密码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.YahooConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "Yahoo Japan",
				Host:        "smtp.mail.yahoo.co.jp",
				Port:        587,
				Username:    "your_email@yahoo.co.jp",
				Password:    "your_app_password",
				Description: "日本Yahoo邮箱，需要使用应用专用密码",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.YahooJPConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "Zoho Mail",
				Host:        "smtp.zoho.com",
				Port:        587,
				Username:    "your_email@zoho.com",
				Password:    "your_password",
				Description: "Zoho企业邮箱服务",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.ZohoConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "新浪邮箱",
				Host:        "smtp.sina.com",
				Port:        25,
				Username:    "your_email@sina.com",
				Password:    "your_password",
				Description: "新浪邮箱SMTP服务",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.SinaMailConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "搜狐邮箱",
				Host:        "smtp.sohu.com",
				Port:        25,
				Username:    "your_email@sohu.com",
				Password:    "your_password",
				Description: "搜狐邮箱SMTP服务",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.SohuMailConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "阿里云邮箱",
				Host:        "smtp.mxhichina.com",
				Port:        587,
				Username:    "your_email@company.com",
				Password:    "your_password",
				Description: "阿里云企业邮箱服务",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.AliMailConfig(username, password)
			},
		},
		{
			Provider: EmailProvider{
				Name:        "ProtonMail",
				Host:        "127.0.0.1",
				Port:        1025,
				Username:    "your_email@protonmail.com",
				Password:    "your_bridge_password",
				Description: "需要安装ProtonMail Bridge",
				TLSType:     "STARTTLS",
				AuthMethod:  "PLAIN",
			},
			ConfigFunc: func(username, password string) *email.Config {
				return email.ProtonMailConfig(username, password)
			},
		},
	}

	// Display all provider configurations
	displayAllProviders(logger, providers)

	// Test provider detection
	testProviderDetection(logger)

	// Test configuration validation
	testConfigurationValidation(logger, providers)

	// Show connection testing options
	showConnectionTestingOptions(logger)

	logger.Success("多邮件服务商测试工具运行完成！")
}

type EmailProvider struct {
	Name        string
	Host        string
	Port        int
	Username    string
	Password    string
	Description string
	TLSType     string
	AuthMethod  string
}

type EmailProviderTest struct {
	Provider   EmailProvider
	ConfigFunc func(username, password string) *email.Config
}

func displayAllProviders(logger *common.Logger, providers []EmailProviderTest) {
	logger.Info("\n📧 支持的邮件服务商配置列表:")
	logger.Info(strings.Repeat("=", 80))

	for i, providerTest := range providers {
		provider := providerTest.Provider
		logger.Info("\n%d. %s", i+1, provider.Name)
		logger.Info("   SMTP服务器: %s", provider.Host)
		logger.Info("   端口: %d", provider.Port)
		logger.Info("   加密方式: %s", provider.TLSType)
		logger.Info("   认证方式: %s", provider.AuthMethod)
		logger.Info("   用户名示例: %s", provider.Username)
		logger.Info("   密码示例: %s", provider.Password)
		logger.Info("   配置说明: %s", provider.Description)

		// Test configuration creation
		testConfig := providerTest.ConfigFunc("test@example.com", "test_password")
		if testConfig != nil {
			logger.Info("   ✅ 预定义配置可用")
			logger.Debug("   配置详情: Host=%s, Port=%d, TLS=%v, STARTTLS=%v",
				testConfig.SMTPHost, testConfig.SMTPPort, testConfig.UseTLS, testConfig.UseStartTLS)
		} else {
			logger.Warn("   ❌ 预定义配置创建失败")
		}
	}
}

func testProviderDetection(logger *common.Logger) {
	logger.Info("\n🔍 邮件服务商自动检测测试:")
	logger.Info(strings.Repeat("=", 50))

	testCases := []struct {
		Email        string
		ExpectedName string
	}{
		{"user@gmail.com", "Gmail"},
		{"user@163.com", "163邮箱"},
		{"user@126.com", "126邮箱"},
		{"user@yeah.net", "Yeah邮箱"},
		{"user@qq.com", "QQ邮箱"},
		{"user@company.exmail.qq.com", "腾讯企业邮箱"},
		{"user@outlook.com", "Outlook"},
		{"user@hotmail.com", "Outlook"},
		{"user@yahoo.com", "Yahoo"},
		{"user@yahoo.co.jp", "Yahoo Japan"},
		{"user@zoho.com", "Zoho"},
		{"user@sina.com", "新浪邮箱"},
		{"user@sohu.com", "搜狐邮箱"},
		{"user@company.mxhichina.com", "阿里云邮箱"},
		{"user@protonmail.com", "ProtonMail"},
		{"user@unknown-provider.com", "未知服务商"},
	}

	for _, testCase := range testCases {
		detected := detectEmailProvider(testCase.Email)
		if detected == testCase.ExpectedName {
			logger.Success("✅ %s -> %s", testCase.Email, detected)
		} else {
			logger.Warn("❌ %s -> %s (期望: %s)", testCase.Email, detected, testCase.ExpectedName)
		}
	}
}

func detectEmailProvider(email string) string {
	domain := strings.ToLower(email)
	if strings.Contains(domain, "@") {
		parts := strings.Split(domain, "@")
		if len(parts) > 1 {
			domain = parts[1]
		}
	}

	switch {
	case strings.Contains(domain, "gmail.com"):
		return "Gmail"
	case strings.Contains(domain, "163.com"):
		return "163邮箱"
	case strings.Contains(domain, "126.com"):
		return "126邮箱"
	case strings.Contains(domain, "yeah.net"):
		return "Yeah邮箱"
	case strings.Contains(domain, "qq.com") && !strings.Contains(domain, "exmail"):
		return "QQ邮箱"
	case strings.Contains(domain, "exmail.qq.com"):
		return "腾讯企业邮箱"
	case strings.Contains(domain, "outlook.com") || strings.Contains(domain, "hotmail.com"):
		return "Outlook"
	case strings.Contains(domain, "yahoo.com"):
		return "Yahoo"
	case strings.Contains(domain, "yahoo.co.jp"):
		return "Yahoo Japan"
	case strings.Contains(domain, "zoho.com"):
		return "Zoho"
	case strings.Contains(domain, "sina.com"):
		return "新浪邮箱"
	case strings.Contains(domain, "sohu.com"):
		return "搜狐邮箱"
	case strings.Contains(domain, "mxhichina.com"):
		return "阿里云邮箱"
	case strings.Contains(domain, "protonmail.com"):
		return "ProtonMail"
	default:
		return "未知服务商"
	}
}

func testConfigurationValidation(logger *common.Logger, providers []EmailProviderTest) {
	logger.Info("\n🔧 配置验证测试:")
	logger.Info(strings.Repeat("=", 40))

	for _, providerTest := range providers {
		provider := providerTest.Provider
		logger.Info("\n测试 %s 配置验证...", provider.Name)

		// Test with valid configuration
		config := providerTest.ConfigFunc("test@example.com", "test_password")
		config.From = "test@example.com"

		if err := config.Validate(); err != nil {
			logger.Error("❌ %s 配置验证失败: %v", provider.Name, err)
		} else {
			logger.Success("✅ %s 配置验证通过", provider.Name)
		}

		// Test NotifyHub integration
		testNotifyHubIntegration(logger, provider.Name, config)
	}
}

func testNotifyHubIntegration(logger *common.Logger, providerName string, emailConfig *email.Config) {
	// Convert to NotifyHub config
	nhConfig := common.DefaultExampleConfig()
	nhConfig.Email.Host = emailConfig.SMTPHost
	nhConfig.Email.Port = emailConfig.SMTPPort
	nhConfig.Email.Username = emailConfig.Username
	nhConfig.Email.Password = emailConfig.Password
	nhConfig.Email.From = emailConfig.From
	nhConfig.Email.UseTLS = emailConfig.UseTLS

	cfg := nhConfig.CreateEmailConfig()

	// Try to create NotifyHub client
	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("❌ %s NotifyHub客户端创建失败: %v", providerName, err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("✅ %s NotifyHub客户端创建成功", providerName)
}

func showConnectionTestingOptions(logger *common.Logger) {
	logger.Info("\n🚀 连接测试选项:")
	logger.Info(strings.Repeat("=", 30))

	logger.Info("要测试实际的邮件发送，请:")
	logger.Info("1. 复制本文件到新的测试文件")
	logger.Info("2. 修改邮件配置为真实的凭据")
	logger.Info("3. 添加真实的收件人地址")
	logger.Info("4. 取消注释测试代码")

	logger.Info("\n示例代码:")
	logger.Info("```go")
	logger.Info("// 测试Gmail发送")
	logger.Info("testRealSending(logger, \"Gmail\", \"your_real_gmail@gmail.com\", \"your_app_password\", \"recipient@example.com\")")
	logger.Info("```")

	logger.Info("\n⚠️  安全提醒:")
	logger.Info("- 不要在代码中硬编码真实的邮箱密码")
	logger.Info("- 使用环境变量或配置文件存储敏感信息")
	logger.Info("- 对于Gmail等服务，使用应用专用密码而不是账户密码")
}

// Commented out real sending test - uncomment and modify for actual testing
/*
func testRealSending(logger *common.Logger, providerName, username, password, recipient string) {
	logger.Info("\n🧪 测试 %s 真实邮件发送...", providerName)

	config := common.DefaultExampleConfig()

	// Configure based on provider
	switch strings.ToLower(providerName) {
	case "gmail":
		emailConfig := email.GmailConfig(username, password)
		config.Email.Host = emailConfig.SMTPHost
		config.Email.Port = emailConfig.SMTPPort
		config.Email.UseTLS = emailConfig.UseTLS
	case "163邮箱":
		emailConfig := email.NetEase163Config(username, password)
		config.Email.Host = emailConfig.SMTPHost
		config.Email.Port = emailConfig.SMTPPort
		config.Email.UseTLS = emailConfig.UseTLS
	}

	config.Email.Username = username
	config.Email.Password = password
	config.Email.From = username
	config.Email.To = recipient

	// Create client and send test message
	cfg := config.CreateEmailConfig()
	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建客户端失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	// Create test message
	msg := common.CreateTestMessage("Email", "basic")
	msg.Title = fmt.Sprintf("NotifyHub %s 测试邮件", providerName)
	msg.Body = fmt.Sprintf("这是通过 %s 发送的测试邮件，发送时间: %s", providerName, time.Now().Format("2006-01-02 15:04:05"))
	msg.Targets = []target.Target{
		common.CreateEmailTarget(recipient),
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		logger.Error("❌ %s 邮件发送失败: %v", providerName, err)
		return
	}

	if receipt.Failed > 0 {
		logger.Error("❌ %s 邮件发送部分失败: %d/%d", providerName, receipt.Failed, receipt.Total)
		for _, result := range receipt.Results {
			if !result.Success {
				logger.Error("失败详情: %v", result.Error)
			}
		}
	} else {
		logger.Success("✅ %s 邮件发送成功!", providerName)
		logger.Info("收件人: %s", recipient)
		logger.Info("主题: %s", msg.Title)
	}
}
*/
