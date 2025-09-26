# 企业微信推送示例 (外部平台实现)

WeChat Work Push Example (External Platform Implementation)

这是一个演示如何使用 NotifyHub 外部企业微信平台进行消息推送的示例应用。

This is an example application demonstrating how to use NotifyHub's external WeChat Work platform for message pushing.

## 功能特性 / Features

- ✅ **外部平台集成**: 演示如何注册和使用外部平台
- ✅ **企业微信机器人**: 支持企业微信群机器人 Webhook API
- ✅ **多种消息格式**: 文本、Markdown、图片、新闻等消息类型
- ✅ **三种安全模式**: IP白名单、签名验证、关键词验证
- ✅ **自动关键词添加**: 配置关键词验证时自动添加到消息中
- ✅ **完整错误处理**: 详细的错误信息和重试机制
- ✅ **健康检查**: 平台可用性检查

- ✅ **External Platform Integration**: Demonstrates how to register and use external platforms
- ✅ **WeChat Work Robot**: Supports WeChat Work group robot Webhook API
- ✅ **Multiple Message Formats**: Text, Markdown, image, news and other message types
- ✅ **Three Security Modes**: IP whitelist, signature verification, keyword verification
- ✅ **Automatic Keyword Addition**: Automatically adds keywords to messages when keyword verification is configured
- ✅ **Complete Error Handling**: Detailed error information and retry mechanism
- ✅ **Health Check**: Platform availability check

## 快速开始 / Quick Start

### 1. 环境准备 / Environment Setup

```bash
# 克隆项目 / Clone project
git clone https://github.com/kart-io/notifyhub.git
cd notifyhub/examples/wechatwork

# 运行设置脚本 / Run setup script
./setup.sh

# 或手动构建 / Or build manually
go build -o wechatwork-example .
```

### 2. 配置企业微信机器人 / Configure WeChat Work Robot

1. 在企业微信群中点击 `...` → `群机器人`
2. 点击 `添加群机器人` → `自定义机器人`
3. 设置机器人名称和描述
4. 选择安全设置（IP白名单、签名验证、关键词验证）
5. 复制生成的 Webhook URL

1. In WeChat Work group, click `...` → `Group Robot`
2. Click `Add Group Robot` → `Custom Robot`
3. Set robot name and description
4. Choose security settings (IP whitelist, signature verification, keyword verification)
5. Copy the generated Webhook URL

### 3. 环境变量配置 / Environment Variables

```bash
# 必需：企业微信 Webhook URL / Required: WeChat Work Webhook URL
export WECHATWORK_WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key"

# 可选：签名密钥（签名验证模式） / Optional: Signature secret (signature verification mode)
export WECHATWORK_SECRET="your_secret_key"

# 可选：关键词验证 / Optional: Keyword verification
export WECHATWORK_KEYWORDS="通知"
```

### 4. 运行示例 / Run Example

```bash
# 使用环境变量 / Using environment variables
./wechatwork-example

# 或直接运行 / Or run directly
go run main.go
```

## 配置选项 / Configuration Options

### 使用环境变量 / Using Environment Variables

```bash
export WECHATWORK_WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key"
export WECHATWORK_SECRET="your_secret"
export WECHATWORK_KEYWORDS="通知"
```

### 使用配置文件 / Using Configuration File

编辑 `config.yaml` 文件：

Edit the `config.yaml` file:

```yaml
platforms:
  wechatwork:
    webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key"
    secret: "your_secret"
    keywords:
      - "通知"
      - "告警"
    timeout: "30s"
```

## 消息格式示例 / Message Format Examples

### 文本消息 / Text Message

```go
textMsg := &message.Message{
    ID:     "text-001",
    Title:  "系统通知",
    Body:   "这是一条企业微信测试消息",
    Format: message.FormatText,
    Targets: []target.Target{
        {Type: "wechatwork", Value: "default", Platform: "wechatwork"},
    },
}
```

### Markdown 消息 / Markdown Message

```go
markdownMsg := &message.Message{
    ID:    "markdown-001",
    Title: "📊 系统状态报告",
    Body: `## 服务状态
**✅ Web服务**: 正常运行
**⚠️ 缓存服务**: 高负载
[查看详情](https://dashboard.example.com)`,
    Format: message.FormatMarkdown,
    Targets: []target.Target{
        {Type: "wechatwork", Value: "status", Platform: "wechatwork"},
    },
}
```

### 新闻消息 / News Message

```go
newsMsg := &message.Message{
    ID: "news-001",
    Title: "新闻发布",
    PlatformData: map[string]interface{}{
        "wechatwork_news": map[string]interface{}{
            "articles": []map[string]interface{}{
                {
                    "title":       "NotifyHub v3.0 发布",
                    "description": "全新的统一通知平台架构",
                    "url":         "https://github.com/kart-io/notifyhub",
                    "picurl":      "https://example.com/logo.png",
                },
            },
        },
    },
    Targets: []target.Target{
        {Type: "wechatwork", Value: "news", Platform: "wechatwork"},
    },
}
```

## 安全模式 / Security Modes

### 1. IP白名单 / IP Whitelist

在企业微信机器人设置中配置服务器IP地址。

Configure server IP addresses in WeChat Work robot settings.

### 2. 签名验证 / Signature Verification

```go
config := &wechatwork.WeChatWorkConfig{
    WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key",
    Secret:     "your_secret_key",
    Timeout:    30 * time.Second,
}
```

### 3. 关键词验证 / Keyword Verification

```go
config := &wechatwork.WeChatWorkConfig{
    WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key",
    Keywords:   []string{"通知", "告警", "监控"},
    Timeout:    30 * time.Second,
}
```

## 外部平台架构 / External Platform Architecture

### 注册外部平台 / Register External Platform

```go
import "github.com/kart-io/notifyhub/external/wechatwork"

// 注册外部企业微信平台
err := wechatwork.RegisterWithNotifyHub()
if err != nil {
    log.Fatalf("注册企业微信平台失败: %v", err)
}
```

### 外部平台优势 / External Platform Advantages

- **插件化架构**: 不修改核心代码即可扩展新平台
- **解耦设计**: 外部平台独立维护，不影响核心系统
- **灵活集成**: 可以选择性地集成所需的平台
- **第三方扩展**: 社区或第三方可以贡献自己的平台实现

- **Plugin Architecture**: Extend new platforms without modifying core code
- **Decoupled Design**: External platforms are maintained independently
- **Flexible Integration**: Selectively integrate required platforms
- **Third-party Extension**: Community or third parties can contribute platform implementations

## 错误处理 / Error Handling

企业微信 API 可能返回以下错误码：

WeChat Work API may return the following error codes:

- `93000`: 无效的机器人 key / Invalid robot key
- `93001`: 机器人已被禁用 / Robot has been disabled
- `93002`: 消息格式错误 / Message format error
- `93003`: 消息内容包含敏感词汇 / Message contains sensitive words
- `93004`: 图片无法解析 / Image cannot be parsed

## 测试和调试 / Testing and Debugging

### 运行健康检查 / Run Health Check

示例应用会自动运行健康检查，验证平台可用性。

The example application automatically runs health checks to verify platform availability.

### 调试模式 / Debug Mode

```bash
# 启用调试日志 / Enable debug logging
export LOG_LEVEL=debug
go run main.go
```

### 测试地址 / Test Endpoint

如果未设置 `WECHATWORK_WEBHOOK_URL`，示例将使用测试地址 `https://httpbin.org/post`。

If `WECHATWORK_WEBHOOK_URL` is not set, the example will use test endpoint `https://httpbin.org/post`.

## 参考资料 / References

- [企业微信群机器人配置说明](https://developer.work.weixin.qq.com/document/path/91770)
- [企业微信机器人消息格式](https://developer.work.weixin.qq.com/document/path/91770#%E6%96%87%E6%9C%AC%E7%B1%BB%E5%9E%8B)
- [NotifyHub 外部平台开发指南](../../external/wechatwork/README.md)
- [NotifyHub 核心文档](../../README.md)

## 许可证 / License

本项目基于 MIT 许可证开源。

This project is open source under the MIT License.