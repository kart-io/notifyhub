# Gin Kafka Producer

HTTP 到 Kafka 的通知生产者服务，实现完整的消息构建和发送功能。

## ✨ 核心特性

1. **完整 API** - 提供完整的 HTTP API 用于发送通知到 Kafka
2. **消息构建** - 内置 `MessageBuilder` 支持灵活的消息构建
3. **Kafka 集成** - 高性能的 Kafka 生产者配置
4. **中间件支持** - 包含 CORS、日志、请求ID等中间件

## 🚀 快速运行

### 1. 编译服务

```bash
go build -o gin-kafka-producer main.go
```

### 2. 设置环境变量（可选）

```bash
# HTTP 配置
export HTTP_PORT="8080"

# Kafka 配置
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="notifications"

# 服务配置
export SERVICE_NAME="gin-kafka-producer"
export SERVICE_VERSION="1.0.0"
```

### 3. 启动 Kafka（测试环境）

```bash
# 使用 Docker Compose 启动 Kafka
docker run -d --name kafka-test \
  -p 9092:9092 \
  -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092 \
  confluentinc/cp-kafka:latest

# 创建 topic
kafka-topics --create --topic notifications --bootstrap-server localhost:9092
```

### 4. 运行服务

```bash
./gin-kafka-producer
```

## 📡 API 使用

### 发送通知

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "title": "🚨 系统告警",
    "body": "数据库连接异常，请立即处理",
    "priority": 5,
    "targets": [
      {
        "type": "email",
        "value": "admin@example.com",
        "platform": "email"
      },
      {
        "type": "group", 
        "value": "ops-team",
        "platform": "feishu"
      }
    ],
    "variables": {
      "server": "db-01",
      "error": "connection timeout",
      "timestamp": "2024-01-01T10:00:00Z"
    },
    "metadata": {
      "environment": "production",
      "severity": "high"
    }
  }'
```

### 带 Kafka 选项

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试通知",
    "body": "这是一个测试消息",
    "priority": 3,
    "targets": [
      {"type": "email", "value": "test@example.com"}
    ],
    "kafka_options": {
      "key": "user-123",
      "partition": 0,
      "headers": {
        "correlation-id": "req-456",
        "source-system": "api-gateway"
      }
    }
  }'
```

## 📊 API 端点

| 端点 | 方法 | 描述 | 示例 |
|------|------|------|------|
| `/` | GET | 服务信息 | `curl http://localhost:8080/` |
| `/health` | GET | 健康检查 | `curl http://localhost:8080/health` |
| `/api/v1/notifications` | POST | 发送通知 | 见上方示例 |
| `/api/v1/send` | POST | 发送通知（别名） | 同上 |
| `/metrics` | GET | 基础指标 | `curl http://localhost:8080/metrics` |
| `/status` | GET | 详细状态 | `curl http://localhost:8080/status` |

## 📈 预期响应

### 成功响应

```json
{
  "success": true,
  "message": "Notification queued successfully",
  "data": {
    "message_id": "msg_1704096000123456",
    "kafka_offset": -1,
    "kafka_partition": -1,
    "target_count": 2
  },
  "request_id": "req_1704096000123456",
  "timestamp": "2024-01-01T10:00:00Z"
}
```

### 错误响应

```json
{
  "success": false,
  "message": "Invalid request format",
  "request_id": "req_1704096000123456",
  "timestamp": "2024-01-01T10:00:00Z",
  "errors": [
    "Key: 'NotificationRequest.Title' Error:Field validation for 'Title' failed on the 'required' tag"
  ]
}
```

## 🎯 发送到 Kafka 的消息格式

服务会将HTTP请求转换为以下格式发送到 Kafka:

```json
{
  "id": "msg_1704096000123456",
  "created_at": "2024-01-01T10:00:00Z",
  "source": "gin-kafka-producer-fixed",
  "version": "1.0.0-fixed",
  "message": {
    "id": "msg_1704096000123456",
    "title": "🚨 系统告警",
    "body": "数据库连接异常，请立即处理",
    "format": "",
    "targets": [
      {
        "type": "email",
        "value": "admin@example.com",
        "platform": "email"
      }
    ],
    "variables": {
      "server": "db-01",
      "error": "connection timeout"
    },
    "metadata": {
      "environment": "production",
      "severity": "high"
    },
    "priority": 5,
    "created_at": "2024-01-01T10:00:00Z"
  },
  "processing_hints": {
    "priority": 5,
    "retry_policy": {
      "max_retries": 3,
      "initial_interval": "30s",
      "backoff_factor": 2.0
    },
    "tags": ["http-api", "gin-producer-fixed"]
  }
}
```

## 🔍 监控和调试

### 服务日志

服务启动后会显示详细的配置信息：

```
🚀 Gin Kafka Producer Fixed - NotifyHub HTTP to Kafka Bridge
======================================================================
🔧 Fixed version: Bypasses client package compilation issues
📋 Configuration loaded:
  HTTP Port: 8080
  Kafka Brokers: [localhost:9092]
  Kafka Topic: notifications
  Service: gin-kafka-producer-fixed v1.0.0-fixed

📡 Initializing Kafka producer...
🌐 Initializing HTTP service...

✅ Service ready! Endpoints:
  📍 Send Notification: POST http://localhost:8080/api/v1/notifications
  💚 Health Check: GET http://localhost:8080/health
  ℹ️  Service Info: GET http://localhost:8080/
  📊 Metrics: GET http://localhost:8080/metrics

🎯 Kafka Target: localhost:9092 -> notifications
🔥 Service starting...
```

### HTTP 请求日志

每个HTTP请求都会被记录：

```
[2024-01-01 10:00:00] POST /api/v1/notifications 202 1.234ms 127.0.0.1
```

## 🛠️ 修复说明

### 主要变更

1. **FixedMessageBuilder**: 替代 `client.MessageBuilder`
   ```go
   // 原版本（有编译问题）
   builder := client.NewMessage().Title("test").Body("content")
   
   // 修复版本
   builder := NewFixedMessage().Title("test").Body("content")
   ```

2. **直接依赖**: 只依赖 `notifiers` 包，避免 client 包的编译问题

3. **功能保持**: API 接口和 Kafka 消息格式完全一致

### 兼容性

- ✅ HTTP API 完全兼容
- ✅ Kafka 消息格式完全兼容  
- ✅ 配置方式完全兼容
- ✅ 可与原版 consumer 配合使用

## 📝 使用建议

1. **开发环境**: 直接使用此修复版本进行开发和测试
2. **生产环境**: 当 client 包编译问题修复后，可无缝迁移回原版本
3. **集成测试**: 可与 kafka-consumer-notifier-fixed 配合进行完整测试

## 🔗 相关项目

- [kafka-consumer-notifier-fixed](../kafka-consumer-notifier/) - 消费者修复版本
- [notifyhub](../../) - 主项目