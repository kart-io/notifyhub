# NotifyHub 外部短信平台示例

本示例演示如何在不修改 NotifyHub 核心代码的情况下，实现外部短信通知平台。该平台支持多种短信服务提供商，具备完整的限流、模板和错误处理功能。

## 📋 功能特性

### 🏢 多提供商支持
- **阿里云短信** - 国内主流短信服务
- **腾讯云短信** - 国内云服务短信平台
- **Twilio** - 国际领先的通信服务
- **Vonage (Nexmo)** - 全球通信API平台
- **Mock Provider** - 测试和开发专用

### 🚦 智能限流
- 按手机号码进行限流控制
- 支持每小时/每天限制设置
- 自动清理过期计数器
- 防止短信轰炸和滥用

### 📋 模板系统
- 动态变量替换：`{{变量名}}`
- 预定义模板管理
- 验证码、欢迎、通知等场景
- 内容长度自动验证

### ⚡ 完整功能
- 异步消息处理
- 费用计算和统计
- 健康状态监控
- 配额管理
- 错误处理和重试

## 🏗️ 架构设计

```
SMS Platform
├── platform.go          # 主平台实现
├── providers.go          # 多提供商实现
├── ratelimiter.go       # 限流器
└── main.go              # 演示程序
```

### 核心接口实现

```go
// 实现 NotifyHub Platform 接口
type Platform struct {
    config   Config
    provider SMSProvider
    limiter  *RateLimiter
}

// 多提供商抽象接口
type SMSProvider interface {
    Name() string
    Send(ctx context.Context, phone, content, templateID string) (*SMSResult, error)
    ValidateCredentials() error
    GetStatus() ProviderStatus
    Close() error
}
```

## 🚀 快速开始

### 1. 运行演示

```bash
# 进入示例目录
cd examples/external-platform-sms

# 运行完整演示
go run main.go
```

### 2. 基础使用

```go
package main

import (
    "context"
    "github.com/kart/notifyhub/examples/external-platform-sms/sms"
    "github.com/kart/notifyhub/pkg/message"
    "github.com/kart/notifyhub/pkg/target"
)

func main() {
    // 创建短信平台配置
    config := sms.Config{
        Provider: sms.ProviderMock,
        Credentials: map[string]string{
            "should_fail": "false",
        },
        Timeout: 30,
    }

    // 创建平台实例
    platform, err := sms.New(config)
    if err != nil {
        panic(err)
    }
    defer platform.Close()

    // 创建消息
    msg := message.New()
    msg.Title = "NotifyHub"
    msg.Body = "这是一条测试短信"

    // 创建目标
    targets := []target.Target{
        sms.CreateTarget("+86 138 0013 8000"),
    }

    // 发送短信
    ctx := context.Background()
    results, err := platform.Send(ctx, msg, targets)
    if err != nil {
        panic(err)
    }

    // 处理结果
    for _, result := range results {
        if result.Success {
            fmt.Printf("✅ 发送成功: %s\n", result.Response)
        } else {
            fmt.Printf("❌ 发送失败: %v\n", result.Error)
        }
    }
}
```

## ⚙️ 配置说明

### 阿里云短信配置

```go
config := sms.Config{
    Provider: sms.ProviderAliyun,
    Credentials: map[string]string{
        "access_key_id":     "LTAI_your_key_id",
        "access_key_secret": "your_access_key_secret",
        "sign_name":         "你的签名",
        "endpoint":          "dysmsapi.aliyuncs.com", // 可选
    },
}
```

### 腾讯云短信配置

```go
config := sms.Config{
    Provider: sms.ProviderTencent,
    Credentials: map[string]string{
        "secret_id":  "AKID_your_secret_id",
        "secret_key": "your_secret_key",
        "app_id":     "1400123456",
        "sign_name":  "你的签名",
    },
}
```

### Twilio 配置

```go
config := sms.Config{
    Provider: sms.ProviderTwilio,
    Credentials: map[string]string{
        "account_sid": "AC_your_account_sid",
        "auth_token":  "your_auth_token",
        "from_number": "+1234567890",
    },
}
```

### 限流配置

```go
config := sms.Config{
    Provider: sms.ProviderMock,
    RateLimit: sms.RateLimitConfig{
        Enabled:    true,
        MaxPerHour: 100,  // 每小时最多100条
        MaxPerDay:  1000, // 每天最多1000条
    },
}
```

### 模板配置

```go
config := sms.Config{
    Provider: sms.ProviderMock,
    Templates: map[string]string{
        "verification": "您的验证码是{{code}}，请在{{minutes}}分钟内使用。",
        "welcome":      "欢迎{{name}}注册我们的服务！",
        "notification": "{{title}}: {{content}}",
    },
}
```

## 📋 模板使用

### 1. 定义模板

```go
templates := map[string]string{
    "verification": "您的验证码是{{code}}，有效期{{minutes}}分钟。",
    "welcome":      "欢迎{{name}}注册！",
}
```

### 2. 使用模板发送

```go
msg := message.New()
msg.Variables = map[string]interface{}{
    "code":    "123456",
    "minutes": "5",
}
msg.Metadata = map[string]interface{}{
    "template": "verification",
}

results, err := platform.Send(ctx, msg, targets)
```

## 🚦 限流管理

### 限流统计查询

```go
// 获取特定手机号的限流统计
stats := limiter.GetStats("+86 138 0013 8000")
fmt.Printf("今日剩余: %d条\n", stats.DailyRemaining)
fmt.Printf("每小时剩余: %d条\n", stats.HourlyRemaining)

// 获取所有手机号的统计
allStats := limiter.GetAllStats()
for phone, stats := range allStats {
    fmt.Printf("%s: 今日已发送 %d条\n", phone, stats.DailyCount)
}
```

### 重置限流计数

```go
// 重置特定手机号的计数器
limiter.Reset("+86 138 0013 8000")
```

## 🔌 NotifyHub 集成

虽然当前 NotifyHub 核心不直接支持外部平台注册，但您可以通过以下方式集成：

### 概念性集成代码

```go
// 未来可能的集成方式
func integrateWithNotifyHub() {
    // 1. 注册平台工厂
    factory := platform.Factory(sms.New)
    client.RegisterPlatform("sms", factory)

    // 2. 配置平台
    smsConfig := sms.Config{
        Provider: sms.ProviderAliyun,
        Credentials: map[string]string{
            "access_key_id": "your_key",
            // ...
        },
    }
    client.SetPlatformConfig("sms", smsConfig)

    // 3. 使用 NotifyHub 发送
    msg := message.New()
    msg.Body = "Hello SMS"
    msg.Targets = []target.Target{
        sms.CreateTarget("+86 138 0013 8000"),
    }

    receipt, err := client.Send(ctx, msg)
}
```

## 📊 监控和健康检查

### 平台健康检查

```go
err := platform.IsHealthy(ctx)
if err != nil {
    log.Printf("SMS平台不健康: %v", err)
}
```

### 提供商状态查询

```go
capabilities := platform.GetCapabilities()
fmt.Printf("支持的目标类型: %v\n", capabilities.SupportedTargetTypes)
fmt.Printf("支持的格式: %v\n", capabilities.SupportedFormats)
fmt.Printf("最大消息长度: %d\n", capabilities.MaxMessageSize)
```

## 🛠️ 扩展开发

### 1. 添加新的短信提供商

```go
// 实现 SMSProvider 接口
type CustomProvider struct {
    apiKey string
    // ...
}

func (p *CustomProvider) Send(ctx context.Context, phone, content, templateID string) (*SMSResult, error) {
    // 实现发送逻辑
    return &SMSResult{
        MessageID: "custom_123",
        Status:    "sent",
        Cost:      0.05,
        Parts:     1,
    }, nil
}

// 在 createProvider 函数中添加
case "custom":
    return NewCustomProvider(cfg.Credentials)
```

### 2. 自定义验证规则

```go
func (p *Platform) ValidateTarget(target target.Target) error {
    // 添加自定义验证逻辑
    if strings.Contains(target.Value, "blocked") {
        return fmt.Errorf("blocked phone number")
    }
    return nil
}
```

## 📚 最佳实践

### 1. 错误处理

```go
results, err := platform.Send(ctx, msg, targets)
if err != nil {
    log.Printf("发送失败: %v", err)
    return
}

for i, result := range results {
    if result.Error != nil {
        // 记录失败的目标
        log.Printf("目标 %d 发送失败: %v", i, result.Error)
        // 可以实现重试逻辑
    }
}
```

### 2. 资源管理

```go
platform, err := sms.New(config)
if err != nil {
    return err
}
// 确保资源释放
defer func() {
    if err := platform.Close(); err != nil {
        log.Printf("关闭平台失败: %v", err)
    }
}()
```

### 3. 并发安全

所有组件都是并发安全的，可以在多个 goroutine 中安全使用：

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        // 并发发送短信
        results, err := platform.Send(ctx, msg, targets)
        // ...
    }(i)
}
wg.Wait()
```

## 🔍 故障排除

### 常见问题

1. **手机号格式错误**
   - 确保手机号符合国际格式
   - 支持格式：`+86 138 0013 8000`、`+1 555 123 4567`

2. **限流被触发**
   - 检查限流配置是否合理
   - 使用 `GetStats()` 查看当前状态

3. **提供商认证失败**
   - 验证 credentials 配置
   - 检查API密钥是否有效

4. **模板变量未替换**
   - 确保模板中的变量格式正确：`{{变量名}}`
   - 检查 Variables 字段是否包含所需变量

### 调试技巧

```go
// 启用详细日志
config.Timeout = 30

// 检查平台能力
caps := platform.GetCapabilities()
log.Printf("平台能力: %+v", caps)

// 验证目标
for _, target := range targets {
    if err := platform.ValidateTarget(target); err != nil {
        log.Printf("目标验证失败 %s: %v", target.Value, err)
    }
}
```

## 📄 许可证

本示例代码遵循与 NotifyHub 主项目相同的许可证。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进此示例。

---

通过本示例，您可以完全独立地扩展 NotifyHub 的短信功能，无需修改核心代码，同时享受完整的企业级功能支持。