// Package common provides shared utilities for NotifyHub examples
package common

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// Logger provides example logging utilities
type Logger struct {
	verbose bool
}

// NewLogger creates a new example logger
func NewLogger(verbose bool) *Logger {
	return &Logger{verbose: verbose}
}

// LogMode implements the logger.Logger interface
func (l *Logger) LogMode(level logger.LogLevel) logger.Logger {
	newLogger := &Logger{verbose: l.verbose}
	switch level {
	case logger.Debug:
		newLogger.verbose = true
	case logger.Silent:
		newLogger.verbose = false
	}
	return newLogger
}

// Info logs info messages
func (l *Logger) Info(msg string, args ...any) {
	log.Printf("ℹ️  "+msg, args...)
}

// Success logs success messages
func (l *Logger) Success(msg string, args ...interface{}) {
	log.Printf("✅ "+msg, args...)
}

// Error logs error messages
func (l *Logger) Error(msg string, args ...any) {
	log.Printf("❌ "+msg, args...)
}

// Debug logs debug messages if verbose is enabled
func (l *Logger) Debug(msg string, args ...any) {
	if l.verbose {
		log.Printf("🔍 "+msg, args...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(msg string, args ...any) {
	log.Printf("⚠️  "+msg, args...)
}

// WaitForEnter waits for user to press enter
func WaitForEnter(prompt string) {
	fmt.Print(prompt)
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// ConfirmAction asks user for confirmation
func ConfirmAction(prompt string) bool {
	fmt.Printf("%s (y/N): ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// ValidateConfig checks if configuration is properly set
func ValidateConfig(config *ExampleConfig, platform string) error {
	switch platform {
	case "email":
		if config.Email.Host == "" {
			return fmt.Errorf("邮件服务器地址未设置")
		}
		if config.Email.Username == "" {
			return fmt.Errorf("邮箱用户名未设置")
		}
		if config.Email.Password == "" {
			return fmt.Errorf("邮箱密码未设置")
		}
		if config.Email.From == "" {
			return fmt.Errorf("发件人地址未设置")
		}
	case "feishu":
		if config.Feishu.WebhookURL == "" {
			return fmt.Errorf("飞书Webhook URL未设置")
		}
		if !strings.HasPrefix(config.Feishu.WebhookURL, "https://") {
			return fmt.Errorf("飞书Webhook URL格式不正确")
		}
	case "webhook":
		if config.Webhook.URL == "" {
			return fmt.Errorf("Webhook URL未设置")
		}
		if !strings.HasPrefix(config.Webhook.URL, "http") {
			return fmt.Errorf("Webhook URL格式不正确")
		}
	}
	return nil
}

// CreateTestMessage creates a test message
func CreateTestMessage(platform, messageType string) *message.Message {
	msg := message.New()

	switch messageType {
	case "basic":
		msg.Title = fmt.Sprintf("NotifyHub %s 基础测试", platform)
		msg.Body = fmt.Sprintf("这是一条来自 NotifyHub %s 平台的基础测试消息。\n\n发送时间: %s",
			platform, time.Now().Format("2006-01-02 15:04:05"))
		msg.Format = message.FormatText
		msg.Priority = message.PriorityNormal

	case "html":
		msg.Title = fmt.Sprintf("NotifyHub %s HTML测试", platform)
		msg.Body = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>NotifyHub HTML 测试</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .footer { background-color: #f1f1f1; padding: 10px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>NotifyHub %s 平台测试</h1>
    </div>
    <div class="content">
        <h2>HTML 格式测试</h2>
        <p>这是一条使用 HTML 格式的测试消息。</p>
        <ul>
            <li>✅ 支持丰富格式</li>
            <li>✅ 支持样式设置</li>
            <li>✅ 支持多媒体内容</li>
        </ul>
        <p><strong>发送时间:</strong> %s</p>
    </div>
    <div class="footer">
        <p>此消息由 NotifyHub 发送</p>
    </div>
</body>
</html>`, platform, platform, time.Now().Format("2006-01-02 15:04:05"))
		msg.Format = message.FormatHTML
		msg.Priority = message.PriorityNormal

	case "markdown":
		msg.Title = fmt.Sprintf("NotifyHub %s Markdown测试", platform)
		msg.Body = fmt.Sprintf(`# NotifyHub %s 平台测试

这是一条使用 **Markdown** 格式的测试消息。

## 功能特性

- *文本格式化* 支持
- **粗体** 和 ***斜体*** 文本
- `+"`代码块`"+` 支持

### 列表功能

1. 有序列表项 1
2. 有序列表项 2
3. 有序列表项 3

- 无序列表项 A
- 无序列表项 B
- 无序列表项 C

### 代码示例

`+"```go"+`
msg := message.New()
msg.Title = "测试消息"
msg.Format = message.FormatMarkdown
`+"```"+`

---

**发送时间:** %s

*此消息由 NotifyHub 发送*`, platform, time.Now().Format("2006-01-02 15:04:05"))
		msg.Format = message.FormatMarkdown
		msg.Priority = message.PriorityNormal

	case "urgent":
		msg.Title = fmt.Sprintf("🚨 NotifyHub %s 紧急测试", platform)
		msg.Body = fmt.Sprintf("这是一条 %s 平台的紧急优先级测试消息。\n\n⚠️  请注意这是测试消息，无需处理。\n\n发送时间: %s",
			platform, time.Now().Format("2006-01-02 15:04:05"))
		msg.Format = message.FormatText
		msg.Priority = message.PriorityUrgent

	default:
		return CreateTestMessage(platform, "basic")
	}

	// Add metadata
	msg.Metadata = map[string]interface{}{
		"example_type": messageType,
		"platform":     platform,
		"timestamp":    time.Now().Unix(),
		"version":      "1.0.0",
	}

	return msg
}

// CreateEmailTarget creates an email target
func CreateEmailTarget(email string) target.Target {
	return target.New(target.TargetTypeEmail, email, target.PlatformEmail)
}

// CreateFeishuTarget creates a feishu target
func CreateFeishuTarget() target.Target {
	return target.New(target.TargetTypeWebhook, "feishu", target.PlatformFeishu)
}

// CreateWebhookTarget creates a webhook target
func CreateWebhookTarget(url string) target.Target {
	return target.New(target.TargetTypeWebhook, url, target.PlatformWebhook)
}

// PrintSeparator prints a visual separator
func PrintSeparator(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("  %s\n", title)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()
}

// PrintHeader prints example header
func PrintHeader(platform, example string) {
	fmt.Println()
	fmt.Println("🚀 NotifyHub Examples")
	fmt.Printf("📋 Platform: %s\n", platform)
	fmt.Printf("🔧 Example: %s\n", example)
	fmt.Println(strings.Repeat("-", 30))
	fmt.Println()
}

// CheckConfigurationPrompt prompts user to check configuration
func CheckConfigurationPrompt(platform string) bool {
	fmt.Printf("⚠️  请确保已正确配置 %s 平台的相关参数:\n", platform)

	switch platform {
	case "email":
		fmt.Println("   - SMTP服务器地址和端口")
		fmt.Println("   - 邮箱用户名和密码")
		fmt.Println("   - 发件人和收件人地址")
	case "feishu":
		fmt.Println("   - 飞书群组的Webhook URL")
		fmt.Println("   - 签名密钥(可选)")
		fmt.Println("   - 关键词设置(可选)")
	case "webhook":
		fmt.Println("   - 目标Webhook URL")
		fmt.Println("   - 认证信息(如需要)")
		fmt.Println("   - 请求头设置(如需要)")
	}

	fmt.Println()
	return ConfirmAction("配置已完成，继续运行示例?")
}
