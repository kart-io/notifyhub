# 基础示例

这个目录包含 NotifyHub 飞书集成的基础示例，适合新手学习和快速上手。

## 📋 示例列表

### `auth-modes/` - 认证模式演示
演示飞书的三种认证模式：
- **无认证模式** (`auth_mode: "none"`)
- **签名认证模式** (`auth_mode: "signature"`)
- **关键词认证模式** (`auth_mode: "keywords"`)

**运行方式：**
```bash
# 设置环境变量
export FEISHU_WEBHOOK_URL="你的飞书webhook地址"
export FEISHU_SECRET="你的签名密钥"  # 可选

# 进入目录运行
cd auth-modes
go run main.go

# 或者从父目录运行
go run auth-modes/main.go
```

### `complete-example/` - 完整功能演示
展示飞书通知的6种典型使用场景：
- 简单文本消息
- Markdown格式消息
- 飞书卡片消息
- 批量发送
- 异步发送
- 系统健康检查

**运行方式：**
```bash
# 进入目录运行
cd complete-example
go run main.go

# 或者从父目录运行
go run complete-example/main.go
```

## 🎯 学习路径

1. **新手**: 先运行 `auth-modes/main.go` 了解认证配置
2. **进阶**: 运行 `complete-example/main.go` 学习完整功能
3. **实战**: 参考代码集成到你的项目中

## 🏗️ 构建说明

每个子目录都是独立的可执行程序，可以单独构建：

```bash
# 构建认证模式示例
cd auth-modes
go build -o auth-modes main.go
./auth-modes

# 构建完整功能示例
cd complete-example
go build -o complete-example main.go
./complete-example

# 或者从父目录构建
go build -o auth-modes auth-modes/main.go
go build -o complete-example complete-example/main.go
```

## 📚 相关文档

- [认证模式详细说明](../docs/README.md)
- [API参考文档](https://docs.example.com/notifyhub)
- [故障排除指南](../docs/TROUBLESHOOTING.md)