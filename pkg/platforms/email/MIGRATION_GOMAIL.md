# 迁移到 go-mail 库

## 📋 背景

`net/smtp` 已被 Go 官方标记为 deprecated（弃用），虽然还能用，但推荐迁移到现代化的邮件库。

### 为什么选择 go-mail？

**[wneessen/go-mail](https://github.com/wneessen/go-mail)** 是 net/smtp 的现代化替代方案：

✅ **优势：**

- 基于 net/smtp 的 fork，API 熟悉
- 支持更多 SMTP 认证方法
- 并发安全
- 更好的错误处理
- 积极维护
- 支持上下文（context）
- 更简洁的 API

❌ **net/smtp 问题：**

- 已弃用（deprecated）
- STARTTLS 支持不完善
- 缺少现代功能
- 停止更新

## 🚀 快速开始

### 安装依赖

```bash
go get -u github.com/wneessen/go-mail
```

### 使用新实现

**默认使用 go-mail（推荐）：**

```go
import "github.com/kart-io/notifyhub/pkg/platforms/email"

// go-mail 是默认实现，无需额外配置
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.gmail.com", 587, "from@example.com",
        email.WithEmailAuth("username", "password"),
        email.WithEmailTLS(true),
    ),
)
```

**切换到旧的 net/smtp（不推荐）：**

```go
import "github.com/kart-io/notifyhub/pkg/platforms/email"

// 在创建 Hub 之前调用
email.UseNetSMTP()

hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.gmail.com", 587, "from@example.com",
        email.WithEmailAuth("username", "password"),
        email.WithEmailTLS(true),
    ),
)
```

## 📊 API 对比

### 配置方式（完全兼容）

两个实现使用**相同的配置 API**，无需修改代码：

```go
// 这段代码同时兼容两种实现
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.gmail.com", 587, "from@example.com",
        email.WithEmailAuth("username", "password"),
        email.WithEmailTLS(true),
        email.WithEmailTimeout(30*time.Second),
    ),
)
```

### 功能对比

| 功能 | net/smtp | go-mail |
|------|----------|---------|
| SMTP 基础 | ✅ | ✅ |
| STARTTLS | ⚠️ 有问题 | ✅ 完善 |
| SSL/TLS | ❌ 需手动实现 | ✅ 内置 |
| 认证方法 | ⚠️ 基础 | ✅ 多种 |
| 并发安全 | ❌ | ✅ |
| Context 支持 | ❌ | ✅ |
| 错误处理 | ⚠️ 基础 | ✅ 详细 |
| HTML 邮件 | ✅ | ✅ |
| 附件 | ⚠️ 复杂 | ✅ 简单 |
| 维护状态 | ❌ 弃用 | ✅ 活跃 |

## 🔧 实现细节

### 文件结构

```
pkg/platforms/email/
├── sender.go           # 旧实现（net/smtp）- 保留兼容
├── sender_gomail.go    # 新实现（go-mail）- 推荐使用
├── options.go          # 统一配置，支持两种实现
└── MIGRATION_GOMAIL.md # 本文档
```

### 实现切换机制

```go
// options.go
var useGoMailLibrary = true // 默认使用 go-mail

// 切换到 net/smtp
func UseNetSMTP() {
    useGoMailLibrary = false
}

// 切换到 go-mail（默认）
func UseGoMail() {
    useGoMailLibrary = true
}
```

### 自动选择

```go
func ensureRegistered() {
    registerOnce.Do(func() {
        var creator func(map[string]interface{}) (platform.ExternalSender, error)

        if useGoMailLibrary {
            creator = NewEmailSenderGoMail  // 使用 go-mail
        } else {
            creator = NewEmailSender        // 使用 net/smtp
        }

        // 注册平台...
    })
}
```

## 📝 迁移步骤

### 步骤 1: 安装依赖

```bash
go get -u github.com/wneessen/go-mail
```

### 步骤 2: 更新代码（可选）

**无需修改代码！** go-mail 是默认实现。

如果想明确指定：

```go
import "github.com/kart-io/notifyhub/pkg/platforms/email"

func main() {
    // 明确使用 go-mail（可选，因为是默认）
    email.UseGoMail()

    hub, err := notifyhub.NewHub(
        email.WithEmail("smtp.gmail.com", 587, "from@example.com",
            email.WithEmailAuth("user", "pass"),
            email.WithEmailTLS(true),
        ),
    )
    // ... 其他代码不变
}
```

### 步骤 3: 测试

```bash
go test ./pkg/platforms/email/...
```

### 步骤 4: 回退方案（如有问题）

```go
// 临时切换回 net/smtp
email.UseNetSMTP()
```

## 🎯 新功能示例

### Context 支持

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// go-mail 自动使用 context
receipt, err := hub.Send(ctx, msg)
```

### 更好的错误处理

```go
receipt, err := hub.Send(ctx, msg)
if err != nil {
    // go-mail 提供更详细的错误信息
    fmt.Printf("发送失败: %v\n", err)
    // 示例: "failed to connect to SMTP server: dial tcp timeout"
}
```

### 优先级设置

```go
msg := notifyhub.NewAlert("Critical Alert").
    WithBody("Urgent message").
    WithPlatformData(map[string]interface{}{
        "email_priority": "high",  // go-mail 正确处理优先级
    }).
    Build()
```

## 🔍 故障排查

### 问题 1: 依赖安装失败

```bash
# 使用代理
export GOPROXY=https://goproxy.cn,direct
go get -u github.com/wneessen/go-mail
```

### 问题 2: 想使用旧实现

```go
// 在 main 函数开始处调用
email.UseNetSMTP()
```

### 问题 3: 编译错误

确保导入了正确的包：

```go
import (
    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/platforms/email"
)
```

## 📚 参考文档

- [go-mail GitHub](https://github.com/wneessen/go-mail)
- [go-mail 文档](https://pkg.go.dev/github.com/wneessen/go-mail)
- [NotifyHub Email 示例](../../../examples/platforms/email/)

## 🔄 版本兼容性

### v1.x（旧版）

- 使用 net/smtp
- 需要手动实现 STARTTLS

### v2.x（当前）

- 默认使用 go-mail
- 保留 net/smtp 兼容
- 统一的配置 API
- 更好的功能支持

## 💡 最佳实践

1. **新项目**: 直接使用 go-mail（默认）
2. **旧项目**:
   - 测试环境先切换到 go-mail
   - 验证无误后生产环境切换
   - 保留 `email.UseNetSMTP()` 作为回退方案
3. **问题排查**: 使用详细的错误日志
4. **性能优化**: go-mail 支持连接池和并发

## ❓ FAQ

**Q: 需要修改现有代码吗？**
A: 不需要！配置 API 完全兼容。

**Q: go-mail 更快吗？**
A: 是的，go-mail 支持连接池和更好的并发处理。

**Q: 如何验证使用的是哪个实现？**
A: 检查发送结果的 metadata：

```go
receipt, _ := hub.Send(ctx, msg)
library := receipt.Results[0].Metadata["library"]
fmt.Println(library) // "go-mail" 或 "net/smtp"
```

**Q: 可以混用两种实现吗？**
A: 不建议。在应用启动时选择一种实现即可。

## 🎉 总结

- ✅ **无缝迁移**: 无需修改代码
- ✅ **默认推荐**: go-mail 是默认实现
- ✅ **向后兼容**: 保留 net/smtp 支持
- ✅ **更多功能**: Context、优先级、更好的错误处理
- ✅ **未来保障**: 持续维护和更新
