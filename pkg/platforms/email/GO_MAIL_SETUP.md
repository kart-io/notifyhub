# go-mail 安装指南

## 📦 安装步骤

### 方式 1: 直接安装（推荐）

```bash
go get -u github.com/wneessen/go-mail
```

### 方式 2: 使用国内代理

如果遇到网络问题：

```bash
# 设置代理
export GOPROXY=https://goproxy.cn,direct

# 或者使用阿里云代理
export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 安装
go get -u github.com/wneessen/go-mail
```

### 方式 3: 添加到 go.mod

在项目根目录的 `go.mod` 文件中添加：

```go
require (
    github.com/wneessen/go-mail v0.4.1  // 使用最新版本
)
```

然后运行：

```bash
go mod download
go mod tidy
```

## 🔧 验证安装

### 检查依赖

```bash
go list -m github.com/wneessen/go-mail
```

期望输出：
```
github.com/wneessen/go-mail v0.4.1
```

### 简单测试

创建测试文件 `test_gomail.go`:

```go
package main

import (
    "context"
    "fmt"
    "github.com/wneessen/go-mail"
)

func main() {
    m := mail.NewMsg()
    m.From("test@example.com")
    m.To("recipient@example.com")
    m.Subject("Test")
    m.SetBodyString(mail.TypeTextPlain, "Hello from go-mail!")

    client, err := mail.NewClient("smtp.example.com",
        mail.WithPort(587),
        mail.WithSMTPAuth(mail.SMTPAuthPlain),
        mail.WithUsername("user"),
        mail.WithPassword("pass"),
    )

    if err != nil {
        fmt.Printf("Failed to create client: %v\n", err)
        return
    }

    fmt.Println("✅ go-mail installed successfully!")
}
```

运行测试：

```bash
go run test_gomail.go
```

## 📋 依赖版本

### 推荐版本

```
github.com/wneessen/go-mail v0.4.1 或更高
```

### 检查最新版本

```bash
go list -m -versions github.com/wneessen/go-mail
```

## 🐛 常见问题

### 问题 1: 网络超时

**错误：**
```
dial tcp: i/o timeout
```

**解决：**
```bash
# 使用代理
export GOPROXY=https://goproxy.cn,direct
go get -u github.com/wneessen/go-mail
```

### 问题 2: 依赖冲突

**错误：**
```
conflicts with other requirements
```

**解决：**
```bash
go clean -modcache
go mod tidy
go get -u github.com/wneessen/go-mail
```

### 问题 3: 版本不兼容

**解决：**
```bash
# 安装特定版本
go get github.com/wneessen/go-mail@v0.4.1

# 或更新到最新
go get -u github.com/wneessen/go-mail@latest
```

## 🔄 降级到 net/smtp

如果 go-mail 安装失败，可以临时使用 net/smtp：

```go
import "github.com/kart-io/notifyhub/pkg/platforms/email"

func main() {
    // 使用 net/smtp（不需要额外依赖）
    email.UseNetSMTP()

    // 其他代码保持不变
    hub, err := notifyhub.NewHub(
        email.WithEmail("smtp.gmail.com", 587, "from@example.com",
            email.WithEmailAuth("user", "pass"),
            email.WithEmailTLS(true),
        ),
    )
}
```

## 📊 NotifyHub 集成

### 当前状态检查

```bash
# 检查当前使用的实现
go run -tags debug examples/platforms/email/main.go
```

### 切换实现

**使用 go-mail（默认）：**
```go
// 无需任何操作，默认就是 go-mail
```

**使用 net/smtp：**
```go
import "github.com/kart-io/notifyhub/pkg/platforms/email"

func init() {
    email.UseNetSMTP()
}
```

## 🎯 下一步

安装完成后：

1. 阅读 [MIGRATION_GOMAIL.md](./MIGRATION_GOMAIL.md) 了解迁移指南
2. 查看 [sender_gomail.go](./sender_gomail.go) 了解实现细节
3. 运行 [examples/platforms/email/main.go](../../../examples/platforms/email/main.go) 测试功能

## 📞 获取帮助

- **go-mail 问题**: https://github.com/wneessen/go-mail/issues
- **NotifyHub 问题**: 查看项目文档
- **网络问题**: 使用代理或镜像源