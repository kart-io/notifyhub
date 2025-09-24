# Email Platform 示例索引

## 📚 文档导航

### 🚀 快速开始
1. **[HOW_TO_RUN.md](./HOW_TO_RUN.md)** - 如何运行示例
   - 运行方式说明
   - Build tags解释
   - 常见问题解答

2. **[README.md](./README.md)** - 完整使用指南
   - 功能概述
   - 配置说明
   - 网络要求

### 📋 Demo说明
3. **[DEMOS.md](./DEMOS.md)** - 独立Demo详解
   - 10个独立demo列表
   - 每个demo的功能说明
   - 配置和扩展指南

### 🐛 问题排查
4. **[TROUBLESHOOTING.md](./TROUBLESHOOTING.md)** - 故障排查指南
   - 网络连接超时
   - 认证失败
   - TLS/SSL配置
   - SMTP提供商配置

5. **[NETWORK_ISSUE.md](./NETWORK_ISSUE.md)** - 网络问题专题
   - 问题分析
   - 解决方案
   - MailHog本地测试

6. **[SOLUTION_SUMMARY.md](./SOLUTION_SUMMARY.md)** - 完整解决方案
   - 问题诊断过程
   - 修复内容
   - 技术细节

## 💻 代码文件

### 主程序
- **[main.go](./main.go)** - 10个独立demo（468行）
  - demo1: 基础SMTP配置
  - demo2: 认证SMTP with TLS
  - demo3: SSL配置
  - demo4: 简单文本邮件
  - demo5: HTML邮件
  - demo6: 优先级邮件
  - demo7: CC收件人
  - demo8: 模板邮件
  - demo9: 多收件人
  - demo10: 不同消息类型

### 测试程序
- **[test_local.go](./test_local.go)** - MailHog本地测试
  - 使用`//go:build ignore`标记
  - 单独运行：`go run test_local.go`
  - 无需真实SMTP服务器

### 工具脚本
- **[setup_mailhog.sh](./setup_mailhog.sh)** - 自动化设置
  - 安装MailHog
  - 启动服务
  - 运行测试

## 🎯 使用场景

### 场景1: 学习Email功能
```bash
# 1. 阅读文档
cat HOW_TO_RUN.md

# 2. 运行所有demo
go run main.go

# 3. 查看demo详解
cat DEMOS.md
```

### 场景2: 本地开发测试
```bash
# 1. 安装MailHog
brew install mailhog

# 2. 启动MailHog
mailhog &

# 3. 运行本地测试
go run test_local.go

# 4. 查看邮件
open http://localhost:8025
```

### 场景3: 排查问题
```bash
# 1. 查看故障排查指南
cat TROUBLESHOOTING.md

# 2. 查看网络问题
cat NETWORK_ISSUE.md

# 3. 查看完整解决方案
cat SOLUTION_SUMMARY.md
```

### 场景4: 生产环境配置
```bash
# 1. 阅读README了解配置
cat README.md

# 2. 修改main.go中的SMTP配置
# 3. 测试单个demo
go run main.go
```

## 🔑 关键概念

### Main函数设计
- ✅ **main.go**: 正常编译，包含主程序
- ✅ **test_local.go**: 使用`//go:build ignore`，编译时忽略
- 这样避免了"multiple main functions"冲突

### Demo独立性
- 每个demo创建自己的Hub
- 互不影响，可单独运行
- 便于测试和调试

### 网络处理
- 支持Gmail、Outlook等多种SMTP
- 提供MailHog本地测试方案
- 详细的网络问题排查指南

## 📊 文件大小

```
main.go                  14KB  (主程序)
test_local.go            4.6KB (本地测试)
README.md                8.6KB (使用指南)
TROUBLESHOOTING.md       6.1KB (故障排查)
NETWORK_ISSUE.md         4.4KB (网络问题)
SOLUTION_SUMMARY.md      6.7KB (解决方案)
DEMOS.md                 4.2KB (Demo说明)
HOW_TO_RUN.md            3.0KB (运行说明)
```

## 🚀 快速命令

```bash
# 编译主程序
go build -o email_demo

# 运行主程序
./email_demo

# 运行本地测试
go run test_local.go

# 检查代码
go fmt ./...
go vet ./...

# 查看帮助
cat HOW_TO_RUN.md
```

## 📞 获取帮助

1. 先查看 **[HOW_TO_RUN.md](./HOW_TO_RUN.md)**
2. 如有问题查看 **[TROUBLESHOOTING.md](./TROUBLESHOOTING.md)**
3. 网络问题查看 **[NETWORK_ISSUE.md](./NETWORK_ISSUE.md)**
4. 完整方案查看 **[SOLUTION_SUMMARY.md](./SOLUTION_SUMMARY.md)**