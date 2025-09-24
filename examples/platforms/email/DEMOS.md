# Email Platform - Independent Demos

每个演示都是完全独立的，拥有自己的Hub，互不影响。

## 🎯 Demo列表

### Demo 1: 基础SMTP配置

```go
func demo1BasicSMTPConfig()
```

- 演示基础SMTP配置（无认证）
- 端口: 587
- 只创建Hub，不发送邮件

### Demo 2: 认证SMTP with TLS

```go
func demo2AuthenticatedSMTP()
```

- 演示SMTP认证配置
- TLS加密
- 自定义超时

### Demo 3: SSL配置

```go
func demo3SSLConfiguration()
```

- 演示SSL/TLS加密
- 端口: 465
- SSL连接

### Demo 4: 简单文本邮件

```go
func demo4SimpleTextEmail()
```

- 发送纯文本邮件
- 系统通知示例
- 时间戳

### Demo 5: HTML邮件

```go
func demo5HTMLEmail()
```

- 发送HTML格式邮件
- CSS样式
- 每日报告示例

### Demo 6: 优先级邮件

```go
func demo6EmailWithPriority()
```

- 设置邮件优先级
- 使用Alert类型
- 高优先级标记

### Demo 7: CC收件人

```go
func demo7EmailWithCC()
```

- 添加CC收件人
- PlatformData使用
- 多收件人抄送

### Demo 8: 模板邮件

```go
func demo8TemplateEmail()
```

- 变量替换
- 欢迎邮件模板
- 动态内容

### Demo 9: 多收件人

```go
func demo9MultipleRecipients()
```

- 发送给多个收件人
- 批量发送
- 结果统计

### Demo 10: 不同消息类型

```go
func demo10DifferentMessageTypes()
```

- Regular消息
- Alert消息
- Urgent消息
- 优先级对比

## 🚀 使用方法

### 运行所有Demo

```bash
go run main.go
```

### 运行单个Demo（修改main函数）

```go
func main() {
    demo4SimpleTextEmail()  // 只运行Demo 4
}
```

### 编译并运行

```bash
go build -o email_demo
./email_demo
```

## ⚙️ 配置说明

每个Demo内部都创建独立的Hub：

```go
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.gmail.com", 587, "your-email@gmail.com",
        email.WithEmailAuth("username", "password"),
        email.WithEmailTLS(true),
    ),
)
defer hub.Close(ctx)
```

### 修改SMTP配置

在每个demo函数中修改配置：

1. **修改SMTP服务器**: 将`smtp.gmail.com`改为你的服务器
2. **修改认证信息**: 更改username和password
3. **修改收件人**: 更改`ToTarget()`中的邮箱地址

## 🧪 本地测试（推荐）

使用MailHog进行无网络测试：

```bash
# 1. 安装MailHog
brew install mailhog

# 2. 启动MailHog
mailhog

# 3. 修改demo配置
hub, err := notifyhub.NewHub(
    email.WithEmail("localhost", 1025, "test@example.com",
        email.WithEmailTLS(false),  // MailHog不需要TLS
    ),
)

# 4. 运行demo
go run main.go

# 5. 查看邮件
open http://localhost:8025
```

## 📋 Demo特点

### ✅ 优势

1. **完全独立** - 每个demo有自己的Hub
2. **互不影响** - 一个失败不影响其他
3. **易于测试** - 可单独运行任意demo
4. **清晰结构** - 每个功能独立封装
5. **易于修改** - 直接修改单个函数

### 🔧 如何扩展

添加新的demo：

```go
// demo11YourFeature demonstrates your feature
func demo11YourFeature() {
    fmt.Println("🎉 Demo 11: Your Feature")
    fmt.Println("=========================")

    ctx := context.Background()

    hub, err := notifyhub.NewHub(
        email.WithEmail("smtp.gmail.com", 587, "your-email@gmail.com",
            email.WithEmailAuth("username", "password"),
            email.WithEmailTLS(true),
        ),
    )
    if err != nil {
        fmt.Printf("❌ Failed: %v\n", err)
        return
    }
    defer hub.Close(ctx)

    // Your code here

    fmt.Println()
}

// 在main()中调用
func main() {
    // ... existing demos ...
    demo11YourFeature()
}
```

## 🐛 故障排查

### 网络连接超时

如果遇到`dial tcp timeout`错误：

1. 检查网络连接：`nc -zv smtp.gmail.com 587`
2. 使用MailHog进行本地测试
3. 检查防火墙设置
4. 参考 [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)

### SMTP认证失败

1. 使用Gmail应用密码而非普通密码
2. 检查SMTP服务器地址和端口
3. 确认TLS/SSL配置正确

## 📚 相关文档

- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - 详细故障排查
- [NETWORK_ISSUE.md](./NETWORK_ISSUE.md) - 网络问题解决方案
- [test_local.go](./test_local.go) - MailHog测试示例
- [README.md](./README.md) - 完整使用指南
