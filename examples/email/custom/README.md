# NotifyHub 自定义邮件功能

NotifyHub 提供了强大的自定义邮件功能，支持模板系统、高级配置、频率限制、追踪等企业级特性。

## 🚀 功能特性

### 📧 邮件模板系统
- **多种模板类型**: 支持 HTML、纯文本、Markdown 格式
- **变量替换**: 灵活的模板变量系统
- **内置模板**: 提供常用的邮件模板（通知、警报、营销等）
- **自定义模板**: 支持从文件加载或代码定义模板
- **模板验证**: 自动验证模板语法和变量

### ⚙️ 高级配置
- **多服务商支持**: 支持 Gmail、163、QQ、企业邮箱等
- **自定义头部**: 添加自定义邮件头
- **域名限制**: 允许/禁止特定域名
- **SSL/TLS配置**: 灵活的加密配置
- **认证方式**: 支持多种SMTP认证方式

### 🚦 频率限制
- **令牌桶算法**: 平滑的频率控制
- **突发处理**: 支持短时间突发流量
- **线程安全**: 支持并发使用
- **实时监控**: 查看当前限制状态

### 📊 监控与追踪
- **发送统计**: 实时统计成功/失败次数
- **性能监控**: 延迟、成功率等指标
- **健康检查**: 服务健康状态监控
- **追踪功能**: 邮件打开、点击追踪（可选）

## 📋 使用指南

### 基础用法

```go
// 1. 创建自定义邮件配置
config := &email.CustomEmailConfig{
    Name:        "my-email-service",
    DisplayName: "我的邮件服务",
    Host:        "smtp.gmail.com",
    Port:        587,
    Username:    "your-email@gmail.com",
    Password:    "your-app-password",
    From:        "your-email@gmail.com",
    FromName:    "Your Company",
    UseTLS:      false,
    UseStartTLS: true,
}

// 2. 创建邮件发送器
sender, err := email.NewCustomEmailSender(config, logger)
if err != nil {
    log.Fatal(err)
}
defer sender.Close()

// 3. 发送邮件
options := &email.CustomEmailOptions{
    Template:   "notification",
    Subject:    "重要通知",
    Body:       "这是一条重要通知",
    Recipients: []string{"user@example.com"},
    Variables: map[string]interface{}{
        "user_name": "张三",
        "company":   "示例公司",
    },
}

result, err := sender.SendCustomEmail(context.Background(), options)
```

### 模板系统

#### 内置模板
NotifyHub 提供了以下内置模板：

1. **notification** - 通知邮件模板
2. **alert** - 警报邮件模板
3. **plain** - 纯文本模板
4. **marketing** - 营销邮件模板

#### 自定义模板

```go
// 创建自定义模板
template := &email.EmailTemplate{
    Name:    "welcome",
    Type:    email.TemplateTypeHTML,
    Subject: "欢迎加入 {{.Variables.company}}！",
    Content: `
<h1>欢迎，{{.Variables.user_name}}！</h1>
<p>感谢您加入我们！</p>
<a href="{{.Variables.activation_url}}">激活账户</a>
`,
    Description: "用户欢迎邮件",
}

// 添加模板
templateMgr.AddTemplate(template)
```

#### 模板变量

模板支持以下变量：

```go
type TemplateData struct {
    // 消息数据
    Title     string                 // 邮件标题
    Body      string                 // 邮件内容
    Priority  string                 // 优先级

    // 系统数据
    Timestamp string                 // 发送时间
    Sender    string                 // 发件人
    Recipient string                 // 收件人

    // 自定义变量
    Variables map[string]interface{} // 模板变量
    Custom    map[string]interface{} // 自定义数据
}
```

### 高级配置

#### 域名限制
```go
config := &email.CustomEmailConfig{
    // 只允许发送到这些域名
    AllowedDomains: []string{"company.com", "partner.com"},

    // 禁止发送到这些域名
    BlockedDomains: []string{"tempmail.com", "spam.com"},
}
```

#### 频率限制
```go
config := &email.CustomEmailConfig{
    RateLimit:       60,                // 60封邮件/分钟
    BurstLimit:      20,                // 突发限制20封
    RateLimitWindow: time.Minute,       // 时间窗口
}
```

#### 追踪功能
```go
config := &email.CustomEmailConfig{
    EnableTracking: true,
    TrackingDomain: "track.company.com",
    UnsubscribeURL: "https://company.com/unsubscribe",
}
```

#### 自定义头部
```go
config := &email.CustomEmailConfig{
    CustomHeaders: map[string]string{
        "X-Company":     "Your Company",
        "X-Department":  "Marketing",
        "X-Priority":    "high",
    },
}
```

## 📁 示例文件

### 运行示例

```bash
# 基础自定义邮件演示
go run main.go

# 高级模板演示
go run template-demo.go
```

### 示例文件说明

- `main.go` - 基础自定义邮件功能演示
- `template-demo.go` - 高级模板使用演示
- `templates/` - 示例模板文件目录
  - `welcome.html` - 欢迎邮件模板
  - `invoice.html` - 账单邮件模板
  - `newsletter.html` - 新闻简报模板
  - `system-alert.txt` - 系统警报模板

## 🎯 使用场景

### 1. 用户通知
```go
// 用户注册成功通知
options := &email.CustomEmailOptions{
    Template:   "welcome",
    Recipients: []string{user.Email},
    Variables: map[string]interface{}{
        "user_name":      user.Name,
        "activation_url": generateActivationURL(user.ID),
    },
}
```

### 2. 系统警报
```go
// 服务器监控警报
options := &email.CustomEmailOptions{
    Template:   "system-alert",
    Recipients: opsTeamEmails,
    Priority:   "urgent",
    Variables: map[string]interface{}{
        "alert_type":  "HIGH_CPU",
        "server_name": "web-01",
        "cpu_usage":   "95%",
    },
}
```

### 3. 营销邮件
```go
// 产品推广邮件
options := &email.CustomEmailOptions{
    Template:   "marketing",
    Recipients: subscriberEmails,
    Variables: map[string]interface{}{
        "promotion_title": "限时优惠",
        "discount_code":   "SAVE20",
        "expires_at":      "2024-01-31",
    },
}
```

### 4. 账单通知
```go
// 账单邮件
options := &email.CustomEmailOptions{
    Template:   "invoice",
    Recipients: []string{customer.Email},
    Variables: map[string]interface{}{
        "customer_name":  customer.Name,
        "invoice_number": invoice.Number,
        "total_amount":   invoice.Total,
        "due_date":       invoice.DueDate,
        "items":          invoice.Items,
    },
}
```

## 🔧 配置最佳实践

### 1. 邮件服务商配置

#### Gmail
```go
config := &email.CustomEmailConfig{
    Host:        "smtp.gmail.com",
    Port:        587,
    UseStartTLS: true,
    AuthMethod:  "plain",
    // 使用应用专用密码
}
```

#### 163邮箱
```go
config := &email.CustomEmailConfig{
    Host:        "smtp.163.com",
    Port:        25,
    UseStartTLS: true,
    AuthMethod:  "plain",
    // 使用授权码
}
```

#### 企业邮箱
```go
config := &email.CustomEmailConfig{
    Host:        "smtp.company.com",
    Port:        587,
    UseStartTLS: true,
    RequireSSL:  true,
    // 企业级安全配置
}
```

### 2. 性能优化

#### 批量发送
```go
// 分批发送大量邮件
const batchSize = 50
for i := 0; i < len(recipients); i += batchSize {
    end := i + batchSize
    if end > len(recipients) {
        end = len(recipients)
    }

    options.Recipients = recipients[i:end]
    result, err := sender.SendCustomEmail(ctx, options)

    // 处理结果和错误
}
```

#### 并发控制
```go
// 使用频率限制控制并发
config := &email.CustomEmailConfig{
    RateLimit:  100, // 每分钟100封
    BurstLimit: 20,  // 突发20封
}
```

### 3. 错误处理

```go
result, err := sender.SendCustomEmail(ctx, options)
if err != nil {
    // 检查是否是可重试的错误
    if emailErr, ok := err.(*email.EmailError); ok {
        if emailErr.IsRetryable() {
            // 等待后重试
            time.Sleep(email.GetRetryDelay(err, retryCount))
            // 重试发送
        }
    }
}

// 检查单个收件人的发送结果
for _, result := range result.Results {
    if !result.Success {
        log.Printf("发送到 %s 失败: %s", result.Recipient, result.Error)
    }
}
```

### 4. 监控和日志

```go
// 获取发送统计
metrics := sender.GetMetrics()
log.Printf("成功: %d, 失败: %d, 成功率: %.2f%%",
    metrics.TotalSent,
    metrics.TotalFailed,
    metrics.SuccessRate)

// 获取健康状态
health := monitor.GetHealthStatus()
if health.Status != "healthy" {
    log.Printf("邮件服务状态异常: %s", health.Status)
    for _, issue := range health.Issues {
        log.Printf("问题: %s - %s", issue.Type, issue.Description)
    }
}
```

## ⚠️ 注意事项

### 安全建议
1. **密码保护**: 不要在代码中硬编码邮箱密码
2. **使用授权码**: Gmail等服务商使用应用专用密码
3. **SSL/TLS**: 生产环境启用加密连接
4. **域名验证**: 使用域名白名单防止误发

### 性能建议
1. **连接复用**: 使用连接池减少连接开销
2. **批量发送**: 大量邮件分批处理
3. **频率控制**: 遵守服务商的发送限制
4. **监控告警**: 设置发送失败率告警

### 合规建议
1. **退订链接**: 营销邮件必须提供退订功能
2. **隐私保护**: 遵守数据保护法规
3. **内容审核**: 避免垃圾邮件内容
4. **发送记录**: 保留发送日志用于审计

## 🔗 相关链接

- [NotifyHub 基础邮件功能](../basic/)
- [多服务商配置](../multi-provider-test/)
- [服务商验证工具](../provider-validation/)
- [163邮箱配置指南](../README-163.md)

---

**技术支持**: 如遇问题请查看项目文档或提交 Issue