# NotifyHub HTTP Service Examples - 从"可用"到"易用"的进化

This directory contains comprehensive examples showing the evolution of NotifyHub from a "usable" to a "user-friendly" library through systematic optimizations.

## 🚀 快速开始 (Ultra-Optimized Version)

```bash
# 运行超级优化版服务器 (推荐)
go run cmd/ultra_optimized_server.go

# 服务器启动时会自动配置所有路由:
# POST /api/v1/send      - 发送通知
# POST /api/v1/batch     - 批量通知
# POST /api/v1/text      - 快速文本消息
# POST /api/v1/alert     - 紧急警报
# POST /api/v1/template  - 模板消息
# GET  /api/v1/health    - 健康检查
# GET  /api/v1/metrics   - 服务指标
```

## 📊 优化进化历程

### 服务器实现版本

| 版本 | 文件 | 代码行数 | 主要特性 | 适用场景 |
|---------|------|---------------|--------------|----------|
| **原版** | `cmd/server.go` | ~200 行 | 基础实现 | 学习/参考 |
| **优化版** | `cmd/optimized_server.go` | ~150 行 | 增强模式 | 生产就绪 |
| **超级优化版** | `cmd/ultra_optimized_server.go` | **~80 行** | 全自动化 | **推荐使用** |

### 代码减少成效

```go
// ❌ 优化前：复杂初始化 (15+ 行)
cfg := config.New()
queueConfig := &config.QueueConfig{
    Type:        "memory",
    BufferSize:  1000,
    Workers:     2,
    RetryPolicy: queue.DefaultRetryPolicy(),
}
hub := &client.Hub{
    config:    cfg,
    notifiers: make(map[string]notifiers.Notifier),
    // ... 10+ 更多行
}
if err := hub.Start(ctx); err != nil {
    return err
}

// ✅ 优化后：一行初始化
hub, err := client.NewWithDefaultsAndStart(ctx)
```

## 🌟 核心特性

- **🚀 极简API**：从200行代码减少到80行，代码减少60%
- **📦 一键部署**：一行代码创建完整HTTP服务
- **🛡️ 自动安全**：内置安全中间件、CORS、限流等
- **🎯 智能路由**：自动检测目标类型和平台
- **📊 完整监控**：健康检查、指标收集、链路追踪
- **🧪 测试友好**：一行代码创建测试环境

## 📁 项目结构

```
examples/http-service/
├── cmd/
│   └── server.go              # 主服务器入口
├── internal/
│   ├── handlers/              # HTTP 处理器
│   │   └── handlers.go
│   ├── middleware/            # 中间件
│   │   └── middleware.go
│   └── models/                # 数据模型
│       └── requests.go
├── test/
│   ├── unit/                  # 单元测试
│   │   ├── handlers_test.go
│   │   └── middleware_test.go
│   ├── e2e/                   # 端到端测试
│   │   └── server_test.go
│   └── performance/           # 性能测试
│       └── load_test.go
├── config/
│   └── config.yaml            # 配置文件
├── Dockerfile                 # Docker 镜像
├── docker-compose.yml         # Docker Compose
├── Makefile                   # 构建脚本
├── .env.example              # 环境变量示例
└── README.md                 # 文档
```

## 🚀 快速开始

### 1. 环境准备

```bash
# 克隆项目（如果需要）
cd examples/http-service

# 复制环境变量配置
cp .env.example .env

# 编辑配置
vim .env
```

### 2. 安装依赖

```bash
# 安装 Go 依赖
make deps

# 安装开发工具（可选）
make install-tools
```

### 3. 配置环境变量

编辑 `.env` 文件，设置必要的配置：

```bash
# 基本配置
API_KEY=your-secret-api-key
PORT=8080

# 飞书配置
NOTIFYHUB_FEISHU_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/xxx
NOTIFYHUB_FEISHU_SECRET=your-secret

# 邮件配置
NOTIFYHUB_SMTP_HOST=smtp.gmail.com
NOTIFYHUB_SMTP_PORT=587
NOTIFYHUB_SMTP_USERNAME=your-email@gmail.com
NOTIFYHUB_SMTP_PASSWORD=your-app-password
NOTIFYHUB_SMTP_FROM=your-email@gmail.com
```

### 4. 运行服务

```bash
# 开发模式运行
make run

# 或者使用 Docker
make docker-build
make docker-run

# 或者使用 Docker Compose
docker-compose up -d
```

### 5. 验证服务

```bash
# 检查健康状态
curl http://localhost:8080/health

# 查看服务指标
curl http://localhost:8080/metrics

# 发送测试通知
make example-notification
```

## 📚 API 文档

### 基础信息

- **Base URL**: `http://localhost:8080`
- **认证**: Bearer token (如果配置了 `API_KEY`)
- **Content-Type**: `application/json`

### 端点列表

#### 1. 健康检查

```bash
GET /health
```

响应示例：
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": {
    "hub": "healthy",
    "queue": "healthy"
  },
  "uptime": "2h30m45s"
}
```

#### 2. 服务指标

```bash
GET /metrics
```

响应示例：
```json
{
  "total_sent": 1250,
  "success_rate": 0.98,
  "avg_duration": "150ms",
  "sends_by_platform": {
    "email": 800,
    "feishu": 450
  },
  "last_updated": "2024-01-15T10:30:00Z"
}
```

#### 3. 发送通知

```bash
POST /api/v1/notifications
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "title": "系统告警",
  "body": "服务器 CPU 使用率过高",
  "format": "markdown",
  "targets": [
    {
      "type": "email",
      "value": "ops@company.com"
    },
    {
      "type": "group",
      "value": "ops-alerts",
      "platform": "feishu"
    }
  ],
  "priority": 4,
  "variables": {
    "server": "web-01",
    "cpu_usage": "95%"
  },
  "metadata": {
    "environment": "production",
    "service": "web-server"
  }
}
```

#### 4. 批量发送

```bash
POST /api/v1/notifications/bulk
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "notifications": [
    {
      "title": "通知 1",
      "body": "内容 1",
      "targets": [{"type": "email", "value": "user1@example.com"}]
    },
    {
      "title": "通知 2",
      "body": "内容 2",
      "targets": [{"type": "email", "value": "user2@example.com"}]
    }
  ]
}
```

#### 5. 快速文本通知

```bash
GET /api/v1/notifications/text?title=测试&body=Hello&target=test@example.com
Authorization: Bearer your-api-key
```

## 🧪 测试

### 单元测试

```bash
# 运行所有单元测试
make unit-test

# 运行单元测试并生成覆盖率报告
make unit-test-coverage

# 查看覆盖率报告
open coverage.html
```

### E2E 测试

```bash
# 先启动服务（端口 8081）
PORT=8081 API_KEY=test-api-key-12345 make run

# 在另一个终端运行 E2E 测试
make e2e-test
```

### 性能测试

```bash
# 确保服务运行在端口 8080
make run

# 在另一个终端运行性能测试
make performance-test

# 运行负载测试
make load-test

# 运行压力测试
make stress-test

# 运行基准测试
make benchmark
```

### 测试覆盖范围

- **单元测试**：处理器逻辑、中间件功能、数据验证
- **E2E 测试**：完整的请求响应流程、认证、错误处理
- **性能测试**：并发负载、延迟分析、吞吐量测试
- **压力测试**：渐进式负载增加、资源使用监控

## 🐳 Docker 部署

### 单独部署

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run

# 或者直接使用 Docker 命令
docker run -p 8080:8080 --env-file .env notifyhub-http-service:latest
```

### 使用 Docker Compose

```bash
# 启动所有服务
docker-compose up -d

# 启动包含 Redis 的完整环境
docker-compose --profile redis up -d

# 启动包含监控的完整环境
docker-compose --profile monitoring up -d

# 启动包含链路追踪的环境
docker-compose --profile tracing up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f notifyhub-http-service
```

### 生产部署检查清单

- [ ] 设置安全的 `API_KEY`
- [ ] 配置 TLS/HTTPS
- [ ] 设置适当的资源限制
- [ ] 配置日志收集
- [ ] 设置监控和告警
- [ ] 配置负载均衡
- [ ] 设置数据备份策略
- [ ] 配置安全扫描

## 📊 监控与可观测性

### 内置监控

- **健康检查**: `/health` 端点
- **指标暴露**: `/metrics` 端点
- **结构化日志**: JSON 格式日志
- **请求追踪**: 每个请求的唯一 ID

### 集成监控工具

使用 Docker Compose 可以快速启动监控栈：

```bash
# 启动 Prometheus + Grafana
docker-compose --profile monitoring up -d

# 访问 Grafana: http://localhost:3000 (admin/admin)
# 访问 Prometheus: http://localhost:9090
```

### 链路追踪

```bash
# 启动 Jaeger
docker-compose --profile tracing up -d

# 访问 Jaeger UI: http://localhost:16686
```

## ⚡ 性能优化

### 当前性能指标

基于内置性能测试的基准：

- **单个通知延迟**: < 100ms (p95)
- **批量通知延迟**: < 500ms (p95)
- **吞吐量**: > 100 req/s (单个通知)
- **并发支持**: 100+ 并发连接
- **内存使用**: < 50MB (基础负载)

### 优化建议

1. **连接池配置**: 调整 HTTP 客户端连接池大小
2. **队列配置**: 根据负载调整 worker 数量和缓冲区大小
3. **缓存策略**: 实现请求去重和结果缓存
4. **负载均衡**: 使用多实例水平扩展
5. **数据库优化**: 如果使用持久化存储，优化查询

## 🔧 开发指南

### 添加新的中间件

```go
// internal/middleware/custom.go
func CustomMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 自定义逻辑
        next.ServeHTTP(w, r)
    })
}
```

### 添加新的端点

```go
// internal/handlers/handlers.go
func (h *NotificationHandler) CustomEndpoint(w http.ResponseWriter, r *http.Request) {
    // 处理逻辑
}

// cmd/server.go - 在 setupRoutes 中添加
mux.Handle("/api/v1/custom", middlewareChain(http.HandlerFunc(handler.CustomEndpoint)))
```

### 自定义配置

编辑 `config/config.yaml` 或使用环境变量：

```yaml
# config/config.yaml
custom:
  feature_enabled: true
  timeout: 30s
```

## 🐛 故障排除

### 常见问题

1. **服务启动失败**
   ```bash
   # 检查端口是否被占用
   lsof -i :8080

   # 检查环境变量
   env | grep NOTIFYHUB
   ```

2. **通知发送失败**
   ```bash
   # 检查日志
   docker-compose logs notifyhub-http-service

   # 测试网络连接
   curl -v https://open.feishu.cn
   ```

3. **性能问题**
   ```bash
   # 检查系统资源
   docker stats

   # 查看服务指标
   curl http://localhost:8080/metrics
   ```

### 调试模式

```bash
# 启用详细日志
LOG_LEVEL=debug make run

# 或者在 Docker 中
docker run -e LOG_LEVEL=debug -p 8080:8080 notifyhub-http-service
```

## 📈 扩展性考虑

### 水平扩展

- 无状态设计，支持多实例部署
- 使用外部队列（Redis）实现实例间通信
- 负载均衡器分发请求

### 垂直扩展

- 调整 `NOTIFYHUB_QUEUE_WORKERS` 增加处理能力
- 增加 `NOTIFYHUB_QUEUE_BUFFER_SIZE` 提高缓冲能力
- 优化 `RATE_LIMIT_PER_MINUTE` 平衡性能和保护

### 架构演进

1. **微服务拆分**: 将通知发送拆分为独立服务
2. **消息队列**: 使用 Kafka/RabbitMQ 等企业级消息队列
3. **配置中心**: 使用 Consul/etcd 等配置中心
4. **服务网格**: 使用 Istio 等服务网格管理通信

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

### 代码规范

```bash
# 运行所有质量检查
make quality

# 包括：格式化、vet、lint、测试
```

## 📄 许可证

本项目基于 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

**🎯 这是一个生产就绪的 NotifyHub HTTP 服务示例，展示了现代 Go 服务开发的最佳实践。**