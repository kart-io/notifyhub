# 实施计划

## 阶段1：基础架构重构

- [ ] 1. 删除冗余代码和文件
  - 删除 `pkg/notifyhub/types.go` 中的29个类型别名
  - 删除 `pkg/notifyhub/builders.go` 中的兼容性构建器
  - 删除 `internal/platform/interface.go` 和 `manager.go` 重复接口
  - 保留 `internal/platform/testutil.go` 用于测试
  - _需求：需求1.1, 需求1.4, 需求1.5_

- [ ] 1.1 创建统一消息类型结构
  - 在 `pkg/notifyhub/message/message.go` 中创建统一的 Message 结构
  - 合并当前分散在不同包中的 Message 定义
  - 包含 ID、Title、Body、Format、Priority、Targets、Metadata 等字段
  - 编写消息类型的单元测试
  - _需求：需求1.3, 需求3.3_

- [ ] 1.2 创建统一目标类型结构
  - 在 `pkg/notifyhub/target/target.go` 中创建统一的 Target 结构
  - 定义 Type 枚举（Email、Phone、User、Group、Channel、Webhook）
  - 包含验证和工厂函数
  - 编写目标类型的单元测试
  - _需求：需求1.3, 需求5.3_

- [ ] 1.3 实现统一平台接口
  - 在 `pkg/notifyhub/platform/interface.go` 中定义统一的 Platform 接口
  - 合并 internal 和 public 接口定义
  - 包含 Name、Send、Validate、Capabilities、Health、Close 方法
  - _需求：需求5.1, 需求5.2_

## 阶段2：客户端接口层重构

- [ ] 2. 实现统一客户端接口
  - 在 `pkg/notifyhub/client/client.go` 中创建 Client 接口
  - 支持同步和异步发送方法
  - 包含健康检查和生命周期管理
  - 编写客户端接口的单元测试
  - _需求：需求2.6, 需求3.1_

- [ ] 2.1 创建客户端工厂实现
  - 在 `pkg/notifyhub/client/factory.go` 中实现客户端工厂
  - 替换现有的 621行 `hub_factory.go` 巨型文件
  - 使用函数式选项模式进行配置
  - 编写工厂函数的单元测试
  - _需求：需求1.5, 需求4.1, 需求4.2_

- [ ] 2.2 实现同步客户端
  - 在 `pkg/notifyhub/client/sync_client.go` 中实现同步发送逻辑
  - 直接调用分发器，消除 clientAdapter 冗余层
  - 实现批量发送功能
  - 编写同步发送的单元测试和集成测试
  - _需求：需求3.1, 需求3.2_

## 阶段3：异步处理系统实现

- [ ] 3. 创建异步处理核心组件
  - 在 `pkg/notifyhub/async/` 包中创建异步处理基础结构
  - 包含 Handle、Option、Config 等核心类型
  - _需求：需求2.1, 需求2.6_

- [ ] 3.1 实现异步句柄系统
  - 在 `pkg/notifyhub/async/handle.go` 中实现 AsyncHandle 接口
  - 支持 Wait、Cancel、Status、Result 等操作
  - 实现 BatchHandle 用于批量操作进度跟踪
  - 编写异步句柄的单元测试
  - _需求：需求2.6, 需求2.4_

- [ ] 3.2 创建异步选项和配置系统
  - 在 `pkg/notifyhub/async/options.go` 中定义异步配置选项
  - 实现 WithResultCallback、WithErrorCallback、WithProgressCallback 等
  - 支持超时、优先级、重试策略配置
  - 编写配置选项的单元测试
  - _需求：需求2.2, 需求2.3_

- [ ] 3.3 实现异步队列系统
  - 在 `pkg/notifyhub/async/queue.go` 中创建 AsyncQueue 接口
  - 实现基于内存的队列实现
  - 支持消息入队、出队和状态查询
  - 编写队列系统的单元测试
  - _需求：需求2.1, 需求2.5_

- [ ] 3.4 实现工作池和执行器
  - 在 `pkg/notifyhub/async/worker.go` 中实现 WorkerPool
  - 在 `pkg/notifyhub/async/executor.go` 中实现 AsyncExecutor
  - 支持工作器数量调整和任务执行
  - 编写工作池和执行器的单元测试
  - _需求：需求2.1, 需求2.4_

- [ ] 3.5 创建回调注册和管理系统
  - 在 `pkg/notifyhub/async/callback.go` 中实现 CallbackRegistry
  - 支持全局和消息级回调注册
  - 实现回调触发和清理机制
  - 编写回调系统的单元测试
  - _需求：需求2.2, 需求2.3, 需求2.4_

- [ ] 3.6 实现异步客户端
  - 在 `pkg/notifyhub/client/async_client.go` 中实现异步发送逻辑
  - 集成队列、工作池、回调系统
  - 替换现有的伪异步实现
  - 编写异步客户端的集成测试
  - _需求：需求2.1, 需求2.2, 需求2.3_

## 阶段4：核心分发器重构

- [ ] 4. 实现简化的消息分发器
  - 在 `pkg/notifyhub/core/dispatcher.go` 中创建新的 Dispatcher
  - 实现3层调用链路：Client → Dispatcher → Platform
  - 消除消息格式转换的中间层
  - 编写分发器的单元测试
  - _需求：需求3.1, 需求3.3_

- [ ] 4.1 创建目标路由系统
  - 在 `pkg/notifyhub/target/router.go` 中实现智能路由
  - 根据目标类型自动选择合适的平台
  - 支持负载均衡和故障转移
  - 编写路由系统的单元测试
  - _需求：需求5.3, 需求5.4_

- [ ] 4.2 实现平台管理器
  - 在 `pkg/notifyhub/platform/manager.go` 中实现非全局的平台管理
  - 移除 globalPlatformRegistry，改为 Hub 级别注册
  - 支持动态平台注册和注销
  - 编写平台管理器的单元测试
  - _需求：需求5.1, 需求5.5_

## 阶段5：配置系统统一化

- [ ] 5. 创建统一配置系统
  - 在 `pkg/notifyhub/config/` 包中创建配置管理结构
  - 定义 Config、Option 等核心类型
  - _需求：需求4.1, 需求4.2_

- [ ] 5.1 实现函数式配置选项
  - 在 `pkg/notifyhub/config/options.go` 中定义统一的选项模式
  - 实现 WithPlatform、WithFeishu、WithEmail 等配置函数
  - 支持环境变量和 YAML 配置加载
  - 编写配置选项的单元测试
  - _需求：需求4.2, 需求4.5_

- [ ] 5.2 创建强类型平台配置
  - 为各平台创建强类型配置结构 (EmailConfig, FeishuConfig, SMSConfig)
  - 替换 map[string]interface{} 配置方式
  - 添加配置验证和默认值
  - 编写平台配置的单元测试
  - _需求：需求4.3, 需求9.4_

- [ ] 5.3 实现配置验证系统
  - 在 `pkg/notifyhub/config/validator.go` 中集成验证框架
  - 为所有配置结构添加验证标签
  - 实现统一的配置校验函数
  - 编写配置验证的单元测试
  - _需求：需求6.1, 需求9.4_

## 阶段6：模板管理系统

- [ ] 6. 创建模板管理核心
  - 在 `pkg/notifyhub/template/` 包中创建模板管理结构
  - 定义 Manager、Engine、Cache 等接口
  - _需求：需求8.1, 需求8.2_

- [ ] 6.1 实现多引擎模板支持
  - 在 `pkg/notifyhub/template/manager.go` 中实现模板管理器
  - 支持 Go Template、Mustache、Handlebars 引擎
  - 实现模板注册、渲染和验证
  - 编写模板管理器的单元测试
  - _需求：需求8.1, 需求8.3_

- [ ] 6.2 实现3层缓存系统
  - 在 `pkg/notifyhub/template/cache.go` 中实现缓存接口
  - 支持内存、Redis、数据库3层缓存
  - 实现缓存失效和更新机制
  - 编写缓存系统的单元测试
  - _需求：需求8.2_

- [ ] 6.3 实现热重载机制
  - 扩展模板管理器支持文件监听
  - 实现模板热重载而不重启系统
  - 添加重载事件通知机制
  - 编写热重载的集成测试
  - _需求：需求8.5_

## 阶段7：错误处理和健康监控

- [ ] 7. 创建统一错误处理系统
  - 在 `pkg/notifyhub/errors/` 包中定义统一错误类型
  - 实现 NotifyError、Code 等核心错误结构
  - _需求：需求6.1, 需求6.2_

- [ ] 7.1 实现错误分类和代码系统
  - 在 `pkg/notifyhub/errors/codes.go` 中定义错误代码常量
  - 实现错误分类和上下文信息收集
  - 创建错误工厂函数和包装函数
  - 编写错误处理的单元测试
  - _需求：需求6.1, 需求6.5_

- [ ] 7.2 实现重试策略系统
  - 在 `pkg/notifyhub/errors/retry.go` 中实现重试策略
  - 支持指数退避和抖动算法
  - 集成到异步执行器中
  - 编写重试机制的单元测试
  - _需求：需求6.4_

- [ ] 7.3 创建健康监控系统
  - 在 `pkg/notifyhub/health/` 包中实现健康检查
  - 支持组件级健康状态报告
  - 实现健康检查聚合和监控
  - 编写健康监控的单元测试
  - _需求：需求6.3_

## 阶段8：平台实现更新

- [ ] 8. 更新现有平台实现
  - 更新 Email、Feishu、SMS 平台以实现统一 Platform 接口
  - 移除对旧接口的依赖
  - 使用新的强类型配置
  - _需求：需求5.1, 需求5.5_

- [ ] 8.1 重构 Email 平台实现
  - 更新 `pkg/platforms/email/sender.go` 实现新 Platform 接口
  - 使用 EmailConfig 替换 map 配置
  - 集成错误处理和健康检查
  - 编写 Email 平台的集成测试
  - _需求：需求5.1, 需求5.5_

- [ ] 8.2 重构 Feishu 平台实现
  - 更新 `pkg/platforms/feishu/sender.go` 实现新 Platform 接口
  - 使用 FeishuConfig 替换 map 配置
  - 集成错误处理和健康检查
  - 编写 Feishu 平台的集成测试
  - _需求：需求5.1, 需求5.5_

- [ ] 8.3 重构 SMS 平台实现
  - 更新 `pkg/platforms/sms/sender.go` 实现新 Platform 接口
  - 使用 SMSConfig 替换 map 配置
  - 集成错误处理和健康检查
  - 编写 SMS 平台的集成测试
  - _需求：需求5.1, 需求5.5_

## 阶段9：集成和优化

- [ ] 9. 创建统一入口点
  - 在 `pkg/notifyhub/notifyhub.go` 中实现统一 API 入口
  - 集成所有重构后的组件
  - 提供向后兼容的 API 包装
  - 编写入口点的集成测试
  - _需求：需求3.4, 需求9.3_

- [ ] 9.1 实现回执处理系统
  - 在 `pkg/notifyhub/receipt/` 包中创建回执处理
  - 实现回执收集、聚合和报告
  - 支持异步操作的回执跟踪
  - 编写回执系统的单元测试
  - _需求：需求2.2, 需求2.4_

- [ ] 9.2 创建中间件系统
  - 在 `pkg/notifyhub/middleware/` 包中实现中间件支持
  - 创建日志、指标、重试、限流中间件
  - 支持中间件链和顺序执行
  - 编写中间件系统的单元测试
  - _需求：需求6.4, 需求6.5_

- [ ] 9.3 实现向后兼容层
  - 创建兼容性 API 包装器
  - 提供废弃警告和迁移指导
  - 确保现有用户代码无需修改即可工作
  - 编写兼容性测试用例
  - _需求：需求9.1, 需求9.2, 需求9.3_

## 阶段10：功能验证和测试

- [ ] 10. 验证双层接口消除
  - 编写测试验证不再存在 internal/platform 和 public platform 重复接口
  - 验证消息转换层数从3层减少到1层
  - 测试类型断言次数减少到零（编译时检查）
  - 创建调用链路追踪测试确认只有3层调用
  - _需求：需求1.3, 需求3.1, 需求3.3_

- [ ] 10.1 验证异步处理真实性
  - 编写测试验证 SendAsync 使用真实队列而非同步调用
  - 测试异步操作状态正确反映（pending, processing, completed）
  - 验证回调函数在适当时机被触发（OnResult, OnError, OnProgress）
  - 测试异步操作可以被正确取消和资源清理
  - 验证 AsyncHandle 接口的所有功能正常工作
  - _需求：需求2.1, 需求2.2, 需求2.3, 需求2.4, 需求2.5_

- [ ] 10.2 验证类型安全配置
  - 创建测试确保所有平台使用强类型配置结构
  - 验证编译时类型检查，无运行时类型断言
  - 测试配置验证在编译时捕获错误
  - 验证配置选项的统一性（WithPlatform 模式）
  - _需求：需求4.2, 需求4.3_

- [ ] 10.3 验证全局状态消除
  - 编写测试确认不存在全局 platformRegistry
  - 验证 Hub 级别的平台注册工作正常
  - 测试多个 Hub 实例的隔离性
  - 验证并发测试时无全局状态冲突
  - _需求：需求7.2_

- [ ] 10.4 验证平台发现和能力协商
  - 实现测试验证 ListAvailablePlatforms() API
  - 测试 IsPlatformAvailable() 功能
  - 验证平台能力查询和自动格式转换
  - 测试消息大小检查和自动截断
  - _需求：需求5.2, 需求5.4, 需求8.3, 需求8.4_

- [ ] 10.5 验证性能改进目标
  - 实现基准测试对比重构前后性能指标
  - 验证调用链路从6层减少到3层的性能提升（25-30%）
  - 测量内存分配减少（40%目标）
  - 验证并发性能提升（实例锁 vs 全局锁）
  - 生成性能对比报告
  - _需求：需求10.1, 需求10.4_

- [ ] 10.6 验证可维护性改进
  - 计算代码重复率从30%减少到10%
  - 验证接口数量从2套减少到1套
  - 测试新平台开发时间从4-6小时减少到2-3小时
  - 验证配置复杂度简化（强类型结构 vs map + 类型断言）
  - _需求：需求1.1, 需求1.2, 需求10.2, 需求10.3_

- [ ] 10.7 验证模板系统功能
  - 测试多引擎模板支持（Go Template、Mustache、Handlebars）
  - 验证3层缓存系统（内存、Redis、数据库）
  - 测试模板热重载功能无需系统重启
  - 验证模板变量安全插值和错误处理
  - _需求：需求8.1, 需求8.2, 需求8.4, 需求8.5_

- [ ] 10.8 验证错误处理统一性
  - 测试所有错误使用统一的错误代码分类
  - 验证错误上下文和元数据正确收集
  - 测试重试策略的指数退避和抖动算法
  - 验证健康检查每个组件独立报告状态
  - _需求：需求6.1, 需求6.2, 需求6.3, 需求6.4_

- [ ] 10.9 验证向后兼容性
  - 编写测试确保现有用户代码无需修改
  - 验证废弃 API 提供适当的迁移警告
  - 测试所有核心功能在迁移后工作正常
  - 验证迁移文档的准确性通过自动化测试
  - _需求：需求9.1, 需求9.2, 需求9.3_

- [ ] 10.10 创建全面集成测试
  - 编写端到端测试覆盖完整消息发送流程
  - 测试多平台组合发送场景
  - 验证错误场景和边界条件处理
  - 创建压力测试验证系统稳定性
  - 确保单元测试覆盖率达到90%+
  - _需求：需求10.5_