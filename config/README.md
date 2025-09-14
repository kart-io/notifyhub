# Config Package

## 功能概述

Config包实现了NotifyHub的配置管理系统，采用函数式选项模式(Functional Options Pattern)提供灵活、类型安全的配置接口。支持环境变量、代码配置和默认值的组合使用。

## 核心设计

### Config结构体
```go
type Config struct {
    feishu    *FeishuConfig
    email     *EmailConfig
    queue     *QueueConfig
    routing   *RoutingConfig
    telemetry *TelemetryConfig
    logger    logger.Interface
}
```

### Option接口
```go
type Option interface {
    apply(*Config)
}

type optionFunc func(*Config)
```

## 配置类型

### 1. 飞书配置 (FeishuConfig)
```go
type FeishuConfig struct {
    WebhookURL string
    Secret     string
    Timeout    time.Duration
}
```

### 2. 邮件配置 (EmailConfig)
```go
type EmailConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    From     string
    UseTLS   bool
    Timeout  time.Duration
}
```

### 3. 队列配置 (QueueConfig)
```go
type QueueConfig struct {
    Type        string
    BufferSize  int
    Workers     int
    RetryPolicy *queue.RetryPolicy
}
```

### 4. 路由配置 (RoutingConfig)
```go
type RoutingConfig struct {
    Rules []RoutingRule
}

type RoutingRule struct {
    Name       string
    Priority   int
    Enabled    bool
    Conditions RuleConditions
    Actions    []RuleAction
}
```

### 5. 遥测配置 (TelemetryConfig)
```go
type TelemetryConfig struct {
    ServiceName     string
    ServiceVersion  string
    Environment     string
    OTLPEndpoint    string
    OTLPHeaders     map[string]string
    TracingEnabled  bool
    SampleRate      float64
    MetricsEnabled  bool
    Enabled         bool
}
```

## 使用示例

### 基本配置

```go
// 创建Hub配置
hub, err := client.New(
    config.WithFeishu("webhook-url", "secret"),
    config.WithEmail("smtp.gmail.com", 587, "user", "pass", "from@example.com", true, 30*time.Second),
    config.WithQueue("memory", 1000, 4),
    config.WithDefaultLogger(logger.Info),
)
```

### 环境变量配置

```bash
# 设置环境变量
export NOTIFYHUB_FEISHU_WEBHOOK_URL="https://open.feishu.cn/..."
export NOTIFYHUB_FEISHU_SECRET="your-secret"
export NOTIFYHUB_SMTP_HOST="smtp.gmail.com"
export NOTIFYHUB_SMTP_PORT=587
export NOTIFYHUB_SMTP_USERNAME="user@gmail.com"
export NOTIFYHUB_SMTP_PASSWORD="password"
export NOTIFYHUB_SMTP_FROM="from@example.com"
export NOTIFYHUB_QUEUE_WORKERS=4
```

```go
// 从环境变量加载
hub, err := client.New(config.WithDefaults())
```

### 预设配置组合

```go
// 默认配置（从环境变量）
config.WithDefaults()

// 测试配置
config.WithTestDefaults()

// 生产配置示例
hub, err := client.New(
    config.WithDefaults(),                    // 基础环境变量
    config.WithQueue("redis", 5000, 8),     // 生产级队列
    config.WithTelemetry("notifyhub", "v1.2.0", "production", "https://otlp.example.com"),
    config.WithDefaultLogger(logger.Info),   // 适中日志级别
)
```

## 配置选项详解

### 飞书配置选项

```go
// 基本飞书配置
config.WithFeishu("webhook-url", "secret")

// 带超时的飞书配置
config.WithFeishuTimeout(30 * time.Second)

// 从环境变量加载飞书配置
config.WithFeishuFromEnv()
// 读取: NOTIFYHUB_FEISHU_WEBHOOK_URL, NOTIFYHUB_FEISHU_SECRET
```

### 邮件配置选项

```go
// 完整邮件配置
config.WithEmail("smtp.gmail.com", 587, "user", "pass", "from@example.com", true, 30*time.Second)

// Gmail快速配置
config.WithGmail("username", "app-password", "from@gmail.com")

// 从环境变量加载
config.WithEmailFromEnv()
// 读取: NOTIFYHUB_SMTP_*, NOTIFYHUB_EMAIL_*
```

### 队列配置选项

```go
// 内存队列
config.WithQueue("memory", 1000, 4)

// Redis队列
config.WithRedisQueue("localhost:6379", "", 0) // host, password, db

// 自定义重试策略
retryPolicy := &queue.RetryPolicy{
    MaxRetries:      5,
    InitialInterval: 10 * time.Second,
    Multiplier:      2.0,
    MaxJitter:       2 * time.Second,
}
config.WithQueueRetryPolicy(retryPolicy)

// 从环境变量加载
config.WithQueueFromEnv()
```

### 遥测配置选项

```go
// 完整遥测配置
config.WithTelemetry("notifyhub", "v1.2.0", "production", "http://localhost:4318")

// 从环境变量加载
config.WithTelemetryFromEnv()
// 读取: NOTIFYHUB_TELEMETRY_ENABLED, NOTIFYHUB_SERVICE_NAME等

// 禁用遥测
config.WithTelemetryDisabled()
```

## 路由系统配置

### 创建路由规则

```go
// 高优先级规则 - 重要消息同时发送到多个平台
highPriorityRule := config.RoutingRule{
    Name:     "high_priority_all",
    Priority: 100,
    Enabled:  true,
    Conditions: config.RuleConditions{
        Priority: []int{4, 5}, // 优先级4-5
    },
    Actions: []config.RuleAction{
        {Type: "route", Platforms: []string{"feishu", "email"}},
    },
}

// 告警类消息 - 发送到飞书
alertRule := config.RoutingRule{
    Name:     "alerts_to_feishu",
    Priority: 50,
    Enabled:  true,
    Conditions: config.RuleConditions{
        Metadata: map[string]string{"type": "alert"},
    },
    Actions: []config.RuleAction{
        {Type: "route", Platforms: []string{"feishu"}},
    },
}
```

### 应用路由配置

```go
// 自定义路由规则
config.WithRouting(highPriorityRule, alertRule)

// 使用默认路由规则
config.WithDefaultRouting()
```

## 日志配置

### 日志级别配置

```go
// 自定义日志器
customLogger := logger.New(os.Stdout, logger.Config{
    Level:      logger.Debug,
    TimeFormat: "2006-01-02 15:04:05",
    Colorful:   true,
})
config.WithLogger(customLogger)

// 默认日志器配置
config.WithDefaultLogger(logger.Info)

// 静默模式
config.WithSilentLogger()
```

## 环境变量列表

### 飞书相关
- `NOTIFYHUB_FEISHU_WEBHOOK_URL` - Webhook URL
- `NOTIFYHUB_FEISHU_SECRET` - 签名密钥
- `NOTIFYHUB_FEISHU_TIMEOUT` - 超时时间

### 邮件相关
- `NOTIFYHUB_SMTP_HOST` - SMTP服务器地址
- `NOTIFYHUB_SMTP_PORT` - SMTP端口
- `NOTIFYHUB_SMTP_USERNAME` - 用户名
- `NOTIFYHUB_SMTP_PASSWORD` - 密码
- `NOTIFYHUB_SMTP_FROM` - 发件人地址
- `NOTIFYHUB_SMTP_USE_TLS` - 是否使用TLS

### 队列相关
- `NOTIFYHUB_QUEUE_TYPE` - 队列类型 (memory/redis)
- `NOTIFYHUB_QUEUE_BUFFER_SIZE` - 缓冲区大小
- `NOTIFYHUB_QUEUE_WORKERS` - Worker数量
- `NOTIFYHUB_RETRY_MAX` - 最大重试次数
- `NOTIFYHUB_RETRY_INTERVAL` - 重试间隔

### 遥测相关
- `NOTIFYHUB_TELEMETRY_ENABLED` - 是否启用遥测
- `NOTIFYHUB_SERVICE_NAME` - 服务名称
- `NOTIFYHUB_SERVICE_VERSION` - 服务版本
- `NOTIFYHUB_ENVIRONMENT` - 环境标识
- `NOTIFYHUB_OTLP_ENDPOINT` - OTLP端点
- `NOTIFYHUB_OTLP_AUTH` - OTLP认证头
- `NOTIFYHUB_TRACING_ENABLED` - 启用链路追踪
- `NOTIFYHUB_METRICS_ENABLED` - 启用指标收集
- `NOTIFYHUB_SAMPLE_RATE` - 采样率

## 高级用法

### 自定义配置选项

```go
// 创建自定义配置选项
func WithCustomTimeout(timeout time.Duration) config.Option {
    return config.optionFunc(func(c *config.Config) {
        if feishu := c.Feishu(); feishu != nil {
            feishu.Timeout = timeout
        }
        if email := c.Email(); email != nil {
            email.Timeout = timeout
        }
    })
}

// 使用自定义选项
hub, err := client.New(
    config.WithDefaults(),
    WithCustomTimeout(60 * time.Second),
)
```

### 配置验证

```go
// 验证配置完整性
func ValidateConfig(cfg *config.Config) error {
    if cfg.Feishu() == nil && cfg.Email() == nil {
        return errors.New("至少需要配置一个通知平台")
    }

    if queue := cfg.Queue(); queue != nil {
        if queue.Workers <= 0 {
            return errors.New("worker数量必须大于0")
        }
    }

    return nil
}
```

### 配置合并

```go
// 创建基础配置
baseConfig := config.New(
    config.WithDefaults(),
    config.WithQueue("memory", 1000, 2),
)

// 创建生产配置（覆盖基础配置）
prodConfig := config.New(
    config.WithDefaults(),
    config.WithQueue("redis", 5000, 8),
    config.WithTelemetry("notifyhub", "v1.2.0", "production", "https://otlp.example.com"),
)
```

## 最佳实践

### 1. 环境分离

```go
// 开发环境
func NewDevelopmentConfig() []config.Option {
    return []config.Option{
        config.WithTestDefaults(),
        config.WithDefaultLogger(logger.Debug),
        config.WithTelemetryDisabled(),
    }
}

// 生产环境
func NewProductionConfig() []config.Option {
    return []config.Option{
        config.WithDefaults(),
        config.WithQueue("redis", 10000, 12),
        config.WithTelemetryFromEnv(),
        config.WithDefaultLogger(logger.Warn),
    }
}
```

### 2. 配置文件支持

```go
// 从YAML文件加载配置
func WithConfigFile(filename string) config.Option {
    return config.optionFunc(func(c *config.Config) {
        // 实现YAML配置文件解析
        data, _ := ioutil.ReadFile(filename)
        var yamlConfig YamlConfig
        yaml.Unmarshal(data, &yamlConfig)
        // 应用配置...
    })
}
```

### 3. 配置热重载

```go
// 监控配置文件变化
func WatchConfigFile(filename string, reloadFunc func()) {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(filename)

    go func() {
        for event := range watcher.Events {
            if event.Op&fsnotify.Write == fsnotify.Write {
                reloadFunc()
            }
        }
    }()
}
```

## 错误处理

### 配置验证错误
- 必需字段缺失
- 参数格式错误
- 网络连接配置错误

### 环境变量解析错误
- 类型转换失败
- 格式不正确
- 缺少必需的环境变量

## 文件说明

- `options.go` - 核心配置结构体和选项函数
- `routing.go` - 路由引擎和规则配置
- `defaults.go` - 默认配置和预设组合（如存在）