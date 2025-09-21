# Feishu Demo

本示例演示如何使用NotifyHub发送Feishu（飞书）通知。

## ⚠️ 重要说明

本示例已完全修复并可正常工作！示例代码默认从环境变量读取配置，需要设置有效的飞书机器人凭证。

## 🔧 配置步骤

### 第一步：创建飞书机器人

1. 在飞书群组中，点击右上角设置 → 群机器人
2. 点击"添加机器人" → "自定义机器人"
3. 设置机器人名称和描述
4. **开启"签名校验"**（重要！）
5. 可选：配置IP白名单和自定义关键词
6. 复制生成的**Webhook URL**和**签名密钥**

### 第二步：配置凭证

#### 方法1：使用环境变量（推荐）

```bash
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/YOUR_WEBHOOK_ID"
export FEISHU_SECRET="your-actual-secret"
go run main.go
```

#### 方法2：直接修改代码

编辑 `main.go` 文件第54-58行，将空字符串替换为你的实际webhook URL和secret。

#### 方法3：高级配置（新功能）

使用命令行参数运行不同的演示类型。

## 🚀 运行示例

### 基础示例（默认）

```bash
go run main.go
# 或者明确指定
go run main.go -demo=basic
```

### 高级功能示例

```bash
go run main.go -demo=advanced
```

### 运行所有演示

```bash
go run main.go -demo=all
```

### 查看帮助

```bash
go run main.go -h
```

## 🆕 新功能 - 安全设置选项

根据飞书官方文档，安全设置支持**四选一**模式：签名校验、IP白名单、自定义关键词或无安全验证。

### 1. 签名校验（推荐）

使用密钥进行HMAC-SHA256签名验证：

```go
client, err := notifyhub.New(
    notifyhub.WithFeishu(webhookURL, secret), // 默认方式
    // 或者明确指定
    notifyhub.WithFeishuSignature(webhookURL, secret),
)
```

### 2. IP白名单

限制特定IP地址访问webhook（IP需在飞书机器人后台预先配置）：

```go
client, err := notifyhub.New(
    notifyhub.WithFeishuIPWhitelist(webhookURL),
)
```

**代理支持**：如果需要通过代理服务器访问，可以配置代理：

```go
client, err := notifyhub.New(
    notifyhub.WithFeishuIPWhitelistProxy(webhookURL, "http://proxy-server:8080"),
)
```

注意：使用代理时，需要在飞书机器人后台配置代理服务器的IP地址。

### 3. 自定义关键词

为消息添加自动标签，便于分类和检索：

```go
client, err := notifyhub.New(
    notifyhub.WithFeishuKeywords(webhookURL, []string{
        "紧急", "监控", "生产环境",
    }),
)
```

### 4. 无安全验证

不使用任何安全验证（不推荐）：

```go
client, err := notifyhub.New(
    notifyhub.WithFeishuNone(webhookURL),
)
```

## ✅ 预期结果

**成功时：**

```
✅ 消息发送成功! 结果: ...
```

**失败时（未配置凭证）：**

```
❌ 发送失败: webhook URL和secret不能为空

📋 故障排除:
1. 确认已设置环境变量 FEISHU_WEBHOOK_URL 和 FEISHU_SECRET
2. 确认webhook URL格式正确
3. 确认secret是从飞书后台复制的正确值
4. 确认飞书机器人已在群组中启用
```

## 🔍 常见问题

### Q: 如何获取飞书机器人的webhook URL和secret？

A:
1. 在飞书群组中点击设置 → 群机器人 → 添加机器人 → 自定义机器人
2. 设置机器人名称并**开启签名校验**
3. 复制生成的Webhook URL和签名密钥

### Q: 为什么消息发送成功但群组里看不到？

A: 请检查：
- 机器人是否已添加到正确的群组
- 群组权限设置是否允许机器人发送消息
- 是否使用了正确的webhook URL

### Q: 签名验证算法是如何实现的？

A: NotifyHub使用飞书官方标准算法：以`timestamp + "\n" + secret`作为HMAC-SHA256密钥，空字符串作为数据进行签名。

### Q: 为什么安全设置是"四选一"？

A: 根据飞书官方文档，机器人安全设置只能选择其中一种方式：签名校验、IP白名单、自定义关键词或无安全验证。这是飞书平台的限制，不能同时配置多种安全方式。

### Q: 推荐使用哪种安全设置？

A:
- **签名校验**：最安全，推荐用于生产环境
- **IP白名单**：适合固定IP的内网环境
- **自定义关键词**：适合需要消息分类的场景
- **无验证**：仅适合测试环境，不推荐生产使用

### Q: 自定义关键词如何显示？

A: 使用自定义关键词模式时，关键词会自动添加到消息末尾，格式为`🏷️ 标签: #关键词1 | #关键词2`。

### Q: 可以切换安全设置类型吗？

A: 可以通过创建不同的客户端实例来使用不同的安全设置，但单个客户端只能使用一种安全设置。

### Q: 如何在IP白名单模式下使用代理？

A:
1. 配置代理服务器，确保代理服务器可以访问飞书API
2. 在飞书机器人后台配置代理服务器的IP地址（而不是应用服务器的IP）
3. 使用 `WithFeishuIPWhitelistProxy(webhook, proxyURL)` 配置客户端
4. 或设置环境变量 `HTTP_PROXY` 后使用普通的IP白名单配置

### Q: 代理支持哪些协议？

A: 支持HTTP/HTTPS代理，代理URL格式：
- HTTP代理：`http://proxy-server:port`
- HTTPS代理：`https://proxy-server:port`
- 带认证：`http://username:password@proxy-server:port`

## 🔧 技术说明

### 修复历史
- ✅ **签名验证算法已修复** - 根据飞书官方文档和PHP实现参考，修复了HMAC-SHA256签名算法
- ✅ **配置系统完善** - 支持环境变量配置，提供详细的错误处理和故障排除
- ✅ **完全测试验证** - 示例已通过实际飞书API测试，确保正常工作

### 核心特性
- ✅ 支持飞书官方签名校验
- ✅ 环境变量配置支持
- ✅ 详细的错误处理和提示
- ✅ 完整的文本、Markdown和卡片消息支持
- 🆕 **三选一安全设置** - 签名校验、IP白名单、自定义关键词
- 🆕 **自动配置验证** - 确保只使用一种安全方式
- 🆕 **多种配置函数** - 针对不同安全需求的专用配置

## 📚 参考文档

- [飞书机器人开发指南](https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot)
- [飞书Webhook签名验证](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN)
