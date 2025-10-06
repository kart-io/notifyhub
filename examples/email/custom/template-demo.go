// Package main demonstrates advanced template usage with NotifyHub custom email
package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/kart-io/notifyhub/examples/common"
	"github.com/kart-io/notifyhub/pkg/platforms/email"
)

func runTemplateDemo() {
	// Print header
	common.PrintHeader("Email Templates", "Advanced Demo")

	// Create logger
	logger := common.NewLogger(true)

	logger.Info("🎨 高级邮件模板演示")
	logger.Info("展示如何使用自定义模板发送各种类型的邮件")

	// Get current directory for templates
	templateDir := "./templates"

	// Example 1: Welcome email with template file
	if err := demonstrateWelcomeEmail(logger, templateDir); err != nil {
		logger.Error("欢迎邮件演示失败: %v", err)
	}

	// Example 2: Invoice email with complex data
	if err := demonstrateInvoiceEmail(logger, templateDir); err != nil {
		logger.Error("账单邮件演示失败: %v", err)
	}

	// Example 3: Newsletter with rich content
	if err := demonstrateNewsletterEmail(logger, templateDir); err != nil {
		logger.Error("新闻简报演示失败: %v", err)
	}

	// Example 4: System alert with plain text
	if err := demonstrateSystemAlertEmail(logger, templateDir); err != nil {
		logger.Error("系统警报演示失败: %v", err)
	}

	logger.Success("高级邮件模板演示完成！")
}

// demonstrateWelcomeEmail demonstrates welcome email template
func demonstrateWelcomeEmail(logger *common.Logger, templateDir string) error {
	common.PrintSeparator("欢迎邮件模板演示")

	// Create custom email configuration
	config := &email.CustomEmailConfig{
		Name:        "welcome-demo",
		DisplayName: "欢迎邮件服务",
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "welcome@company.com",
		Password:    "password",
		From:        "welcome@company.com",
		FromName:    "Company Welcome Team",
		UseTLS:      false,
		UseStartTLS: true,

		// Custom headers for welcome emails
		CustomHeaders: map[string]string{
			"X-Email-Type": "welcome",
			"X-Priority":   "high",
		},

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create sender
	sender, err := email.NewCustomEmailSender(config, logger)
	if err != nil {
		logger.Error("创建邮件发送器失败: %v", err)
		return err
	}
	defer func() { _ = sender.Close() }()

	// Load template from file (in real scenario)
	// For demo, we'll create a welcome template programmatically
	welcomeTemplate := &email.EmailTemplate{
		Name:    "welcome-user",
		Type:    email.TemplateTypeHTML,
		Subject: "欢迎加入 {{.Variables.company}}！",
		Content: `
<h1>欢迎，{{.Variables.user_name}}！</h1>
<p>感谢您加入 {{.Variables.company}}！</p>
<p>您的账户已成功创建，现在可以开始使用我们的服务了。</p>
<a href="{{.Variables.activation_url}}" style="background: #007bff; color: white; padding: 10px 20px; text-decoration: none;">激活账户</a>
`,
		Description: "用户欢迎邮件模板",
	}

	// Add template to sender
	templateManager := email.NewTemplateManager(templateDir, logger)
	_ = templateManager.AddTemplate(welcomeTemplate)

	// Prepare email options
	options := &email.CustomEmailOptions{
		RequestID:  fmt.Sprintf("welcome-%d", time.Now().Unix()),
		Template:   "welcome-user",
		Recipients: []string{"newuser@example.com"},
		Variables: map[string]interface{}{
			"user_name":       "张三",
			"company":         "示例科技有限公司",
			"activation_url":  "https://company.com/activate?token=abc123",
			"support_email":   "support@company.com",
			"support_phone":   "400-123-4567",
			"website":         "https://company.com",
			"unsubscribe_url": "https://company.com/unsubscribe",
		},
		CustomData: map[string]interface{}{
			"user_id":           12345,
			"registration_date": time.Now().Format("2006-01-02"),
			"source":            "website",
		},
	}

	logger.Info("📧 准备发送欢迎邮件")
	logger.Info("   收件人: %v", options.Recipients)
	logger.Info("   模板: %s", options.Template)
	logger.Info("   用户: %s", options.Variables["user_name"])

	// In a real scenario, you would call:
	// result, err := sender.SendCustomEmail(context.Background(), options)

	logger.Success("✅ 欢迎邮件模板配置完成")
	return nil
}

// demonstrateInvoiceEmail demonstrates invoice email template
func demonstrateInvoiceEmail(logger *common.Logger, templateDir string) error {
	common.PrintSeparator("账单邮件模板演示")

	// Create billing-specific configuration
	config := &email.CustomEmailConfig{
		Name:           "billing-service",
		DisplayName:    "账单系统",
		Host:           "smtp.billing.com",
		Port:           587,
		Username:       "billing@company.com",
		Password:       "secure-password",
		From:           "billing@company.com",
		FromName:       "Company Billing",
		ReplyToAddress: "billing-support@company.com",
		UseTLS:         false,
		UseStartTLS:    true,

		CustomHeaders: map[string]string{
			"X-Email-Type":     "billing",
			"X-Invoice-System": "v2.0",
		},

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	sender, err := email.NewCustomEmailSender(config, logger)
	if err != nil {
		return err
	}
	defer func() { _ = sender.Close() }()

	// Prepare invoice email options with complex data
	options := &email.CustomEmailOptions{
		RequestID:  fmt.Sprintf("invoice-%d", time.Now().Unix()),
		Template:   "invoice", // This would load from invoice.html template file
		Recipients: []string{"customer@company.com"},
		Variables: map[string]interface{}{
			"customer_name":  "李四",
			"company":        "示例科技有限公司",
			"invoice_number": "INV-2024-001",
			"invoice_date":   "2024-01-01",
			"due_date":       "2024-01-31",
			"billing_period": "2024年1月",
			"total_amount":   "1,299.00",
			"currency":       "CNY",
			"is_overdue":     false,

			// Billing items
			"items": []map[string]interface{}{
				{
					"name":     "云服务器 - 高性能型",
					"quantity": "1",
					"price":    "899.00",
					"subtotal": "899.00",
				},
				{
					"name":     "数据库服务",
					"quantity": "1",
					"price":    "200.00",
					"subtotal": "200.00",
				},
				{
					"name":     "CDN流量包",
					"quantity": "1",
					"price":    "200.00",
					"subtotal": "200.00",
				},
			},

			// Contact info
			"billing_email":   "billing@company.com",
			"billing_phone":   "400-123-4567",
			"support_url":     "https://company.com/support",
			"payment_url":     "https://billing.company.com/pay/INV-2024-001",
			"company_address": "北京市朝阳区示例大厦10层",
		},
		CustomData: map[string]interface{}{
			"customer_id":     "CUST-001",
			"subscription_id": "SUB-001",
			"billing_cycle":   "monthly",
		},
	}

	logger.Info("💰 准备发送账单邮件")
	logger.Info("   客户: %s", options.Variables["customer_name"])
	logger.Info("   账单号: %s", options.Variables["invoice_number"])
	logger.Info("   金额: ¥%s", options.Variables["total_amount"])
	logger.Info("   项目数: %d", len(options.Variables["items"].([]map[string]interface{})))

	logger.Success("✅ 账单邮件模板配置完成")
	return nil
}

// demonstrateNewsletterEmail demonstrates newsletter template
func demonstrateNewsletterEmail(logger *common.Logger, templateDir string) error {
	common.PrintSeparator("新闻简报模板演示")

	config := &email.CustomEmailConfig{
		Name:        "newsletter-service",
		DisplayName: "新闻简报服务",
		Host:        "smtp.newsletter.com",
		Port:        587,
		Username:    "newsletter@company.com",
		Password:    "newsletter-password",
		From:        "newsletter@company.com",
		FromName:    "Company Newsletter",
		UseTLS:      false,
		UseStartTLS: true,

		// Enable tracking for newsletters
		EnableTracking: true,
		TrackingDomain: "track.company.com",
		UnsubscribeURL: "https://company.com/unsubscribe",

		CustomHeaders: map[string]string{
			"X-Email-Type":     "newsletter",
			"List-Unsubscribe": "<https://company.com/unsubscribe>",
		},

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	sender, err := email.NewCustomEmailSender(config, logger)
	if err != nil {
		return err
	}
	defer func() { _ = sender.Close() }()

	// Newsletter with rich content
	options := &email.CustomEmailOptions{
		RequestID: fmt.Sprintf("newsletter-%d", time.Now().Unix()),
		Template:  "newsletter",
		Recipients: []string{
			"subscriber1@example.com",
			"subscriber2@example.com",
			"subscriber3@example.com",
		},
		Body: "欢迎阅读本期技术简报！本期为您带来最新的技术动态、产品更新和行业洞察。",
		Variables: map[string]interface{}{
			"subscriber_name":     "尊敬的订阅者",
			"newsletter_title":    "技术简报",
			"issue_number":        "第42期",
			"publish_date":        "2024年1月15日",
			"company":             "示例科技",
			"company_description": "专注于创新技术解决方案",

			// Featured article
			"featured_article": map[string]interface{}{
				"title":       "人工智能在企业中的应用",
				"description": "探索AI如何改变现代企业的运营方式",
				"url":         "https://blog.company.com/ai-in-enterprise",
			},

			// Articles list
			"articles": []map[string]interface{}{
				{
					"title":     "云原生架构最佳实践",
					"category":  "技术",
					"author":    "张工程师",
					"read_time": "5分钟",
					"excerpt":   "介绍云原生架构的核心概念和实施建议...",
					"url":       "https://blog.company.com/cloud-native",
				},
				{
					"title":     "DevOps文化建设指南",
					"category":  "管理",
					"author":    "李经理",
					"read_time": "8分钟",
					"excerpt":   "如何在团队中建立有效的DevOps文化...",
					"url":       "https://blog.company.com/devops-culture",
				},
			},

			// Statistics
			"stats": []map[string]interface{}{
				{"value": "50+", "label": "新功能发布"},
				{"value": "99.9%", "label": "服务可用性"},
				{"value": "1000+", "label": "活跃用户"},
			},

			// Upcoming events
			"upcoming_events": []map[string]interface{}{
				{
					"title":       "技术分享会",
					"date":        "2024年1月25日",
					"location":    "北京会议中心",
					"description": "云计算与AI技术前沿分享",
				},
			},

			// Social links
			"social_links": []map[string]interface{}{
				{"name": "官方微博", "url": "https://weibo.com/company"},
				{"name": "技术博客", "url": "https://blog.company.com"},
				{"name": "GitHub", "url": "https://github.com/company"},
			},

			// URLs
			"website":         "https://company.com",
			"blog_url":        "https://blog.company.com",
			"contact_url":     "https://company.com/contact",
			"unsubscribe_url": "https://company.com/unsubscribe",
			"preferences_url": "https://company.com/preferences",
			"company_address": "北京市朝阳区示例大厦",
		},
		CustomData: map[string]interface{}{
			"campaign_id": "newsletter-2024-01",
			"segment":     "tech-subscribers",
			"send_time":   time.Now().Format(time.RFC3339),
		},
	}

	logger.Info("📰 准备发送新闻简报")
	logger.Info("   期号: %s", options.Variables["issue_number"])
	logger.Info("   收件人: %d 位订阅者", len(options.Recipients))
	logger.Info("   文章数: %d", len(options.Variables["articles"].([]map[string]interface{})))
	logger.Info("   追踪: 已启用")

	logger.Success("✅ 新闻简报模板配置完成")
	return nil
}

// demonstrateSystemAlertEmail demonstrates system alert template
func demonstrateSystemAlertEmail(logger *common.Logger, templateDir string) error {
	common.PrintSeparator("系统警报邮件演示")

	config := &email.CustomEmailConfig{
		Name:        "alert-system",
		DisplayName: "系统警报服务",
		Host:        "smtp.monitoring.com",
		Port:        587,
		Username:    "alerts@company.com",
		Password:    "alert-password",
		From:        "alerts@company.com",
		FromName:    "System Monitor",
		UseTLS:      false,
		UseStartTLS: true,

		// High priority for alerts
		CustomHeaders: map[string]string{
			"X-Priority":   "1",
			"X-Alert-Type": "system",
			"Importance":   "high",
		},

		// No rate limiting for critical alerts
		RateLimit:  0,
		BurstLimit: 0,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	sender, err := email.NewCustomEmailSender(config, logger)
	if err != nil {
		return err
	}
	defer func() { _ = sender.Close() }()

	// System alert with detailed information
	options := &email.CustomEmailOptions{
		RequestID: fmt.Sprintf("alert-%d", time.Now().Unix()),
		Template:  "system-alert", // Plain text template
		Subject:   "🚨 高CPU使用率警报",
		Body:      "检测到服务器CPU使用率异常，当前使用率为95%，已持续10分钟。",
		Recipients: []string{
			"ops-team@company.com",
			"oncall-engineer@company.com",
			"team-lead@company.com",
		},
		Priority: "urgent",
		Variables: map[string]interface{}{
			"alert_type":        "CPU_HIGH",
			"severity":          "CRITICAL",
			"system_name":       "生产环境监控系统",
			"environment":       "production",
			"server_name":       "web-server-01",
			"server_ip":         "192.168.1.100",
			"service_name":      "nginx",
			"alert_id":          "ALT-2024-001",
			"monitoring_system": "Prometheus",

			// Performance metrics
			"cpu_usage":      "95",
			"memory_usage":   "78",
			"disk_usage":     "45",
			"network_status": "正常",

			// Impact information
			"affected_users":     "约500名用户",
			"affected_services":  "Web服务, API服务",
			"estimated_downtime": "如不处理可能在30分钟内导致服务中断",

			// Troubleshooting
			"recommended_actions": []string{
				"检查当前运行的进程",
				"重启高CPU消耗的服务",
				"检查是否有异常流量",
				"考虑扩容服务器资源",
			},

			// Escalation
			"escalation_required": true,
			"escalation_contact":  "运维经理",
			"escalation_phone":    "138-0000-0000",

			// Error details
			"error_details": "CPU使用率在过去10分钟内从30%急剧上升到95%",
			"recent_changes": []map[string]string{
				{
					"timestamp":   "2024-01-15 14:30",
					"description": "部署了新版本应用",
				},
				{
					"timestamp":   "2024-01-15 14:25",
					"description": "增加了新的定时任务",
				},
			},

			// Links
			"dashboard_url": "https://monitoring.company.com/dashboard",
			"logs_url":      "https://logs.company.com/search",
			"incident_url":  "https://incident.company.com/ALT-2024-001",

			// Contact info
			"ops_email":       "ops@company.com",
			"emergency_phone": "400-emergency",
			"oncall_group":    "运维值班群",
		},
		CustomData: map[string]interface{}{
			"alert_source":  "prometheus",
			"rule_name":     "high_cpu_usage",
			"trigger_time":  time.Now().Format(time.RFC3339),
			"severity_code": 1,
		},
	}

	logger.Info("🚨 准备发送系统警报")
	logger.Info("   警报类型: %s", options.Variables["alert_type"])
	logger.Info("   严重程度: %s", options.Variables["severity"])
	logger.Info("   受影响服务器: %s", options.Variables["server_name"])
	logger.Info("   收件人: %d 位运维人员", len(options.Recipients))
	logger.Info("   优先级: %s", options.Priority)

	logger.Success("✅ 系统警报模板配置完成")
	return nil
}

// Helper function for file path
func getTemplatePath(templateDir, filename string) string {
	return filepath.Join(templateDir, filename)
}

// Suppress unused function warnings for example code
var _ = func() {
	_ = runTemplateDemo
	_ = demonstrateWelcomeEmail
	_ = demonstrateInvoiceEmail
	_ = demonstrateNewsletterEmail
	_ = demonstrateSystemAlertEmail
	_ = getTemplatePath
}
