# 企业微信外部平台实现

这是一个外部企业微信平台实现，演示了如何为 NotifyHub 创建自定义外部平台适配器。

## 架构说明

### 外部平台 vs 内部平台

- **内部平台**: 位于 `pkg/platforms/` 目录下，在编译时自动注册
- **外部平台**: 位于 `external/` 目录下，需要手动注册和集成

### 设计理念

外部平台实现允许：

1. **插件化架构**: 不修改核心代码即可扩展新平台
2. **解耦设计**: 外部平台独立维护，不影响核心系统
3. **灵活集成**: 可以选择性地集成所需的平台
4. **第三方扩展**: 社区或第三方可以贡献自己的平台实现

## 功能特性

- ✅ **企业微信机器人集成**: 支持企业微信群机器人 Webhook API
- ✅ **多种消息格式**: 支持文本、Markdown、图片、新闻等消息类型
- ✅ **安全模式支持**: 支持签名验证和关键词验证
- ✅ **自动关键词添加**: 当配置关键词验证时自动添加关键词
- ✅ **完整的错误处理**: 详细的错误信息和重试机制
- ✅ **平台接口兼容**: 完全兼容 NotifyHub 的 Platform 接口

## 企业微信机器人配置

### 创建企业微信机器人

1. 在企业微信群中点击 `...` → `群机器人`
2. 点击 `添加群机器人`
3. 选择 `自定义机器人`
4. 设置机器人名称和描述
5. 选择安全设置（IP白名单、签名验证、关键词验证）
6. 复制生成的 Webhook URL

### 安全配置选项

**1. IP白名单（推荐生产环境）**

```
配置服务器IP地址到企业微信机器人白名单中
```

**2. 签名验证**

```go
config := &wechatwork.WeChatWorkConfig{
    WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key",
    Secret:     "your_secret_key",
    Timeout:    30 * time.Second,
}
```

**3. 关键词验证**

```go
config := &wechatwork.WeChatWorkConfig{
    WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key",
    Keywords:   []string{"通知", "告警", "监控"},
    Timeout:    30 * time.Second,
}
```

## 使用方法

### 1. 注册外部平台

在应用启动时注册外部平台：

```go
import "github.com/kart-io/notifyhub/external/wechatwork"

// 注册企业微信外部平台
err := wechatwork.RegisterWithNotifyHub()
if err != nil {
    log.Fatalf("注册企业微信平台失败: %v", err)
}
```

### 2. 配置和使用

```go
// 创建配置
cfg := &config.Config{
    Platforms: map[string]map[string]interface{}{
        "wechatwork": {
            "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key",
            "secret":      "your_secret", // 可选
            "keywords":    []string{"通知"}, // 可选
            "timeout":     "30s",
        },
    },
    Logger: logger,
}

// 创建 Hub
hub, err := core.NewHub(cfg)
if err != nil {
    log.Fatal(err)
}

// 发送消息
msg := &message.Message{
    Title: "系统通知",
    Body:  "这是一条企业微信测试消息",
    Targets: []target.Target{
        {Type: "wechatwork", Value: "default", Platform: "wechatwork"},
    },
}

receipt, err := hub.Send(context.Background(), msg)
```

### 3. 环境变量配置

```bash
# 必需：企业微信 Webhook URL
export WECHATWORK_WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key"

# 可选：签名密钥
export WECHATWORK_SECRET="your_secret_key"

# 可选：关键词验证
export WECHATWORK_KEYWORDS="通知"
```

## 支持的消息格式

### 1. 文本消息

```go
msg := &message.Message{
    Title:  "通知标题",
    Body:   "这是消息内容",
    Format: message.FormatText,
    Targets: []target.Target{
        {Type: "wechatwork", Value: "default", Platform: "wechatwork"},
    },
}
```

### 2. Markdown 消息

```go
msg := &message.Message{
    Title: "📊 状态报告",
    Body:  `## 系统状态

**CPU**: <font color="warning">65%</font>
**内存**: <font color="info">45%</font>

[查看详情](https://dashboard.example.com)`,
    Format: message.FormatMarkdown,
}
```

### 3. 新闻消息

```go
msg := &message.Message{
    Title: "新闻发布",
    PlatformData: map[string]interface{}{
        "wechatwork_news": map[string]interface{}{
            "articles": []map[string]interface{}{
                {
                    "title":       "标题",
                    "description": "描述",
                    "url":         "https://example.com",
                    "picurl":      "https://example.com/image.png",
                },
            },
        },
    },
}
```

### 4. 图片消息

```go
msg := &message.Message{
    Title: "图片消息",
    PlatformData: map[string]interface{}{
        "wechatwork_image": map[string]interface{}{
            "base64": "base64_encoded_image_data",
            "md5":    "image_md5_hash",
        },
    },
}
```

## 错误处理

企业微信 API 可能返回以下错误码：

- `93000`: 无效的机器人 key
- `93001`: 机器人已被禁用
- `93002`: 消息格式错误
- `93003`: 消息内容包含敏感词汇
- `93004`: 图片无法解析（图片消息）

所有错误都会在 `SendResult.Error` 中返回详细信息。

## 扩展开发指南

### 创建新的外部平台

1. **创建平台目录**:

   ```bash
   mkdir external/myplatform
   ```

2. **实现 Platform 接口**:

   ```go
   type MyPlatformSender struct {
       // 平台特定字段
   }

   func (m *MyPlatformSender) Name() string { return "myplatform" }
   func (m *MyPlatformSender) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
       // 实现发送逻辑
   }
   // 实现其他必需方法...
   ```

3. **创建注册函数**:

   ```go
   func RegisterWithNotifyHub() error {
       return platform.RegisterPlatform("myplatform", func(cfg map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
           // 创建和返回平台实例
       })
   }
   ```

4. **在应用中注册**:

   ```go
   import "path/to/myplatform"

   err := myplatform.RegisterWithNotifyHub()
   ```

### 最佳实践

1. **配置验证**: 在创建时验证必需的配置项
2. **错误处理**: 提供详细的错误信息和适当的重试机制
3. **日志记录**: 使用提供的 logger 记录关键操作
4. **安全考虑**: 正确处理敏感信息如密钥和令牌
5. **文档完善**: 提供清晰的使用文档和示例

## 测试

运行企业微信外部平台示例：

```bash
cd examples/wechatwork
go run main.go
```

## 参考资料

- [企业微信群机器人配置说明](https://developer.work.weixin.qq.com/document/path/91770)
- [企业微信机器人消息格式](https://developer.work.weixin.qq.com/document/path/91770#%E6%96%87%E6%9C%AC%E7%B1%BB%E5%9E%8B)
- [NotifyHub 平台接口文档](../../pkg/notifyhub/platform/README.md)
