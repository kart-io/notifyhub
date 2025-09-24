# 如何运行Email Platform示例

## 📋 文件说明

```
examples/platforms/email/
├── main.go          # 主程序 - 包含10个独立demo
├── test_local.go    # MailHog本地测试 (需单独运行)
└── *.md            # 文档
```

## 🚀 运行方式

### 1. 运行主程序（所有demos）

```bash
# 方式1: 直接运行
go run main.go

# 方式2: 编译后运行
go build -o email_demo
./email_demo
```

这将执行所有10个独立的demo。

### 2. 运行单个Demo

修改`main.go`的`main()`函数，只保留想要运行的demo：

```go
func main() {
    fmt.Println("📧 Email Platform - Single Demo")

    // 只运行Demo 4
    demo4SimpleTextEmail()
}
```

然后：
```bash
go run main.go
```

### 3. 运行MailHog本地测试

`test_local.go` 有自己的main函数，需要单独运行：

```bash
# 1. 启动MailHog
brew install mailhog
mailhog &

# 2. 运行测试（使用 -tags 或直接指定文件）
go run test_local.go

# 3. 查看邮件
open http://localhost:8025
```

## 🔧 为什么test_local.go能单独运行？

`test_local.go` 使用了build tag：

```go
//go:build ignore
// +build ignore

package main
```

这些标记告诉Go编译器：
- ❌ `go build` 时忽略此文件
- ❌ `go run .` 时忽略此文件
- ✅ `go run test_local.go` 可以直接运行

## 📝 常见用法

### 用法1: 快速测试所有功能
```bash
go run main.go
```

### 用法2: 测试单个功能
编辑`main.go`，只保留需要的demo函数调用。

### 用法3: 本地测试（无需真实SMTP）
```bash
mailhog &
go run test_local.go
```

### 用法4: 创建自定义Demo
在`main.go`中添加新函数：

```go
func demo11MyCustomDemo() {
    // Your code here
}

func main() {
    demo11MyCustomDemo()
}
```

## 🐛 常见问题

### Q: 为什么有两个main函数不冲突？

**A:** 因为`test_local.go`使用了`//go:build ignore`标记，正常编译时会被忽略。

### Q: 如何只运行某个demo？

**A:** 两种方式：
1. 修改`main()`函数，只调用需要的demo
2. 将demo函数复制到新文件，创建独立的main

### Q: test_local.go是什么？

**A:** 它是用于MailHog本地测试的独立程序，使用`go run test_local.go`单独运行。

### Q: 如何添加更多demo？

**A:** 在`main.go`中：
1. 添加新的`demoXX`函数
2. 在`main()`中调用它

## 📚 推荐学习顺序

1. **先运行main.go** - 查看所有功能演示
2. **阅读DEMOS.md** - 了解每个demo的详细说明
3. **运行test_local.go** - 使用MailHog进行本地测试
4. **修改main()** - 只运行感兴趣的demo
5. **创建自定义demo** - 基于现有demo修改

## ⚙️ Build Tags说明

### main.go
```go
package main  // 正常编译
```
- ✅ `go build` 会编译
- ✅ `go run .` 会运行
- ✅ `go run main.go` 会运行

### test_local.go
```go
//go:build ignore
package main  // 编译时忽略
```
- ❌ `go build` 会忽略
- ❌ `go run .` 会忽略
- ✅ `go run test_local.go` 可以运行

这样设计避免了"multiple main functions"的冲突！