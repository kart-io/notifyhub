# 飞书通知示例

这是 NotifyHub 飞书（Lark）通知功能的完整示例集合，展示了最新的 v2 架构。

## 📁 项目结构

```
examples/feishu/
├── basic/                       # 基础示例 - 新手入门
│   ├── auth-modes/             # 认证模式演示
│   │   └── main.go            # 可执行程序
│   ├── complete-example/       # 完整功能演示
│   │   └── main.go            # 可执行程序
│   └── README.md              # 基础示例说明
├── advanced/                    # 高级示例 - 进阶功能
│   ├── comprehensive.go       # 综合高级特性演示
│   └── README.md              # 高级示例说明
├── tools/                       # 工具和调试
│   ├── debug/                  # 网络调试工具
│   │   ├── main.go            # 直接HTTP请求测试
│   │   └── go.mod             # 独立模块
│   ├── test/                   # 集成测试工具
│   │   ├── main.go            # 完整集成测试
│   │   └── go.mod             # 独立模块
│   └── README.md              # 工具使用说明
├── docs/                        # 文档
│   ├── README.md              # 详细文档
│   ├── QUICKSTART.md          # 快速开始指南
│   └── TROUBLESHOOTING.md     # 故障排除指南
├── scripts/                     # 脚本和工具
│   ├── Makefile               # 构建脚本
│   └── demo.sh                # 演示脚本
├── go.mod                       # 统一模块管理
└── README.md                    # 本文件
```

## 🆕 新版本特性

### 最新架构 (v2)
- 基于 `pkg/notifyhub` 包的新架构
- 支持三种飞书认证模式：无认证、签名认证、关键词认证
- 改进的消息构建器和目标系统
- 更好的错误处理和状态报告
- 完整的健康检查和监控支持

## 🚀 快速开始

### 1. 环境配置

```bash
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your-token"
export FEISHU_SECRET="your-secret"  # 可选
```

### 2. 运行示例

#### 🎯 推荐学习路径

```bash
# 1. 新手入门 - 基础示例
cd basic
go run auth-modes/main.go          # 学习认证模式
go run complete-example/main.go    # 体验完整功能

# 2. 进阶学习 - 高级特性
cd advanced
go run comprehensive.go      # 探索高级功能

# 3. 调试工具 - 问题排查
cd tools/debug
go run main.go              # 网络调试

cd tools/test
go run main.go              # 集成测试
```

#### 📋 快速命令参考

```bash
# 从项目根目录运行
go run basic/auth-modes/main.go
go run basic/complete-example/main.go
go run advanced/comprehensive.go

# 工具使用（需要进入对应目录）
cd tools/debug && go run main.go
cd tools/test && go run main.go
```

## 📋 各示例功能说明

### 🎓 基础示例 (`basic/`)

#### `auth-modes/` - 认证模式演示
- 演示三种飞书认证模式：无认证、签名认证、关键词认证
- 展示显式设置认证模式的方法
- 包含配置错误处理示例
- **适合**: 新手学习认证配置

#### `complete-example/` - 完整功能演示
- 基于最新架构的6种飞书通知场景
- 简化的API使用和错误处理
- 完整的生命周期管理
- **适合**: 快速了解所有基础功能

### 🚀 高级示例 (`advanced/`)

#### `comprehensive.go` - 综合高级特性演示
- 消息模板和变量替换
- 飞书卡片消息（复杂交互式卡片）
- @提及用户和@所有人功能
- 批量发送和异步发送
- 系统健康检查和消息优先级设置
- **适合**: 企业级应用开发参考

### 🔧 工具和调试 (`tools/`)

#### `debug/` - 网络调试工具
- 直接发送HTTP请求到飞书API
- 测试网络连通性和签名验证
- 详细的请求/响应日志输出
- **适合**: 网络问题排查和API调试

#### `test/` - 集成测试工具
- 测试NotifyHub与飞书的完整集成
- 系统健康检查和性能测试
- 全面的错误诊断和回归测试
- **适合**: 部署验证和持续集成

## 📚 详细文档

- [详细使用指南](docs/README.md)
- [快速开始](docs/QUICKSTART.md)
- [故障排除](docs/TROUBLESHOOTING.md)

## 🔧 构建说明

### 📦 统一模块管理
项目现在使用统一的 `go.mod` 文件进行依赖管理，简化了构建过程：

```bash
# 在项目根目录构建所有示例
go build -o bin/auth-modes basic/auth_modes.go
go build -o bin/complete-example basic/complete_example.go
go build -o bin/comprehensive advanced/comprehensive.go

# 创建输出目录
mkdir -p bin/

# 批量构建
go build -o bin/auth-modes basic/auth_modes.go
go build -o bin/complete-example basic/complete_example.go
go build -o bin/comprehensive advanced/comprehensive.go
```

### 🛠️ 工具独立构建
调试和测试工具保持独立模块，需要在对应目录构建：

```bash
# 构建调试工具
cd tools/debug && go build -o debug-sender main.go

# 构建测试工具
cd tools/test && go build -o integration-test main.go
```

### 📋 构建脚本
使用 Makefile 进行批量操作：

```bash
cd scripts
make build     # 构建所有示例
make clean     # 清理构建产物
make test      # 运行测试
```

## 💡 使用建议

### 🎯 学习路径

#### 新手用户 (0-1小时)
1. **环境配置**: 设置 `FEISHU_WEBHOOK_URL` 环境变量
2. **认证学习**: 运行 `basic/auth-modes/main.go` 了解认证模式
3. **功能体验**: 运行 `basic/complete-example/main.go` 体验完整功能

#### 进阶用户 (1-2小时)
1. **高级特性**: 运行 `advanced/comprehensive.go` 探索高级功能
2. **代码研读**: 分析高级示例中的最佳实践
3. **集成应用**: 将示例代码集成到自己的项目中

#### 开发者/运维 (随时使用)
1. **调试工具**: 使用 `tools/debug/` 排查网络问题
2. **集成测试**: 使用 `tools/test/` 验证部署
3. **持续监控**: 实施健康检查和告警机制

### 🔧 问题排查

| 问题类型 | 推荐工具 | 说明 |
|---------|---------|------|
| 认证失败 | `basic/auth-modes/` | 检查认证模式配置 |
| 网络连接 | `tools/debug/` | 直接HTTP测试 |
| 功能异常 | `tools/test/` | 完整集成验证 |
| 性能问题 | `advanced/comprehensive.go` | 批量和异步发送参考 |

### 🏭 生产环境最佳实践

- ✅ **使用统一模块**: 基于根目录的 `go.mod` 进行依赖管理
- ✅ **错误处理**: 参考 `advanced/comprehensive.go` 中的错误处理模式
- ✅ **健康监控**: 实施定期健康检查和指标收集
- ✅ **配置管理**: 使用环境变量管理敏感配置信息
- ✅ **日志记录**: 记录发送结果和错误信息用于调试

## ⚡ 快速演示

使用脚本快速体验：

```bash
cd scripts && ./demo.sh
```