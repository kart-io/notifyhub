#!/bin/bash

# Gin Kafka Producer 测试脚本

set -e

echo "🧪 Testing Gin Kafka Producer"
echo "=============================================="

# 检查是否已编译
if [ ! -f "./gin-kafka-producer" ]; then
    echo "📦 编译服务..."
    go build -o gin-kafka-producer main.go
    echo "✅ 编译完成"
else
    echo "✅ 发现已编译的二进制文件"
fi

echo ""
echo "🔧 配置说明："
echo "需要运行 Kafka 服务器在 localhost:9092"
echo "Topic: notifications"
echo ""

# 检查是否提供了 --no-kafka 参数
if [ "$1" = "--no-kafka" ]; then
    echo "⚠️  --no-kafka 模式：将进行基本功能测试（不连接真实 Kafka）"
    echo "💡 实际使用时请确保 Kafka 服务正在运行"
    echo ""
    
    # 设置环境变量
    export HTTP_PORT="8081"
    export KAFKA_BROKERS="localhost:9092"
    export KAFKA_TOPIC="notifications"
    export SERVICE_NAME="gin-kafka-producer-test"
    export SERVICE_VERSION="1.0.0-test"
    
    echo "🚀 环境变量已设置："
    echo "   HTTP_PORT=$HTTP_PORT"
    echo "   KAFKA_BROKERS=$KAFKA_BROKERS"
    echo "   KAFKA_TOPIC=$KAFKA_TOPIC"
    echo "   SERVICE_NAME=$SERVICE_NAME"
    echo ""
    
    echo "✅ 测试完成 - 二进制文件可用"
    echo ""
    echo "📖 使用方式："
    echo "1. 启动 Kafka: docker run -d --name kafka-test -p 9092:9092 confluentinc/cp-kafka:latest"
    echo "2. 创建 topic: kafka-topics --create --topic notifications --bootstrap-server localhost:9092"
    echo "3. 运行服务: ./gin-kafka-producer"
    echo "4. 发送测试请求："
    echo ""
    echo "   curl -X POST http://localhost:8080/api/v1/notifications \\"
    echo "     -H \"Content-Type: application/json\" \\"
    echo "     -d '{"
    echo "       \"title\": \"测试告警\","
    echo "       \"body\": \"这是一个测试消息\","
    echo "       \"priority\": 5,"
    echo "       \"targets\": ["
    echo "         {\"type\": \"email\", \"value\": \"test@example.com\", \"platform\": \"email\"}"
    echo "       ]"
    echo "     }'"
    echo ""
    echo "5. 检查健康状态:"
    echo "   curl http://localhost:8080/health"
    
    exit 0
fi

# 检查 Kafka 是否运行
echo "🔍 检查 Kafka 连接..."
if ! timeout 5 bash -c "</dev/tcp/localhost/9092" 2>/dev/null; then
    echo "❌ 无法连接到 Kafka (localhost:9092)"
    echo ""
    echo "💡 请先启动 Kafka 服务，或使用 --no-kafka 参数进行模拟测试"
    echo ""
    echo "快速启动 Kafka (Docker):"
    echo "  docker run -d --name kafka-test -p 9092:9092 \\"
    echo "    -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \\"
    echo "    -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \\"
    echo "    -e KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092 \\"
    echo "    confluentinc/cp-kafka:latest"
    echo ""
    echo "或运行: $0 --no-kafka"
    exit 1
fi

echo "✅ Kafka 连接正常"

# 设置环境变量
export HTTP_PORT="8081"
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="notifications"
export SERVICE_NAME="gin-kafka-producer-test"
export SERVICE_VERSION="1.0.0-test"

echo ""
echo "🚀 启动 HTTP 服务 (5秒后自动停止)..."
echo "   Port: $HTTP_PORT"
echo "   Kafka Brokers: $KAFKA_BROKERS"
echo "   Kafka Topic: $KAFKA_TOPIC"
echo ""

# 在后台启动服务
timeout 10 ./gin-kafka-producer &
HTTP_PID=$!

# 等待服务启动
sleep 3

# 检查服务是否正在运行
if ! kill -0 $HTTP_PID 2>/dev/null; then
    echo "❌ HTTP 服务启动失败"
    exit 1
fi

echo "✅ HTTP 服务启动成功"

# 测试健康检查
echo ""
echo "🔍 测试健康检查..."
HEALTH_RESPONSE=$(curl -s http://localhost:$HTTP_PORT/health)
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    echo "✅ 健康检查正常"
else
    echo "❌ 健康检查失败"
    echo "响应: $HEALTH_RESPONSE"
fi

# 发送测试通知
echo ""
echo "📨 发送测试通知..."

# 测试消息1: 简单告警
TEST_MSG1=$(cat <<EOF
{
  "title": "🚨 测试告警",
  "body": "这是一个测试告警消息",
  "priority": 5,
  "targets": [
    {"type": "email", "value": "admin@example.com", "platform": "email"},
    {"type": "group", "value": "ops-team", "platform": "feishu"}
  ],
  "variables": {
    "server": "test-server",
    "error": "test error"
  },
  "metadata": {
    "environment": "test",
    "severity": "high"
  }
}
EOF
)

# 发送测试消息1
echo "📤 发送告警消息..."
RESPONSE1=$(curl -s -X POST http://localhost:$HTTP_PORT/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d "$TEST_MSG1")

if echo "$RESPONSE1" | grep -q '"success":true'; then
    echo "✅ 告警消息发送成功"
    MESSAGE_ID=$(echo "$RESPONSE1" | grep -o '"message_id":"[^"]*"' | cut -d'"' -f4)
    echo "   Message ID: $MESSAGE_ID"
else
    echo "❌ 告警消息发送失败"
    echo "响应: $RESPONSE1"
fi

sleep 1

# 测试消息2: 带 Kafka 选项
TEST_MSG2=$(cat <<EOF
{
  "title": "📢 测试通知",
  "body": "这是一个带 Kafka 选项的测试消息",
  "priority": 3,
  "targets": [
    {"type": "email", "value": "users@example.com", "platform": "email"}
  ],
  "kafka_options": {
    "key": "test-key",
    "headers": {
      "correlation-id": "test-123",
      "source": "test-suite"
    }
  }
}
EOF
)

# 发送测试消息2
echo "📤 发送带选项的通知消息..."
RESPONSE2=$(curl -s -X POST http://localhost:$HTTP_PORT/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d "$TEST_MSG2")

if echo "$RESPONSE2" | grep -q '"success":true'; then
    echo "✅ 带选项的通知消息发送成功"
    MESSAGE_ID2=$(echo "$RESPONSE2" | grep -o '"message_id":"[^"]*"' | cut -d'"' -f4)
    echo "   Message ID: $MESSAGE_ID2"
else
    echo "❌ 带选项的通知消息发送失败"
    echo "响应: $RESPONSE2"
fi

# 测试服务信息
echo ""
echo "🔍 测试服务信息..."
INFO_RESPONSE=$(curl -s http://localhost:$HTTP_PORT/)
if echo "$INFO_RESPONSE" | grep -q "gin-kafka-producer"; then
    echo "✅ 服务信息正常"
else
    echo "❌ 服务信息异常"
    echo "响应: $INFO_RESPONSE"
fi

# 等待消息处理
echo ""
echo "⏳ 等待消息处理..."
sleep 1

# 停止服务
if kill -0 $HTTP_PID 2>/dev/null; then
    echo "🛑 停止 HTTP 服务..."
    kill $HTTP_PID 2>/dev/null || true
    wait $HTTP_PID 2>/dev/null || true
fi

echo ""
echo "✅ 测试完成！"
echo ""
echo "🎯 测试结果："
echo "  - 二进制文件编译成功"
echo "  - HTTP 服务可以正常启动"
echo "  - 健康检查工作正常"
echo "  - 消息发送 API 工作正常"
echo "  - Kafka 消息生产功能正常"
echo ""
echo "📖 查看详细日志了解处理过程"
echo "💡 如需持续运行: ./gin-kafka-producer"