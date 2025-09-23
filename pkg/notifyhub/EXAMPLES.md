# NotifyHub 使用示例

本文档提供了 NotifyHub 的详细使用示例，涵盖从基础用法到高级特性的各种场景。

## 目录

- [快速开始](#快速开始)
- [基础示例](#基础示例)
- [平台配置](#平台配置)
- [消息构建](#消息构建)
- [目标管理](#目标管理)
- [错误处理](#错误处理)
- [健康检查](#健康检查)
- [异步发送](#异步发送)
- [平台扩展](#平台扩展)
- [高级用法](#高级用法)

## 快速开始

### 最简单的示例

```go
package main

import (
    "context"
    "log"

    "github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
    // 创建 Hub（使用测试配置）
    hub, err := notifyhub.NewHub(notifyhub.WithTestDefaults())
    if err != nil {
        log.Fatal(err)
    }
    defer hub.Close(context.Background())

    // 创建并发送消息
    msg := notifyhub.NewMessage("Hello World").
        Body("这是我的第一条通知").
        AddTarget(notifyhub.NewEmailTarget("user@example.com")).
        Build()

    receipt, err := hub.Send(context.Background(), msg)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("发送结果: %s", receipt.Status)
}
```

## 基础示例

### 1. 创建 Hub 并发送简单消息

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
    // 创建配置了飞书的 Hub
    hub, err := notifyhub.NewHub(
        notifyhub.WithFeishu(
            "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook",
            "your-secret",
        ),
        notifyhub.WithTimeout(30*time.Second),
    )
    if err != nil {
        log.Fatal("创建 Hub 失败:", err)
    }
    defer hub.Close(context.Background())

    // 创建消息
    msg := notifyhub.NewMessage("系统通知").
        Body("服务器维护已完成，系统已恢复正常运行").
        Priority(notifyhub.PriorityNormal).
        AddTarget(notifyhub.NewFeishuUserTarget("ou_your_user_id")).
        Build()

    // 发送消息
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    receipt, err := hub.Send(ctx, msg)
    if err != nil {
        log.Fatal("发送失败:", err)
    }

    fmt.Printf("发送成功！消息ID: %s, 状态: %s\n", receipt.MessageID, receipt.Status)
    fmt.Printf("成功: %d, 失败: %d, 总计: %d\n",
        receipt.Successful, receipt.Failed, receipt.Total)
}
```

### 2. 批量发送到多个目标

```go
package main

import (
    "context"
    "log"

    "github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
    hub, err := notifyhub.NewHub(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithEmail("smtp.gmail.com", 587, "user", "pass", "from@company.com", true, 30*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer hub.Close(context.Background())

    // 创建多个目标
    targets := []notifyhub.Target{
        notifyhub.NewEmailTarget("admin@company.com"),
        notifyhub.NewEmailTarget("devops@company.com"),
        notifyhub.NewFeishuUserTarget("user1"),
        notifyhub.NewFeishuGroupTarget("group1"),
    }

    // 创建消息
    msg := notifyhub.NewAlert("数据库连接异常").
        Body("数据库连接数已达到最大限制，请立即检查").
        AddTargets(targets...).
        Build()

    receipt, err := hub.Send(context.Background(), msg)
    if err != nil {
        log.Fatal(err)
    }

    // 打印详细结果
    for _, result := range receipt.Results {
        status := "成功"
        if !result.Success {
            status = fmt.Sprintf("失败: %s", result.Error)
        }
        log.Printf("平台: %s, 目标: %s, 状态: %s",
            result.Platform, result.Target, status)
    }
}
```

## 平台配置

### 1. 飞书配置

```go
// 基础配置
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu("webhook-url", "secret"),
)

// 从环境变量配置
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuFromMap(map[string]interface{}{
        "webhook_url": os.Getenv("FEISHU_WEBHOOK_URL"),
        "secret":      os.Getenv("FEISHU_SECRET"),
        "timeout":     "30s",
    }),
)

// 配置关键词
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuKeywords([]string{"紧急", "警告", "错误"}),
)
```

### 2. 邮件配置

```go
// Gmail 配置
hub, err := notifyhub.NewHub(
    notifyhub.WithEmail(
        "smtp.gmail.com", 587,
        "your-email@gmail.com", "your-password",
        "notifications@company.com",
        true, // 使用 TLS
        30*time.Second,
    ),
)

// 企业邮箱配置
hub, err := notifyhub.NewHub(
    notifyhub.WithEmailFromMap(map[string]interface{}{
        "smtp_host":     "mail.company.com",
        "smtp_port":     587,
        "smtp_username": "notifications@company.com",
        "smtp_password": "password",
        "smtp_from":     "no-reply@company.com",
        "smtp_tls":      true,
        "timeout":       "45s",
    }),
)
```

### 3. 多平台配置

```go
hub, err := notifyhub.NewHub(
    // 飞书
    notifyhub.WithFeishu("feishu-webhook", "feishu-secret"),

    // 邮件
    notifyhub.WithEmail("smtp.gmail.com", 587, "user", "pass", "from@company.com", true, 30*time.Second),

    // SMS
    notifyhub.WithSMS("aliyun", "api-key", "company"),

    // 自定义平台
    notifyhub.WithPlatformConfig("slack", map[string]interface{}{
        "webhook_url": "https://example.com/slack/webhook/your-id",
        "channel":     "#alerts",
    }),

    // 全局设置
    notifyhub.WithTimeout(45*time.Second),
)
```

## 消息构建

### 1. 基础消息

```go
// 普通消息
msg := notifyhub.NewMessage("标题").
    Body("消息内容").
    Build()

// 警告消息（高优先级）
msg := notifyhub.NewAlert("磁盘空间不足").
    Body("服务器 /var 分区使用率已达 90%").
    Build()

// 紧急消息（最高优先级）
msg := notifyhub.NewUrgent("服务宕机").
    Body("支付服务当前不可用，请立即处理").
    Build()
```

### 2. 带元数据的消息

```go
msg := notifyhub.NewMessage("系统监控报告").
    Body("CPU 使用率: {{cpu_usage}}%, 内存使用率: {{memory_usage}}%").
    WithVariable("cpu_usage", "85").
    WithVariable("memory_usage", "72").
    WithMetadata("server", "web-01").
    WithMetadata("alert_type", "performance").
    Build()
```

### 3. 富文本消息

```go
// Markdown 格式
msg := notifyhub.NewMessage("代码发布通知").
    Body(`
## 发布信息
- **版本**: v2.1.0
- **时间**: 2024-01-15 14:30
- **分支**: main

### 更新内容
1. 修复登录问题
2. 优化性能
3. 新增用户管理功能

详情请查看 [发布说明](https://github.com/company/project/releases/v2.1.0)
    `).
    Format("markdown").
    Build()
```

### 4. 平台特定消息

```go
// 飞书卡片消息
msg := notifyhub.NewMessage("系统状态").
    Body("系统运行正常").
    WithPlatformData("feishu", map[string]interface{}{
        "card": map[string]interface{}{
            "elements": []map[string]interface{}{
                {
                    "tag": "div",
                    "text": map[string]interface{}{
                        "content": "**系统状态**: 正常\n**CPU**: 45%\n**内存**: 62%",
                        "tag": "lark_md",
                    },
                },
                {
                    "tag": "action",
                    "actions": []map[string]interface{}{
                        {
                            "tag": "button",
                            "text": map[string]interface{}{
                                "content": "查看详情",
                                "tag": "plain_text",
                            },
                            "url": "https://monitor.company.com",
                            "type": "default",
                        },
                    },
                },
            },
        },
    }).
    Build()
```

## 目标管理

### 1. 不同类型的目标

```go
// 邮件目标
emailTarget := notifyhub.NewEmailTarget("user@company.com")

// 电话目标
phoneTarget := notifyhub.NewPhoneTarget("+86-13800138000")

// 飞书用户
feishuUser := notifyhub.NewFeishuUserTarget("ou_user_id")

// 飞书群组
feishuGroup := notifyhub.NewFeishuGroupTarget("oc_group_id")

// Webhook
webhookTarget := notifyhub.NewWebhookTarget("https://api.example.com/notifications")

// 自动检测
autoTarget := notifyhub.AutoDetectTarget("user@company.com") // 自动识别为邮件
```

### 2. 动态目标列表

```go
func getNotificationTargets(alertLevel string) []notifyhub.Target {
    var targets []notifyhub.Target

    switch alertLevel {
    case "low":
        targets = append(targets, notifyhub.NewEmailTarget("devops@company.com"))
    case "medium":
        targets = append(targets,
            notifyhub.NewEmailTarget("devops@company.com"),
            notifyhub.NewFeishuGroupTarget("devops-group"),
        )
    case "high":
        targets = append(targets,
            notifyhub.NewEmailTarget("devops@company.com"),
            notifyhub.NewEmailTarget("admin@company.com"),
            notifyhub.NewFeishuGroupTarget("devops-group"),
            notifyhub.NewFeishuUserTarget("oncall-engineer"),
        )
    case "critical":
        targets = append(targets,
            notifyhub.NewEmailTarget("devops@company.com"),
            notifyhub.NewEmailTarget("admin@company.com"),
            notifyhub.NewEmailTarget("cto@company.com"),
            notifyhub.NewFeishuGroupTarget("emergency"),
            notifyhub.NewPhoneTarget("+86-13800138000"), // 紧急电话
        )
    }

    return targets
}

// 使用
targets := getNotificationTargets("critical")
msg := notifyhub.NewUrgent("生产环境故障").
    Body("数据库集群宕机").
    AddTargets(targets...).
    Build()
```

## 错误处理

### 1. 基础错误处理

```go
receipt, err := hub.Send(ctx, msg)
if err != nil {
    log.Printf("发送失败: %v", err)
    return
}

switch receipt.Status {
case "success":
    log.Printf("消息发送成功")
case "partial":
    log.Printf("部分发送成功: %d/%d", receipt.Successful, receipt.Total)
    // 检查失败的平台
    for _, result := range receipt.Results {
        if !result.Success {
            log.Printf("平台 %s 发送失败: %s", result.Platform, result.Error)
        }
    }
case "failed":
    log.Printf("消息发送完全失败")
}
```

### 2. 重试机制

```go
func sendWithRetry(hub notifyhub.Hub, msg *notifyhub.Message, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        receipt, err := hub.Send(ctx, msg)
        cancel()

        if err == nil && receipt.Status == "success" {
            log.Printf("发送成功，重试次数: %d", i)
            return nil
        }

        if i < maxRetries-1 {
            backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
            log.Printf("发送失败，%v 后重试...", backoff)
            time.Sleep(backoff)
        }
    }

    return fmt.Errorf("重试 %d 次后仍然失败", maxRetries)
}

// 使用
err := sendWithRetry(hub, msg, 3)
if err != nil {
    log.Fatal(err)
}
```

### 3. 平台特定错误处理

```go
receipt, err := hub.Send(ctx, msg)
if err != nil {
    return err
}

for _, result := range receipt.Results {
    if !result.Success {
        switch result.Platform {
        case "feishu":
            if strings.Contains(result.Error, "invalid webhook") {
                log.Printf("飞书 webhook 配置错误，请检查 URL")
            }
        case "email":
            if strings.Contains(result.Error, "authentication failed") {
                log.Printf("邮件认证失败，请检查用户名密码")
            }
        default:
            log.Printf("平台 %s 未知错误: %s", result.Platform, result.Error)
        }
    }
}
```

## 健康检查

### 1. 基础健康检查

```go
health, err := hub.Health(context.Background())
if err != nil {
    log.Printf("健康检查失败: %v", err)
    return
}

if health.Healthy {
    log.Printf("系统状态正常")
} else {
    log.Printf("系统状态异常: %s", health.Status)
}

// 检查各平台状态
for platform, status := range health.Platforms {
    if status.Available {
        log.Printf("平台 %s: 正常", platform)
    } else {
        log.Printf("平台 %s: 异常 - %s", platform, status.Status)
        if len(status.Details) > 0 {
            for key, value := range status.Details {
                log.Printf("  %s: %s", key, value)
            }
        }
    }
}
```

### 2. 定期健康检查

```go
func startHealthMonitor(hub notifyhub.Hub, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            health, err := hub.Health(ctx)
            cancel()

            if err != nil {
                log.Printf("健康检查失败: %v", err)
                continue
            }

            if !health.Healthy {
                // 发送警告消息
                alertMsg := notifyhub.NewAlert("NotifyHub 健康检查异常").
                    Body(fmt.Sprintf("系统状态: %s", health.Status)).
                    AddTarget(notifyhub.NewEmailTarget("admin@company.com")).
                    Build()

                // 这里需要另一个通知渠道或降级机制
                log.Printf("系统异常，需要人工干预")
            }
        }
    }
}

// 启动监控
go startHealthMonitor(hub, 5*time.Minute)
```

## 异步发送

### 1. 基础异步发送

```go
// 异步发送消息
asyncReceipt, err := hub.SendAsync(context.Background(), msg)
if err != nil {
    log.Printf("提交失败: %v", err)
    return
}

log.Printf("消息已提交: ID=%s, 状态=%s, 提交时间=%v",
    asyncReceipt.MessageID, asyncReceipt.Status, asyncReceipt.QueuedAt)

// 注意：异步发送立即返回，不等待实际发送完成
```

### 2. 批量异步发送

```go
func sendBulkNotifications(hub notifyhub.Hub, users []User, title, body string) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(users))

    for _, user := range users {
        wg.Add(1)
        go func(u User) {
            defer wg.Done()

            msg := notifyhub.NewMessage(title).
                Body(body).
                AddTarget(notifyhub.NewEmailTarget(u.Email)).
                Build()

            _, err := hub.SendAsync(context.Background(), msg)
            if err != nil {
                errors <- err
            }
        }(user)
    }

    wg.Wait()
    close(errors)

    // 收集错误
    var allErrors []error
    for err := range errors {
        allErrors = append(allErrors, err)
    }

    if len(allErrors) > 0 {
        return fmt.Errorf("批量发送失败，错误数: %d", len(allErrors))
    }

    return nil
}
```

## 平台扩展

### 1. 实现自定义平台

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "bytes"
    "encoding/json"

    "github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// SlackSender 实现 Slack 通知
type SlackSender struct {
    webhookURL string
    channel    string
}

func (s *SlackSender) Name() string {
    return "slack"
}

func (s *SlackSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
    results := make([]*platform.SendResult, len(targets))

    for i, target := range targets {
        result := &platform.SendResult{
            Target: target,
        }

        // 构建 Slack 消息
        slackMsg := map[string]interface{}{
            "text":    msg.Title,
            "channel": s.channel,
            "attachments": []map[string]interface{}{
                {
                    "color": "good",
                    "text":  msg.Body,
                },
            },
        }

        jsonData, _ := json.Marshal(slackMsg)

        req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
        if err != nil {
            result.Success = false
            result.Error = err.Error()
            results[i] = result
            continue
        }

        req.Header.Set("Content-Type", "application/json")

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            result.Success = false
            result.Error = err.Error()
            results[i] = result
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode == 200 {
            result.Success = true
            result.MessageID = fmt.Sprintf("slack-%d", i)
        } else {
            result.Success = false
            result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
        }

        results[i] = result
    }

    return results, nil
}

func (s *SlackSender) ValidateTarget(target platform.Target) error {
    if target.Platform != "slack" {
        return fmt.Errorf("不支持的平台: %s", target.Platform)
    }
    return nil
}

func (s *SlackSender) GetCapabilities() platform.Capabilities {
    return platform.Capabilities{
        Name:                 "slack",
        SupportedTargetTypes: []string{"channel", "user"},
        SupportedFormats:     []string{"text", "markdown"},
        MaxMessageSize:       4000,
        SupportsRichContent:  true,
    }
}

func (s *SlackSender) IsHealthy(ctx context.Context) error {
    // 简单的健康检查
    return nil
}

func (s *SlackSender) Close() error {
    // 清理资源
    return nil
}

// 工厂函数
func NewSlackSender(config map[string]interface{}) (platform.ExternalSender, error) {
    webhookURL, ok := config["webhook_url"].(string)
    if !ok {
        return nil, fmt.Errorf("缺少 webhook_url 配置")
    }

    channel, _ := config["channel"].(string)
    if channel == "" {
        channel = "#general"
    }

    return &SlackSender{
        webhookURL: webhookURL,
        channel:    channel,
    }, nil
}

func init() {
    // 注册平台
    platform.RegisterPlatform("slack", NewSlackSender)
}
```

### 2. 使用自定义平台

```go
// 使用自定义 Slack 平台
hub, err := notifyhub.NewHub(
    notifyhub.WithPlatformConfig("slack", map[string]interface{}{
        "webhook_url": "https://example.com/slack/webhook/your-token",
        "channel":     "#alerts",
    }),
)

msg := notifyhub.NewAlert("系统告警").
    Body("数据库连接异常").
    AddTarget(notifyhub.NewTarget("channel", "#ops", "slack")).
    Build()

receipt, err := hub.Send(context.Background(), msg)
```

## 高级用法

### 1. 条件性发送

```go
type AlertLevel int

const (
    AlertLevelInfo AlertLevel = iota
    AlertLevelWarning
    AlertLevelError
    AlertLevelCritical
)

func sendAlert(hub notifyhub.Hub, level AlertLevel, title, body string) error {
    var priority notifyhub.Priority
    var targets []notifyhub.Target

    switch level {
    case AlertLevelInfo:
        priority = notifyhub.PriorityLow
        targets = []notifyhub.Target{
            notifyhub.NewEmailTarget("logs@company.com"),
        }
    case AlertLevelWarning:
        priority = notifyhub.PriorityNormal
        targets = []notifyhub.Target{
            notifyhub.NewEmailTarget("devops@company.com"),
            notifyhub.NewFeishuGroupTarget("devops"),
        }
    case AlertLevelError:
        priority = notifyhub.PriorityHigh
        targets = []notifyhub.Target{
            notifyhub.NewEmailTarget("devops@company.com"),
            notifyhub.NewEmailTarget("admin@company.com"),
            notifyhub.NewFeishuGroupTarget("devops"),
        }
    case AlertLevelCritical:
        priority = notifyhub.PriorityUrgent
        targets = []notifyhub.Target{
            notifyhub.NewEmailTarget("devops@company.com"),
            notifyhub.NewEmailTarget("admin@company.com"),
            notifyhub.NewEmailTarget("ceo@company.com"),
            notifyhub.NewFeishuGroupTarget("emergency"),
            notifyhub.NewPhoneTarget("+86-13800138000"),
        }
    }

    msg := notifyhub.NewMessage(title).
        Body(body).
        Priority(priority).
        AddTargets(targets...).
        WithMetadata("alert_level", fmt.Sprintf("%d", level)).
        Build()

    receipt, err := hub.Send(context.Background(), msg)
    if err != nil {
        return err
    }

    if receipt.Status != "success" {
        return fmt.Errorf("发送失败或部分失败: %s", receipt.Status)
    }

    return nil
}
```

### 2. 模板化消息

```go
type ServerMetrics struct {
    ServerName   string
    CPUUsage     float64
    MemoryUsage  float64
    DiskUsage    float64
    Timestamp    time.Time
}

func sendServerAlert(hub notifyhub.Hub, metrics ServerMetrics) error {
    template := `
## 服务器监控告警

**服务器**: {{.server_name}}
**时间**: {{.timestamp}}

### 资源使用情况
- **CPU**: {{.cpu_usage}}%
- **内存**: {{.memory_usage}}%
- **磁盘**: {{.disk_usage}}%

{{if gt .cpu_usage 80.0}}⚠️ CPU 使用率过高！{{end}}
{{if gt .memory_usage 85.0}}⚠️ 内存使用率过高！{{end}}
{{if gt .disk_usage 90.0}}🚨 磁盘空间不足！{{end}}
    `

    msg := notifyhub.NewAlert("服务器资源告警").
        Body(template).
        Format("markdown").
        WithVariable("server_name", metrics.ServerName).
        WithVariable("cpu_usage", fmt.Sprintf("%.1f", metrics.CPUUsage)).
        WithVariable("memory_usage", fmt.Sprintf("%.1f", metrics.MemoryUsage)).
        WithVariable("disk_usage", fmt.Sprintf("%.1f", metrics.DiskUsage)).
        WithVariable("timestamp", metrics.Timestamp.Format("2006-01-02 15:04:05")).
        AddTarget(notifyhub.NewFeishuGroupTarget("ops")).
        Build()

    receipt, err := hub.Send(context.Background(), msg)
    if err != nil {
        return err
    }

    log.Printf("服务器告警已发送: %s", receipt.MessageID)
    return nil
}
```

### 3. 批量处理和限流

```go
func processBulkNotifications(hub notifyhub.Hub, notifications []NotificationRequest) {
    // 使用限流器控制发送频率
    limiter := time.NewTicker(100 * time.Millisecond) // 每 100ms 发送一条
    defer limiter.Stop()

    // 并发控制
    semaphore := make(chan struct{}, 10) // 最多 10 个并发

    var wg sync.WaitGroup

    for _, notification := range notifications {
        wg.Add(1)

        go func(notif NotificationRequest) {
            defer wg.Done()

            // 获取许可
            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            // 限流
            <-limiter.C

            msg := notifyhub.NewMessage(notif.Title).
                Body(notif.Body).
                AddTarget(notifyhub.NewEmailTarget(notif.Email)).
                Build()

            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()

            receipt, err := hub.Send(ctx, msg)
            if err != nil {
                log.Printf("发送失败 [%s]: %v", notif.Email, err)
                return
            }

            if receipt.Status != "success" {
                log.Printf("发送部分失败 [%s]: %s", notif.Email, receipt.Status)
            }
        }(notification)
    }

    wg.Wait()
    log.Printf("批量发送完成，共处理 %d 条通知", len(notifications))
}
```

### 4. 配置热更新

```go
type ConfigurableHub struct {
    hub    notifyhub.Hub
    config *Config
    mutex  sync.RWMutex
}

func (c *ConfigurableHub) UpdateConfig(newConfig *Config) error {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    // 关闭旧的 hub
    if c.hub != nil {
        c.hub.Close(context.Background())
    }

    // 创建新的 hub
    var options []notifyhub.HubOption

    if newConfig.Feishu.Enabled {
        options = append(options, notifyhub.WithFeishu(
            newConfig.Feishu.WebhookURL,
            newConfig.Feishu.Secret,
        ))
    }

    if newConfig.Email.Enabled {
        options = append(options, notifyhub.WithEmail(
            newConfig.Email.Host,
            newConfig.Email.Port,
            newConfig.Email.Username,
            newConfig.Email.Password,
            newConfig.Email.From,
            newConfig.Email.UseTLS,
            newConfig.Email.Timeout,
        ))
    }

    hub, err := notifyhub.NewHub(options...)
    if err != nil {
        return err
    }

    c.hub = hub
    c.config = newConfig

    log.Printf("配置更新成功")
    return nil
}

func (c *ConfigurableHub) Send(ctx context.Context, msg *notifyhub.Message) (*notifyhub.Receipt, error) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()

    if c.hub == nil {
        return nil, fmt.Errorf("hub 未初始化")
    }

    return c.hub.Send(ctx, msg)
}
```

这些示例展示了 NotifyHub 的各种使用场景，从简单的消息发送到复杂的企业级应用。您可以根据实际需求选择合适的模式和功能。