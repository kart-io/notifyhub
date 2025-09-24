# NotifyHub 文档中心

欢迎使用 NotifyHub！这里是完整的文档导航。

## 📚 文档目录

### 🚀 快速开始

- **[README.md](README.md)** - 项目概览、安装指南和快速开始
- **[EXAMPLES.md](EXAMPLES.md)** - 详细的使用示例和最佳实践

### 📖 API 参考

- **[API.md](API.md)** - 完整的 API 参考文档
- **[doc.go](doc.go)** - Go 包文档和代码示例

### 🏗️ 架构文档

- **[核心接口](core/)** - Hub 核心实现和接口定义
- **[平台注册](platform/)** - 平台扩展机制和注册系统
- **[适配器模式](adapters/)** - 内部平台适配器实现

## 📋 文档分类

### 入门指南

1. [安装和配置](README.md#安装)
2. [第一个通知](README.md#快速开始)
3. [基础概念](README.md#架构概览)

### 使用指南

1. [平台配置](EXAMPLES.md#平台配置)
2. [消息构建](EXAMPLES.md#消息构建)
3. [目标管理](EXAMPLES.md#目标管理)
4. [错误处理](EXAMPLES.md#错误处理)

### 高级功能

1. [异步发送](EXAMPLES.md#异步发送)
2. [健康检查](EXAMPLES.md#健康检查)
3. [平台扩展](EXAMPLES.md#平台扩展)
4. [批量处理](EXAMPLES.md#高级用法)

### API 参考

1. [核心接口](API.md#核心接口)
2. [配置选项](API.md#配置选项)
3. [消息类型](API.md#消息构建)
4. [回执处理](API.md#回执和状态)

## 🔗 快速链接

### 常用功能

- [创建 Hub](README.md#基本使用) - 基础 Hub 创建
- [发送消息](EXAMPLES.md#基础示例) - 消息发送示例
- [配置平台](API.md#配置选项) - 平台配置参考
- [错误处理](EXAMPLES.md#错误处理) - 错误处理模式

### 平台集成

- [飞书集成](API.md#withfeishu) - 飞书平台配置
- [邮件集成](API.md#withemail) - 邮件平台配置
- [自定义平台](EXAMPLES.md#平台扩展) - 扩展新平台

### 高级主题

- [批量发送](EXAMPLES.md#批量发送到多个目标) - 批量消息处理
- [模板消息](EXAMPLES.md#模板化消息) - 消息模板化
- [健康监控](EXAMPLES.md#定期健康检查) - 系统健康监控

## 📝 代码示例

### 最简单的使用

```go
hub, err := notifyhub.NewHub(notifyhub.WithTestDefaults())
msg := notifyhub.NewMessage("Hello").Body("World").AddTarget(notifyhub.NewEmailTarget("user@example.com")).Build()
receipt, err := hub.Send(context.Background(), msg)
```

### 生产环境配置

```go
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithEmail("smtp.gmail.com", 587, "user", "pass", "from@company.com", true, 30*time.Second),
    notifyhub.WithTimeout(45*time.Second),
)
```

### 多目标发送

```go
targets := []notifyhub.Target{
    notifyhub.NewEmailTarget("admin@company.com"),
    notifyhub.NewFeishuUserTarget("user123"),
    notifyhub.NewFeishuGroupTarget("group456"),
}
msg := notifyhub.NewAlert("Alert").Body("Message").AddTargets(targets...).Build()
```

## 🛠️ 开发资源

### 包结构

```
pkg/notifyhub/
├── README.md          # 主要文档
├── API.md             # API 参考
├── EXAMPLES.md        # 使用示例
├── DOCS.md           # 文档索引（本文件）
├── doc.go            # Go 包文档
├── api_adapter.go    # 公共 API 适配器
├── core/             # 核心实现
├── platform/         # 平台注册
├── config/           # 配置管理
├── message/          # 消息类型
├── target/           # 目标类型
└── receipt/          # 回执类型

internal/pkg/         # 内部实现 (符合 Go 设计原则)
├── adapters/         # 平台适配器
└── register/         # 自动注册
```

### Go 文档生成

```bash
# 生成本地文档
go doc github.com/kart-io/notifyhub/pkg/notifyhub

# 启动文档服务器
godoc -http=:6060
# 访问 http://localhost:6060/pkg/github.com/kart-io/notifyhub/pkg/notifyhub/
```

### 测试和验证

```bash
# 运行测试
go test ./...

# 运行示例
go run examples/feishu/cmd/example/main.go

# 验证导入
go build ./pkg/notifyhub/...
```

## 🆘 获取帮助

### 常见问题

1. **配置问题** - 查看 [配置选项](API.md#配置选项)
2. **发送失败** - 查看 [错误处理](EXAMPLES.md#错误处理)
3. **平台不支持** - 查看 [平台扩展](EXAMPLES.md#平台扩展)
4. **性能问题** - 查看 [批量处理](EXAMPLES.md#高级用法)

### 学习路径

1. **初学者**：README → 基础示例 → API 参考
2. **开发者**：API 参考 → 高级示例 → 平台扩展
3. **架构师**：架构概览 → 核心实现 → 扩展机制

### 贡献指南

- 报告问题：创建 GitHub Issue
- 提交代码：创建 Pull Request
- 改进文档：更新相关 Markdown 文件
- 添加示例：扩展 EXAMPLES.md

## 📊 文档状态

| 文档类型 | 状态 | 最后更新 |
|---------|------|----------|
| 基础文档 | ✅ 完成 | 2024-01-15 |
| API 参考 | ✅ 完成 | 2024-01-15 |
| 使用示例 | ✅ 完成 | 2024-01-15 |
| Go 文档 | ✅ 完成 | 2024-01-15 |

---

💡 **提示**：建议从 [README.md](README.md) 开始阅读，然后根据需要查看具体的功能文档。

📧 **反馈**：如果您发现文档中的错误或有改进建议，请创建 Issue 或 Pull Request。
