# NotifyHub Kafka Pipeline 完整使用指南

🔄 **gin-kafka-producer** + **kafka-consumer-notifier** 异步通知处理完整解决方案

## 📖 概述

本指南展示如何使用 NotifyHub 构建一个完整的异步通知处理管道：

```
HTTP API → Gin Producer → Kafka → Consumer → NotifyHub → Notifications
   ↓            ↓          ↓        ↓          ↓           ↓
REST请求    JSON序列化   消息队列   消费处理   通知发送    平台投递
```

## 🏗️ 架构设计

### 组件职责分离

| 组件 | 职责 | NotifyHub 使用方式 |
|------|------|--------------------|
| **gin-kafka-producer** | HTTP → Kafka | 仅使用 `client.MessageBuilder` 构建消息 |
| **kafka-consumer-notifier** | Kafka → 通知发送 | 使用完整的 `client.Hub.Send()` 发送通知 |

### 设计优势

✅ **职责单一**: 每个服务只负责特定功能  
✅ **异步解耦**: HTTP 请求和通知发送完全分离  
✅ **可扩展性**: 可独立扩展生产者和消费者  
✅ **容错性**: Kafka 提供消息持久化和重试机制  
✅ **可监控性**: 每个环节都有独立的监控指标  

## 🚀 快速开始

### 方式一：Docker Compose（推荐）

#### 1. 启动完整环境

```bash
# 克隆项目（如果还没有）
cd examples

# 启动完整的 Kafka 管道
docker-compose up -d

# 查看服务状态
docker-compose ps
```

#### 2. 测试管道

```bash
# 发送测试通知
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试通知",
    "body": "这是一个完整管道测试",
    "priority": 3,
    "targets": [
      {"type": "email", "value": "admin@example.com"}
    ]
  }'
```

#### 3. 查看处理结果

```bash
# 查看 producer 日志
docker-compose logs gin-kafka-producer

# 查看 consumer 日志
docker-compose logs kafka-consumer-notifier

# 查看 Kafka UI（可选）
open http://localhost:8090
```

### 方式二：本地开发模式

#### 1. 启动 Kafka

```bash
# 仅启动 Kafka 和 Zookeeper
docker-compose up -d zookeeper kafka
```

#### 2. 启动消费者

```bash
cd kafka-consumer-notifier
export NOTIFYHUB_ENABLE_MOCK="true"
go run main.go
```

#### 3. 启动生产者

```bash
# 新终端
cd gin-kafka-producer
go run main.go
```

#### 4. 测试管道

```bash
# 使用示例脚本
cd gin-kafka-producer/examples
chmod +x curl-examples.sh
./curl-examples.sh
```

## 📨 消息流转详解

### 1. HTTP 请求 → Kafka 消息

**gin-kafka-producer** 接收 HTTP 请求并转换为 Kafka 消息：

```json
// HTTP 请求
{
  "title": "系统告警",
  "body": "CPU 使用率过高: {{cpu_usage}}",
  "priority": 4,
  "targets": [
    {"type": "email", "value": "admin@example.com"},
    {"type": "user", "value": "oncall", "platform": "slack"}
  ],
  "variables": {
    "cpu_usage": "94%",
    "server": "web-01"
  }
}
```

**转换为 Kafka 消息**:

```json
{
  "id": "msg_1234567890",
  "created_at": "2024-01-15T10:30:00Z",
  "source": "gin-kafka-producer",
  "version": "1.0.0",
  "message": {
    // NotifyHub 标准消息格式
    "id": "msg_1234567890",
    "title": "系统告警",
    "body": "CPU 使用率过高: {{cpu_usage}}",
    "priority": 4,
    "targets": [...],
    "variables": {...}
  },
  "processing_hints": {
    "priority": 4,
    "retry_policy": {
      "max_retries": 3,
      "initial_interval": "30s",
      "backoff_factor": 2.0
    }
  }
}
```

### 2. Kafka 消息 → 通知发送

**kafka-consumer-notifier** 消费 Kafka 消息并发送通知：

```go
// 1. 消费 Kafka 消息
message, err := reader.ReadMessage(ctx)

// 2. 反序列化
var kafkaMsg KafkaNotificationMessage
json.Unmarshal(message.Value, &kafkaMsg)

// 3. 使用 NotifyHub 发送
results, err := hub.Send(ctx, kafkaMsg.Message)

// 4. 处理结果
for _, result := range results {
    if result.Error != nil {
        log.Printf("发送失败: %v", result.Error)
    } else {
        log.Printf("发送成功: %s", result.Target.Value)
    }
}
```

## ⚙️ 配置指南

### gin-kafka-producer 配置

```bash
# HTTP 服务配置
export HTTP_PORT="8080"

# Kafka 配置
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="notifications"

# 服务配置
export SERVICE_NAME="gin-kafka-producer"
export SERVICE_VERSION="1.0.0"
```

### kafka-consumer-notifier 配置

```bash
# Kafka 配置
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="notifications"
export KAFKA_CONSUMER_GROUP="notifyhub-consumer"
export KAFKA_AUTO_OFFSET_RESET="earliest"

# NotifyHub 配置
export NOTIFYHUB_ENABLE_MOCK="true"      # 测试模式
export NOTIFYHUB_ENABLE_FEISHU="false"   # 飞书通知
export NOTIFYHUB_ENABLE_EMAIL="false"    # 邮件通知

# 消费者配置
export CONSUMER_WORKERS="4"              # 工作线程数
export CONSUMER_BATCH_SIZE="10"          # 批处理大小
export CONSUMER_PROCESSING_TIMEOUT="30s" # 处理超时
export CONSUMER_RETRY_ATTEMPTS="3"       # 重试次数
```

### 真实通知配置

要启用真实的通知发送，需要配置 NotifyHub：

```bash
# 飞书配置
export NOTIFYHUB_FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook"
export NOTIFYHUB_FEISHU_SECRET="your-secret"
export NOTIFYHUB_ENABLE_FEISHU="true"

# 邮件配置
export NOTIFYHUB_SMTP_HOST="smtp.gmail.com"
export NOTIFYHUB_SMTP_PORT="587"
export NOTIFYHUB_SMTP_USERNAME="your-email@gmail.com"
export NOTIFYHUB_SMTP_PASSWORD="your-app-password"
export NOTIFYHUB_SMTP_FROM="your-email@gmail.com"
export NOTIFYHUB_ENABLE_EMAIL="true"

# 禁用测试模式
export NOTIFYHUB_ENABLE_MOCK="false"
```

## 📊 监控和运维

### 健康检查

```bash
# Producer 健康检查
curl http://localhost:8080/health

# Producer 指标
curl http://localhost:8080/metrics

# Producer 状态
curl http://localhost:8080/status
```

### 日志监控

```bash
# 实时查看 Producer 日志
docker-compose logs -f gin-kafka-producer

# 实时查看 Consumer 日志  
docker-compose logs -f kafka-consumer-notifier

# 查看 Kafka 日志
docker-compose logs -f kafka
```

### Kafka 监控

```bash
# 查看 Topic 列表
docker exec notifyhub-kafka kafka-topics --bootstrap-server localhost:9092 --list

# 查看 Topic 详情
docker exec notifyhub-kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic notifications

# 监控消费者组
docker exec notifyhub-kafka kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group notifyhub-consumer

# 手动消费消息（调试用）
docker exec notifyhub-kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic notifications --from-beginning
```

### 性能指标

Consumer 提供实时性能指标：

```
📊 Metrics: processed=150, success=145, errors=5, rate=25.5 msg/min
```

## 🧪 测试场景

### 1. 基础功能测试

```bash
# 简单邮件通知
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试邮件",
    "body": "这是一个测试邮件",
    "targets": [{"type": "email", "value": "test@example.com"}]
  }'
```

### 2. 多平台通知测试

```bash
# 多平台通知
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "title": "多平台测试",
    "body": "这个消息会发送到多个平台",
    "priority": 4,
    "targets": [
      {"type": "email", "value": "admin@example.com"},
      {"type": "user", "value": "oncall", "platform": "slack"},
      {"type": "channel", "value": "alerts", "platform": "slack"}
    ]
  }'
```

### 3. 模板变量测试

```bash
# 带变量的模板消息
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "title": "服务器告警: {{server}}",
    "body": "服务器 {{server}} CPU 使用率达到 {{cpu_usage}}，超过阈值 {{threshold}}",
    "priority": 5,
    "targets": [
      {"type": "email", "value": "sre@company.com"}
    ],
    "variables": {
      "server": "web-01",
      "cpu_usage": "94%",
      "threshold": "90%"
    }
  }'
```

### 4. 批量消息测试

```bash
# 批量发送测试
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/notifications \
    -H "Content-Type: application/json" \
    -d "{
      \"title\": \"批量测试消息 $i\",
      \"body\": \"这是第 $i 个测试消息\",
      \"targets\": [{\"type\": \"email\", \"value\": \"test$i@example.com\"}]
    }"
  sleep 1
done
```

### 5. 错误处理测试

```bash
# 无效请求测试
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "body": "缺少标题字段",
    "targets": []
  }'

# 无效目标类型测试
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试",
    "body": "测试消息",
    "targets": [{"type": "invalid_type", "value": "test"}]
  }'
```

## 🔧 故障排查

### 常见问题

#### 1. Kafka 连接失败

**症状**:
```
❌ Error reading Kafka message: kafka server: Request timed out
```

**解决方案**:
```bash
# 检查 Kafka 是否运行
docker-compose ps kafka

# 检查 Kafka 日志
docker-compose logs kafka

# 重启 Kafka
docker-compose restart kafka
```

#### 2. Consumer 无法处理消息

**症状**:
```
❌ Worker 1 failed to process message: failed to unmarshal Kafka message
```

**解决方案**:
```bash
# 检查消息格式
docker exec notifyhub-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic notifications \
  --from-beginning \
  --max-messages 1

# 检查 Producer 配置
curl http://localhost:8080/status
```

#### 3. NotifyHub 配置问题

**症状**:
```
❌ failed to send notification: no notifiers configured
```

**解决方案**:
```bash
# 确保至少启用一个通知器
export NOTIFYHUB_ENABLE_MOCK="true"

# 或配置真实通知器
export NOTIFYHUB_ENABLE_FEISHU="true"
export NOTIFYHUB_FEISHU_WEBHOOK_URL="your-webhook"
```

### 调试命令

```bash
# 检查所有服务状态
docker-compose ps

# 重启特定服务
docker-compose restart gin-kafka-producer
docker-compose restart kafka-consumer-notifier

# 查看详细日志
docker-compose logs --tail=100 gin-kafka-producer
docker-compose logs --tail=100 kafka-consumer-notifier

# 进入容器调试
docker exec -it notifyhub-gin-producer sh
docker exec -it notifyhub-kafka-consumer sh

# 清理和重启
docker-compose down
docker-compose up -d
```

## 🚀 生产部署

### 1. 环境准备

```bash
# 生产环境配置
export ENV="production"
export KAFKA_BROKERS="kafka-cluster-1:9092,kafka-cluster-2:9092,kafka-cluster-3:9092"
export KAFKA_TOPIC="notifications"

# 性能配置
export CONSUMER_WORKERS="16"
export CONSUMER_BATCH_SIZE="50"
export CONSUMER_PROCESSING_TIMEOUT="30s"

# 安全配置（如需要）
export KAFKA_SSL_ENABLED="true"
export KAFKA_SASL_ENABLED="true"
```

### 2. Docker 部署

```yaml
# docker-compose.prod.yml
version: '3.8'
services:
  gin-kafka-producer:
    image: notifyhub/gin-kafka-producer:v1.0.0
    replicas: 3
    environment:
      - KAFKA_BROKERS=kafka-cluster:9092
    deploy:
      resources:
        limits:
          memory: 256M
        reservations:
          memory: 128M

  kafka-consumer-notifier:
    image: notifyhub/kafka-consumer-notifier:v1.0.0
    replicas: 2
    environment:
      - KAFKA_BROKERS=kafka-cluster:9092
      - CONSUMER_WORKERS=8
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M
```

### 3. Kubernetes 部署

```yaml
# k8s/producer-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gin-kafka-producer
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gin-kafka-producer
  template:
    metadata:
      labels:
        app: gin-kafka-producer
    spec:
      containers:
      - name: gin-kafka-producer
        image: notifyhub/gin-kafka-producer:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: KAFKA_BROKERS
          value: "kafka-service:9092"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
```

## 📈 性能优化

### Producer 优化

```bash
# 高吞吐量配置
export HTTP_PORT="8080"
export KAFKA_BATCH_SIZE="100"
export KAFKA_BATCH_TIMEOUT="10ms"
export KAFKA_COMPRESSION="gzip"
```

### Consumer 优化

```bash
# 高性能消费配置
export CONSUMER_WORKERS="16"           # 更多工作线程
export CONSUMER_BATCH_SIZE="50"        # 更大的批处理
export CONSUMER_PROCESSING_TIMEOUT="10s" # 更短的超时
export KAFKA_FETCH_MIN="1048576"       # 1MB 最小抓取
export KAFKA_FETCH_MAX="52428800"      # 50MB 最大抓取
```

### Kafka 优化

```bash
# Kafka 性能配置
export KAFKA_NUM_PARTITIONS="8"           # 增加分区数
export KAFKA_REPLICATION_FACTOR="3"       # 复制因子
export KAFKA_MIN_INSYNC_REPLICAS="2"      # 最小同步副本
export KAFKA_COMPRESSION_TYPE="lz4"       # 压缩算法
export KAFKA_BATCH_SIZE="65536"           # 批处理大小
export KAFKA_LINGER_MS="5"                # 延迟发送
```

## 🔗 扩展和集成

### 1. 多 Topic 支持

可以为不同类型的通知使用不同的 Topic：

```bash
# 按优先级分 Topic
export KAFKA_TOPIC_HIGH="notifications-high"
export KAFKA_TOPIC_NORMAL="notifications-normal"
export KAFKA_TOPIC_LOW="notifications-low"
```

### 2. 消息路由

基于消息内容进行智能路由：

```go
// 在 Producer 中根据优先级选择 Topic
func selectTopic(priority int) string {
    switch {
    case priority >= 4:
        return "notifications-high"
    case priority >= 2:
        return "notifications-normal"
    default:
        return "notifications-low"
    }
}
```

### 3. 死信队列

处理失败的消息：

```bash
export KAFKA_DLQ_TOPIC="notifications-dlq"
export CONSUMER_MAX_RETRIES="3"
```

### 4. 监控集成

集成 Prometheus + Grafana：

```yaml
# monitoring/docker-compose.yml
services:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
  
  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
```

## 📋 最佳实践

### 1. 消息设计

- ✅ 使用标准的 NotifyHub 消息格式
- ✅ 包含必要的元数据和处理提示
- ✅ 合理设置消息优先级
- ✅ 使用有意义的消息 ID

### 2. 错误处理

- ✅ 实现适当的重试策略
- ✅ 记录详细的错误日志
- ✅ 使用死信队列处理永久失败
- ✅ 监控错误率和成功率

### 3. 性能优化

- ✅ 根据负载调整工作线程数
- ✅ 使用批处理提高吞吐量
- ✅ 合理设置超时时间
- ✅ 监控处理延迟

### 4. 运维监控

- ✅ 实施健康检查
- ✅ 收集关键指标
- ✅ 设置告警规则
- ✅ 定期备份配置

## 🎯 总结

NotifyHub Kafka Pipeline 提供了一个完整的异步通知处理解决方案：

### 核心优势

- **🔄 异步处理**: HTTP 请求立即返回，通知异步处理
- **📈 可扩展性**: Producer 和 Consumer 可独立扩展
- **🛡️ 容错性**: Kafka 提供消息持久化和重试机制
- **🔧 可配置性**: 丰富的配置选项适应不同场景
- **📊 可观测性**: 完整的监控和日志记录

### 适用场景

- ✅ 高并发通知系统
- ✅ 微服务架构中的通知中心
- ✅ 需要解耦的异步处理
- ✅ 多平台通知分发
- ✅ 大规模消息处理

通过这个完整的 Kafka Pipeline，你可以构建一个高性能、高可用的通知处理系统！🚀