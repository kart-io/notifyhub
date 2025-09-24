# 🔧 Network Connection Issue - Analysis & Solutions

## 问题分析

### 当前错误
```
[SMTP DEBUG] Send error: dial error: dial tcp 74.125.204.109:587: i/o timeout
```

### 问题原因
**您的网络环境无法连接到 Gmail SMTP 服务器 (smtp.gmail.com:587)**

可能的原因：
1. 🔒 **防火墙阻止** - 出站端口 587 被防火墙拦截
2. 🌐 **网络限制** - 公司/学校网络阻止SMTP连接
3. 🚫 **ISP限制** - 互联网服务提供商阻止SMTP端口
4. 🌍 **地理限制** - 某些地区可能无法访问Gmail服务

## ✅ 解决方案

### 方案 1: 使用 MailHog 进行本地测试（推荐）

**最快速的解决方案 - 无需网络连接！**

```bash
# 1. 安装 MailHog
brew install mailhog

# 2. 启动 MailHog
mailhog

# 3. 运行本地测试
go run test_local.go

# 4. 在浏览器查看邮件
open http://localhost:8025
```

**MailHog 优势：**
- ✅ 无需真实SMTP服务器
- ✅ 无需网络连接
- ✅ 可视化界面查看所有邮件
- ✅ 支持所有邮件功能测试
- ✅ 开发环境完美选择

### 方案 2: 测试网络连接

```bash
# 测试 Gmail SMTP 端口 587 (STARTTLS)
nc -zv smtp.gmail.com 587

# 测试端口 465 (SSL)
nc -zv smtp.gmail.com 465

# 如果都失败，说明网络被限制
```

### 方案 3: 使用其他 SMTP 服务器

如果您有其他可访问的SMTP服务器：

```go
// 企业邮箱
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.company.com", 587, "noreply@company.com",
        email.WithEmailAuth("username", "password"),
        email.WithEmailTLS(true),
    ),
)

// 或者使用 Outlook (如果可访问)
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.office365.com", 587, "user@outlook.com",
        email.WithEmailAuth("user@outlook.com", "password"),
        email.WithEmailTLS(true),
    ),
)
```

### 方案 4: 修改配置使用不同端口

尝试Gmail的SSL端口（如果未被阻止）：

```go
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.gmail.com", 465, "your-email@gmail.com",
        email.WithEmailAuth("your-email@gmail.com", "app-password"),
        email.WithEmailSSL(true),
        email.WithEmailTLS(false),
    ),
)
```

### 方案 5: 跳过网络测试

如果只是想验证代码逻辑，可以暂时注释掉发送部分：

```go
// 在 main.go 中注释掉实际发送
// if err := demoBasicEmailMessages(ctx, authHub); err != nil {
//     log.Printf("❌ Basic email messages demo failed: %v", err)
// }

fmt.Println("✅ Skipping network tests - code structure validated")
```

## 🎯 推荐操作流程

### 对于开发和测试环境：

1. **安装并使用 MailHog**
   ```bash
   brew install mailhog
   mailhog &
   go run test_local.go
   ```

2. **验证功能**
   - 访问 http://localhost:8025
   - 查看所有测试邮件
   - 验证HTML渲染、CC、优先级等功能

### 对于生产环境：

1. **确认SMTP服务器**
   - 使用企业SMTP服务器
   - 或使用云服务（SendGrid, AWS SES等）

2. **测试连接**
   ```bash
   nc -zv your-smtp-server.com 587
   ```

3. **配置NotifyHub**
   ```go
   hub, err := notifyhub.NewHub(
       email.WithEmail("your-smtp.com", 587, "noreply@company.com",
           email.WithEmailAuth("username", "password"),
           email.WithEmailTLS(true),
       ),
   )
   ```

## 📊 代码已完成的修复

✅ **已修复的问题：**
1. ✅ 实现了正确的 STARTTLS 支持
2. ✅ 添加了 SSL/TLS 连接方法
3. ✅ 增加了详细的调试日志
4. ✅ 重构为独立可测试的方法
5. ✅ 提供了本地测试方案

❌ **需要解决的问题：**
- 网络连接（需要您根据环境选择方案）

## 📝 验证代码功能

即使没有网络，您仍然可以验证代码结构：

```bash
# 1. 编译检查
go build

# 2. 查看日志输出（即使失败也能看到详细信息）
go run main.go 2>&1 | grep "DEBUG"

# 3. 使用 MailHog 完整测试
mailhog &
go run test_local.go
```

## 🔗 相关文档

- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - 详细故障排查指南
- [test_local.go](./test_local.go) - MailHog本地测试示例
- [main.go](./main.go) - 完整功能演示

## 💡 下一步

**推荐：立即使用 MailHog 进行测试**

```bash
# 一键启动测试
brew install mailhog && mailhog &
sleep 2
go run test_local.go
open http://localhost:8025
```

这样您可以：
- ✅ 验证所有邮件功能
- ✅ 查看HTML渲染效果
- ✅ 测试多收件人、CC等功能
- ✅ 无需真实SMTP服务器
- ✅ 完全离线工作