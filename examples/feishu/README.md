# 飞书 (Feishu) 发送示例

本目录包含了使用 NotifyHub 发送飞书消息的完整示例，涵盖了从基础用法到高级功能的各种场景。

## 📋 目录结构

```
feishu/
├── README.md                    # 本文档
├── basic/                       # 基础发送示例
│   ├── main.go                 # 基础功能代码
│   └── go.mod                  # 模块配置
├── advanced/                    # 高级功能示例
│   ├── main.go                 # 高级功能代码
│   └── go.mod                  # 模块配置
├── batch/                       # 批量发送示例
│   ├── main.go                 # 批量发送代码
│   └── go.mod                  # 模块配置
├── quick-demo/                  # 快速功能验证
│   ├── main.go                 # 快速演示代码
│   └── go.mod                  # 模块配置
├── test.sh                      # 测试脚本
└── EXAMPLES_SUMMARY.md          # 示例总结
```

## 🚀 快速开始

### 环境准备

1. **获取飞书机器人 Webhook**
   - 在飞书群聊中添加机器人
   - 获取 Webhook URL 和密钥
   - 设置环境变量：

```bash
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"
export FEISHU_SECRET="your-webhook-secret"
```

2. **安装依赖**
```bash
cd examples/feishu
# 为每个示例安装依赖
cd basic && go mod tidy && cd ..
cd advanced && go mod tidy && cd ..
cd batch && go mod tidy && cd ..
cd quick-demo && go mod tidy && cd ..
```

3. **运行示例**
```bash
# 快速功能验证
cd quick-demo && go run main.go

# 基础发送示例
cd basic && go run main.go

# 高级功能示例
cd advanced && go run main.go

# 批量发送示例
cd batch && go run main.go
```

## 📚 示例说明

### 🔰 基础示例 (basic/main.go)

演示了飞书消息发送的基本用法：

- **简单文本消息发送**
  ```go
  err = hub.FeishuGroup(ctx, "系统通知", "Hello，这是一条测试消息！", "your-group-id")
  ```

- **富文本消息构建**
  ```go
  message := client.NewMessage("📢 系统公告", "系统维护通知").
      Format(notifiers.FormatText).
      Priority(notifiers.PriorityHigh).
      FeishuGroup("your-group-id").
      Build()
  ```

- **警报消息发送**
  ```go
  alert := client.NewAlert("🚨 系统警报", "CPU 使用率超过 85%").
      Variable("server", "web-server-01").
      Variable("cpu_usage", "87.5%").
      FeishuGroup("your-ops-group-id").
      Build()
  ```

- **个人通知**
  ```go
  personalNotice := client.NewNotice("📋 任务提醒", "您有新的代码审查任务").
      FeishuUser("your-user-id").
      Build()
  ```

- **快捷发送方法**
  ```go
  err = hub.QuickSend(ctx, "快速通知", "消息内容", "group:group-id@feishu")
  ```

**适用场景：**
- 系统通知
- 简单警报
- 日常消息推送
- 快速集成测试

### ⚡ 高级示例 (advanced/main.go)

展示了 NotifyHub 的高级功能在飞书场景中的应用：

- **自定义模板使用**
  ```go
  templates.AddTextTemplate("incident_alert", `🚨 **紧急事件通知**
  **事件级别:** {{.severity | upper}}
  **影响服务:** {{.service}}
  **开始时间:** {{formatTime .start_time "2006-01-02 15:04:05"}}`)
  ```

- **消息路由和优先级**
  - 不同优先级消息自动路由到对应群组
  - 支持 Low、Normal、High、Urgent 四个级别

- **重试机制和错误处理**
  ```go
  retryOptions := &client.Options{
      Retry:      true,
      MaxRetries: 3,
      Timeout:    10 * time.Second,
  }
  ```

- **回调和监控**
  ```go
  successCallback := queue.NewCallbackFunc("success-logger", func(ctx context.Context, callbackCtx *queue.CallbackContext) error {
      // 处理发送成功回调
      return nil
  })
  ```

- **延迟发送**
  ```go
  delayedMessage := client.NewMessage("延迟消息", "内容").
      Delay(5 * time.Second).
      FeishuGroup("delayed-messages").
      Build()
  ```

**适用场景：**
- 企业级通知系统
- 事件驱动的告警
- 复杂业务流程通知
- 需要监控和回调的场景

### 🔄 批量示例 (batch/main.go)

专门针对大规模、高频率的飞书消息发送场景：

- **基础批量发送**
  ```go
  batch := hub.NewEnhancedBatch()
  for _, content := range messages {
      message := client.NewMessage(title, content).FeishuGroup(groupID).Build()
      batch.Add(message, options)
  }
  results, err := batch.Send(ctx)
  ```

- **分组批量发送**
  - 为不同团队创建独立批次
  - 并行发送提高效率

- **并发批量处理**
  - 多个批次并发执行
  - 智能负载分配

- **混合类型批量发送**
  - 同一批次包含不同类型消息（警报、通知、报告）
  - 统一处理不同优先级

- **大规模批量发送**
  - 支持数百条消息的批量处理
  - 自动分批避免系统压力
  - 实时统计发送进度

- **智能批量发送**
  - 按优先级自动排序
  - 紧急消息优先处理
  - 根据优先级调整重试策略

**适用场景：**
- 大规模用户通知
- 定时批量报告
- 多团队协作通知
- 高并发告警场景

## 🎯 使用建议

### 配置建议

1. **队列配置**
   ```go
   // 普通应用
   config.WithQueue("memory", 1000, 4)

   // 高频应用
   config.WithQueue("memory", 5000, 16)
   ```

2. **速率限制**
   ```go
   // 避免触发飞书限制
   config.WithRateLimit(10, 5)  // 每秒10次，突发5次
   ```

3. **重试策略**
   ```go
   // 重要消息
   &client.Options{
       Retry:      true,
       MaxRetries: 5,
       Timeout:    60 * time.Second,
   }

   // 普通消息
   &client.Options{
       Retry:      true,
       MaxRetries: 2,
       Timeout:    15 * time.Second,
   }
   ```

### 性能优化

1. **批量发送优化**
   - 单批次建议不超过 50 条消息
   - 使用异步发送提高吞吐量
   - 合理设置批次间延迟

2. **并发控制**
   - 根据飞书 API 限制调整并发数
   - 使用速率限制避免被限流
   - 监控发送成功率

3. **错误处理**
   - 实现重试机制
   - 记录失败消息便于排查
   - 设置合理的超时时间

### 监控和调试

1. **发送统计**
   ```go
   metrics := hub.GetMetrics()
   successRate := metrics["success_rate"].(float64)
   totalSent := metrics["total_sent"].(int64)
   ```

2. **健康检查**
   ```go
   health := hub.GetHealth(ctx)
   // 检查飞书平台健康状态
   ```

3. **日志记录**
   - 启用详细日志便于调试
   - 记录关键操作的耗时
   - 监控异常情况

## ⚠️ 注意事项

### 飞书 API 限制

1. **频率限制**
   - 建议每秒不超过 10 次请求
   - 避免短时间内大量发送

2. **消息大小**
   - 单条消息不超过 8KB
   - 长消息建议分段发送

3. **群组权限**
   - 确保机器人已加入目标群组
   - 验证机器人具有发送权限

### 安全考虑

1. **密钥保护**
   - 不要在代码中硬编码密钥
   - 使用环境变量或密钥管理服务

2. **消息内容**
   - 避免发送敏感信息
   - 对重要信息进行脱敏处理

3. **错误信息**
   - 不要在错误消息中暴露敏感信息
   - 实现适当的错误处理和日志记录

## 🔗 相关资源

- [NotifyHub 官方文档](../../README.md)
- [飞书开放平台文档](https://open.feishu.cn/document/)
- [NotifyHub 配置指南](../config/README.md)
- [更多示例](../README.md)

## 🆘 故障排除

### 常见问题

1. **消息发送失败**
   - 检查 Webhook URL 和密钥是否正确
   - 确认机器人已添加到目标群组
   - 查看错误日志获取详细信息

2. **发送速度慢**
   - 检查网络连接
   - 调整队列和工作协程数量
   - 优化批量发送策略

3. **部分消息丢失**
   - 检查重试配置
   - 监控队列状态
   - 查看发送统计数据

### 调试技巧

1. **启用详细日志**
   ```go
   config.WithLogger(logger.NewConsoleLogger(logger.DEBUG))
   ```

2. **测试连接**
   ```go
   health := hub.GetHealth(ctx)
   // 检查平台健康状态
   ```

3. **监控指标**
   ```go
   metrics := hub.GetMetrics()
   // 分析发送成功率和耗时
   ```