// Package main demonstrates email provider detection and configuration for NotifyHub
package main

import (
	"strings"

	"github.com/kart/notifyhub/examples/common"
	"github.com/kart/notifyhub/pkg/notifyhub"
)

func main() {
	// Print header
	common.PrintHeader("Email Provider", "Configuration Test")

	// Create logger
	logger := common.NewLogger(true)

	logger.Info("🔍 邮件服务商配置检测工具")
	logger.Info("支持的邮件服务商：Gmail, 163, 126, QQ, Outlook, Yahoo 等")

	// Test different email providers
	providers := []EmailProvider{
		{
			Name:        "Gmail",
			Host:        "smtp.gmail.com",
			Port:        587,
			Username:    "your_gmail@gmail.com",
			Password:    "your_app_password",
			Description: "需要开启两步验证并使用应用专用密码",
		},
		{
			Name:        "163邮箱",
			Host:        "smtp.163.com",
			Port:        25,
			Username:    "your_email@163.com",
			Password:    "your_auth_code",
			Description: "需要在163邮箱设置中开启SMTP服务并设置授权码",
		},
		{
			Name:        "126邮箱",
			Host:        "smtp.126.com",
			Port:        25,
			Username:    "your_email@126.com",
			Password:    "your_auth_code",
			Description: "需要在126邮箱设置中开启SMTP服务并设置授权码",
		},
		{
			Name:        "QQ邮箱",
			Host:        "smtp.qq.com",
			Port:        587,
			Username:    "your_email@qq.com",
			Password:    "your_auth_code",
			Description: "需要在QQ邮箱设置中开启SMTP服务并获取授权码",
		},
		{
			Name:        "Outlook",
			Host:        "smtp-mail.outlook.com",
			Port:        587,
			Username:    "your_email@outlook.com",
			Password:    "your_password",
			Description: "可以直接使用登录密码，也可以使用应用密码",
		},
	}

	// Display configuration for each provider
	for i, provider := range providers {
		displayProviderConfig(logger, i+1, provider)
	}

	// Test connection capabilities
	logger.Info("\n🔧 测试当前配置的连接能力...")

	// Create test configuration (using Gmail as example)
	config := common.DefaultExampleConfig()

	// You can modify these for testing
	config.Email.Host = "smtp.gmail.com"
	config.Email.Port = 587
	config.Email.Username = "test@gmail.com" // 这里只是示例，不会真实连接
	config.Email.Password = "test_password"
	config.Email.From = "test@gmail.com"

	detectAndValidateProvider(logger, config)
}

type EmailProvider struct {
	Name        string
	Host        string
	Port        int
	Username    string
	Password    string
	Description string
}

func displayProviderConfig(logger *common.Logger, index int, provider EmailProvider) {
	logger.Info("\n📧 %d. %s 配置", index, provider.Name)
	logger.Info("   SMTP服务器: %s", provider.Host)
	logger.Info("   端口: %d", provider.Port)
	logger.Info("   用户名示例: %s", provider.Username)
	logger.Info("   密码示例: %s", provider.Password)
	logger.Info("   说明: %s", provider.Description)

	// Determine encryption method
	encryption := getEncryptionMethod(provider.Host, provider.Port)
	logger.Info("   加密方式: %s", encryption)
}

func getEncryptionMethod(host string, port int) string {
	lowerHost := strings.ToLower(host)

	if port == 465 {
		return "SSL/TLS (直接加密连接)"
	}

	if port == 587 || port == 25 {
		if strings.Contains(lowerHost, "gmail") {
			return "STARTTLS (先连接后加密)"
		}
		if strings.Contains(lowerHost, "163.com") || strings.Contains(lowerHost, "126.com") {
			return "STARTTLS (先连接后加密) - 推荐端口25"
		}
		if strings.Contains(lowerHost, "qq.com") {
			return "STARTTLS (先连接后加密) - 推荐端口587"
		}
		return "STARTTLS (先连接后加密)"
	}

	return "未知加密方式"
}

func detectAndValidateProvider(logger *common.Logger, config *common.ExampleConfig) {
	host := strings.ToLower(config.Email.Host)

	logger.Info("检测邮件服务商: %s", config.Email.Host)

	var providerName string
	var recommendations []string

	if strings.Contains(host, "gmail.com") {
		providerName = "Gmail"
		recommendations = []string{
			"确保开启了两步验证",
			"使用应用专用密码而不是账户密码",
			"端口587 + STARTTLS是推荐配置",
		}
	} else if strings.Contains(host, "163.com") {
		providerName = "163邮箱"
		recommendations = []string{
			"在163邮箱设置中开启SMTP服务",
			"使用授权码，不是登录密码",
			"端口25 + STARTTLS是常用配置",
			"也可以使用端口587",
		}
	} else if strings.Contains(host, "126.com") {
		providerName = "126邮箱"
		recommendations = []string{
			"在126邮箱设置中开启SMTP服务",
			"使用授权码，不是登录密码",
			"端口25 + STARTTLS是常用配置",
		}
	} else if strings.Contains(host, "qq.com") {
		providerName = "QQ邮箱"
		recommendations = []string{
			"在QQ邮箱设置中开启SMTP服务",
			"使用授权码，不是登录密码",
			"端口587 + STARTTLS是推荐配置",
		}
	} else if strings.Contains(host, "outlook") || strings.Contains(host, "hotmail") {
		providerName = "Outlook/Hotmail"
		recommendations = []string{
			"可以使用登录密码",
			"也可以设置应用密码提高安全性",
			"端口587 + STARTTLS是推荐配置",
		}
	} else {
		providerName = "未知服务商"
		recommendations = []string{
			"使用通用SMTP配置",
			"通常端口587 + STARTTLS可以工作",
			"请确认服务商的SMTP设置要求",
		}
	}

	logger.Success("识别为: %s", providerName)
	logger.Info("📋 配置建议:")
	for i, rec := range recommendations {
		logger.Info("   %d. %s", i+1, rec)
	}

	// Create NotifyHub config and test it
	logger.Info("\n🔧 NotifyHub配置验证...")
	cfg := config.CreateEmailConfig()

	logger.Debug("SMTP配置: Host=%s, Port=%d, UseTLS=%v",
		cfg.Email.Host, cfg.Email.Port, cfg.Email.UseTLS)

	// Try to create client (this will validate configuration)
	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("NotifyHub客户端创建失败: %v", err)
		logger.Info("请检查配置是否正确")
		return
	}
	defer client.Close()

	logger.Success("NotifyHub客户端创建成功！")
	logger.Info("✅ 配置验证通过，可以尝试发送邮件")
}