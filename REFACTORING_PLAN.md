# NotifyHub v2 架构重构计划

## 一、当前问题分析

### 1.1 入口不统一
- **问题**: `pkg/notifyhub` 对外暴露了过多的构建方式
  - `simple_adapter.go`: 示例代码，不应对外暴露
  - `api_adapter.go`: 已废弃的适配器
  - `hub_factory.go`: 实际的构建函数
- **影响**: 增加用户选择和理解成本，API 不清晰

### 1.2 核心职责分散
- **问题**: 核心领域模型散落在多处
  - `pkg/notifyhub/core/`: Hub 实现
  - `internal/platform/`: Platform 接口定义
  - `pkg/notifyhub/message/`: Message 模型
  - `pkg/notifyhub/target/`: Target 模型
- **影响**: 违反单一职责和高内聚原则，难以维护

### 1.3 配置耦合
- **问题**: 调用方需要导入具体平台包
```go
import "github.com/kart-io/notifyhub/pkg/platforms/feishu"
import "github.com/kart-io/notifyhub/pkg/platforms/email"

hub, err := NewHub(
    feishu.WithFeishu(...),  // 直接依赖具体实现
    email.WithEmail(...),
)
```
- **影响**: 上层业务与底层实现耦合，难以动态配置

### 1.4 包结构混乱
- **问题**: pkg 和 internal 职责不清
  - Platform 接口在 `internal/`，实现在 `pkg/`
  - `pkg/utils/logger/`: 更像核心服务而非工具
  - `pkg/utils/`: 包含太多不同职责的代码
- **影响**: 违反 Go 包管理最佳实践，增加理解难度

## 二、目标架构设计

### 2.1 包结构重组
```
notifyhub/
├── pkg/
│   └── notifyhub/           # 纯粹的对外 API 门面
│       ├── client.go        # 统一的客户端入口
│       ├── options.go       # 配置选项（仅公开 API）
│       ├── types.go         # 公开的类型定义
│       └── doc.go           # 包文档
│
├── internal/
│   ├── core/               # 核心领域模型（高内聚）
│   │   ├── hub.go          # Hub 核心实现
│   │   ├── message.go      # Message 领域模型
│   │   ├── target.go       # Target 领域模型
│   │   ├── platform.go     # Platform 接口和管理
│   │   ├── receipt.go      # Receipt 领域模型
│   │   └── router.go       # 路由引擎
│   │
│   ├── factory/            # 工厂模式实现
│   │   ├── platform.go     # 平台工厂（解耦配置）
│   │   └── registry.go     # 平台注册表
│   │
│   ├── services/           # 内部服务（从 utils 提取）
│   │   ├── logger/         # 日志服务
│   │   ├── ratelimit/      # 限流服务
│   │   ├── validator/      # 验证服务
│   │   └── idgen/          # ID 生成服务
│   │
│   └── utils/              # 真正的通用工具
│       └── crypto/         # 加密工具
│
├── platforms/              # 可插拔平台实现（顶层目录）
│   ├── feishu/
│   ├── email/
│   ├── sms/
│   └── slack/
│
└── examples/              # 示例代码
```

### 2.2 统一 API 入口

#### 公开 API（pkg/notifyhub/client.go）
```go
package notifyhub

// Client 是 NotifyHub 的唯一入口
type Client interface {
    Send(ctx context.Context, message *Message) (*Receipt, error)
    SendBatch(ctx context.Context, messages []*Message) ([]*Receipt, error)
    Health(ctx context.Context) (*HealthStatus, error)
    Close() error
}

// New 创建客户端的唯一函数
func New(opts ...Option) (Client, error) {
    // 内部调用 internal/core
    return internal.NewHub(opts...)
}

// NewFromConfig 从配置创建（完全解耦）
func NewFromConfig(config Config) (Client, error) {
    return internal.NewHubFromConfig(config)
}
```

#### 配置解耦（pkg/notifyhub/options.go）
```go
// Option 配置选项
type Option func(*Config)

// WithPlatform 通用平台配置（不需要导入具体包）
func WithPlatform(name string, config map[string]interface{}) Option {
    return func(c *Config) {
        c.Platforms[name] = config
    }
}

// WithYAML 从 YAML 配置文件加载
func WithYAML(path string) Option {
    return func(c *Config) {
        // 读取并解析 YAML
    }
}

// WithJSON 从 JSON 配置加载
func WithJSON(data []byte) Option {
    return func(c *Config) {
        // 解析 JSON
    }
}
```

### 2.3 工厂模式解耦

#### 平台工厂（internal/factory/platform.go）
```go
type PlatformFactory struct {
    registry map[string]PlatformCreator
}

type PlatformCreator func(config map[string]interface{}) (platform.Sender, error)

// CreatePlatform 根据配置创建平台
func (f *PlatformFactory) CreatePlatform(name string, config map[string]interface{}) (platform.Sender, error) {
    // 自动发现或注册的创建器
    creator, ok := f.registry[name]
    if !ok {
        // 尝试动态加载
        return f.tryDynamicLoad(name, config)
    }
    return creator(config)
}

// tryDynamicLoad 尝试动态加载平台
func (f *PlatformFactory) tryDynamicLoad(name string, config map[string]interface{}) (platform.Sender, error) {
    // 使用插件系统或反射加载
    // 例如：查找 platforms/{name}/init.go 中的 init 函数
}
```

### 2.4 配置驱动示例

#### YAML 配置文件
```yaml
platforms:
  feishu:
    type: feishu
    webhook: ${FEISHU_WEBHOOK}
    secret: ${FEISHU_SECRET}

  email:
    type: email
    host: smtp.example.com
    port: 587
    from: noreply@example.com
    username: ${EMAIL_USER}
    password: ${EMAIL_PASS}

routing:
  rules:
    - condition: "priority >= 4"
      platforms: ["feishu", "email"]
    - condition: "type == 'alert'"
      platforms: ["feishu"]

retry:
  max_attempts: 3
  backoff: exponential
  initial_interval: 1s
  max_interval: 30s
```

#### 使用示例
```go
// 方式 1: 配置文件
client, err := notifyhub.NewFromConfig(notifyhub.Config{
    ConfigFile: "config.yaml",
})

// 方式 2: 代码配置（无需导入具体平台包）
client, err := notifyhub.New(
    notifyhub.WithPlatform("feishu", map[string]interface{}{
        "webhook": os.Getenv("FEISHU_WEBHOOK"),
        "secret":  os.Getenv("FEISHU_SECRET"),
    }),
    notifyhub.WithPlatform("email", map[string]interface{}{
        "host": "smtp.example.com",
        "port": 587,
    }),
)

// 方式 3: 混合配置
client, err := notifyhub.New(
    notifyhub.WithYAML("base-config.yaml"),
    notifyhub.WithPlatform("custom", customConfig),
)
```

## 三、实施步骤

### Phase 1: 准备阶段（不破坏现有 API）
1. **创建新的内部结构**
   - [ ] 创建 `internal/core/` 目录
   - [ ] 复制并整合核心领域模型到 `internal/core/`
   - [ ] 创建 `internal/factory/` 实现工厂模式
   - [ ] 创建 `internal/services/` 目录

2. **重构内部实现**
   - [ ] 将 Hub, Message, Target 等整合到 `internal/core/`
   - [ ] 实现平台工厂和注册机制
   - [ ] 将 logger, ratelimit 移至 `internal/services/`

### Phase 2: 创建新 API（并存阶段）
1. **实现新的客户端接口**
   - [ ] 创建 `pkg/notifyhub/client.go`
   - [ ] 实现统一的 `New()` 函数
   - [ ] 实现配置驱动的选项

2. **添加向后兼容层**
   - [ ] 保留旧的 API，标记为 deprecated
   - [ ] 内部重定向到新实现
   - [ ] 添加迁移提示

### Phase 3: 平台迁移
1. **重组平台包**
   - [ ] 创建顶层 `platforms/` 目录
   - [ ] 逐个迁移平台实现
   - [ ] 实现自注册机制

2. **更新示例和文档**
   - [ ] 更新所有示例代码
   - [ ] 编写迁移指南
   - [ ] 更新 README 和文档

### Phase 4: 清理阶段
1. **移除废弃代码**
   - [ ] 删除 `simple_adapter.go`
   - [ ] 删除 `api_adapter.go`
   - [ ] 清理未使用的代码

2. **发布新版本**
   - [ ] 标记为 v2.0.0
   - [ ] 发布迁移指南
   - [ ] 提供升级工具

## 四、向后兼容策略

### 4.1 废弃通知
```go
// Deprecated: Use notifyhub.New() instead
// Will be removed in v3.0.0
func NewHub(opts ...HubOption) (Hub, error) {
    // 内部调用新 API
    return New(adaptOldOptions(opts...))
}
```

### 4.2 迁移助手
```go
// MigrateConfig 将旧配置转换为新配置
func MigrateConfig(old OldConfig) Config {
    // 自动转换逻辑
}
```

### 4.3 兼容性测试
```go
// 确保所有旧测试仍然通过
func TestBackwardCompatibility(t *testing.T) {
    // 测试旧 API 仍然工作
}
```

## 五、收益分析

### 5.1 架构改进
- ✅ **统一入口**: 单一清晰的 API
- ✅ **高内聚**: 核心领域模型集中管理
- ✅ **低耦合**: 配置与实现完全解耦
- ✅ **清晰边界**: pkg/internal 职责明确

### 5.2 开发体验
- ✅ **简化使用**: 不需要导入具体平台包
- ✅ **灵活配置**: 支持多种配置方式
- ✅ **易于扩展**: 插件式平台架构
- ✅ **更好的 IDE 支持**: 清晰的包结构

### 5.3 维护性
- ✅ **模块化**: 各模块独立演进
- ✅ **可测试性**: 依赖注入，易于测试
- ✅ **文档友好**: 结构清晰，易于理解
- ✅ **版本管理**: 清晰的版本边界

## 六、风险和缓解

### 6.1 风险
1. **破坏性变更**: 影响现有用户
2. **迁移成本**: 用户需要更新代码
3. **学习曲线**: 新的 API 需要学习

### 6.2 缓解措施
1. **渐进式迁移**: 保持向后兼容
2. **自动化工具**: 提供迁移脚本
3. **详细文档**: 完整的迁移指南
4. **社区支持**: 积极响应用户反馈

## 七、时间表

| 阶段 | 时间 | 里程碑 |
|-----|------|-------|
| Phase 1 | Week 1-2 | 内部重构完成 |
| Phase 2 | Week 3-4 | 新 API 实现 |
| Phase 3 | Week 5-6 | 平台迁移 |
| Phase 4 | Week 7-8 | 清理和发布 |

## 八、成功标准

1. **代码质量**
   - [ ] 测试覆盖率 > 80%
   - [ ] 无破坏性变更（v1 API 仍可用）
   - [ ] 性能无退化

2. **用户体验**
   - [ ] API 更简洁
   - [ ] 配置更灵活
   - [ ] 文档更完善

3. **架构目标**
   - [ ] 包结构清晰
   - [ ] 职责单一
   - [ ] 易于扩展

## 九、下一步行动

1. **立即开始**
   - 创建 feature/v2-refactoring 分支
   - 开始 Phase 1 的内部重构
   - 编写详细的技术设计文档

2. **团队协作**
   - 评审重构计划
   - 分配开发任务
   - 制定测试策略

3. **社区沟通**
   - 发布重构计划 RFC
   - 收集用户反馈
   - 准备迁移支持

---

*本文档为 NotifyHub v2 架构重构的指导文件，将根据实施过程中的反馈持续更新。*