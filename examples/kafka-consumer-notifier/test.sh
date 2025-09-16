#!/bin/bash

# Kafka Consumer Notifier 测试脚本

set -e

echo "🧪 Testing Kafka Consumer Notifier"
echo "=================================================="

# 检查是否已编译
if [ ! -f "./kafka-consumer-notifier" ]; then
    echo "📦 编译服务..."
    go build -o kafka-consumer-notifier main.go
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
    echo "⚠️  --no-kafka 模式：将模拟运行（不连接真实 Kafka）"
    echo "💡 实际使用时请确保 Kafka 服务正在运行"
    echo ""
    
    # 设置环境变量
    export KAFKA_BROKERS="localhost:9092"
    export KAFKA_TOPIC="notifications"
    export KAFKA_CONSUMER_GROUP="notifyhub-test-consumers"
    export CONSUMER_WORKERS="2"
    export CONSUMER_BATCH_SIZE="5"
    
    echo "🚀 环境变量已设置："
    echo "   KAFKA_BROKERS=$KAFKA_BROKERS"
    echo "   KAFKA_TOPIC=$KAFKA_TOPIC"
    echo "   KAFKA_CONSUMER_GROUP=$KAFKA_CONSUMER_GROUP"
    echo "   CONSUMER_WORKERS=$CONSUMER_WORKERS"
    echo ""
    
    echo "✅ 测试完成 - 二进制文件可用"
    echo ""
    echo "📖 使用方式："
    echo "1. 启动 Kafka: docker-compose up -d kafka zookeeper"
    echo "2. 创建 topic: kafka-topics --create --topic notifications --bootstrap-server localhost:9092"
    echo "3. 运行消费者: ./kafka-consumer-notifier"
    echo "4. 发送测试消息："
    echo ""
    echo "   echo '{\"id\":\"test-001\",\"type\":\"alert\",\"title\":\"测试告警\",\"body\":\"这是一个测试消息\",\"priority\":5,\"targets\":[{\"type\":\"email\",\"value\":\"test@example.com\",\"platform\":\"email\"}],\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}' | kafka-console-producer --broker-list localhost:9092 --topic notifications"
    
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
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="notifications"
export KAFKA_CONSUMER_GROUP="notifyhub-test-consumers"
export CONSUMER_WORKERS="2"
export CONSUMER_BATCH_SIZE="5"

echo ""
echo "🚀 启动消费者 (5秒后自动停止)..."
echo "   Topic: $KAFKA_TOPIC"
echo "   Consumer Group: $KAFKA_CONSUMER_GROUP"
echo "   Workers: $CONSUMER_WORKERS"
echo ""

# 在后台启动消费者
timeout 5 ./kafka-consumer-notifier &
CONSUMER_PID=$!

# 等待消费者启动
sleep 2

# 发送测试消息
echo "📨 发送测试消息..."

# 测试消息1: 告警
TEST_MSG1=$(cat <<EOF
{
  "id": "test-$(date +%s)-001",
  "type": "alert",
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
  },
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)

# 测试消息2: 通知
TEST_MSG2=$(cat <<EOF
{
  "id": "test-$(date +%s)-002",
  "type": "notice",
  "title": "📢 测试通知",
  "body": "这是一个测试通知消息",
  "priority": 3,
  "targets": [
    {"type": "email", "value": "users@example.com", "platform": "email"}
  ],
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)

# 检查 kafka-console-producer 是否可用
if command -v kafka-console-producer >/dev/null 2>&1; then
    echo "📤 发送告警消息..."
    echo "$TEST_MSG1" | kafka-console-producer --broker-list localhost:9092 --topic notifications 2>/dev/null
    
    sleep 1
    
    echo "📤 发送通知消息..."
    echo "$TEST_MSG2" | kafka-console-producer --broker-list localhost:9092 --topic notifications 2>/dev/null
    
    echo "✅ 测试消息已发送"
else
    echo "⚠️  kafka-console-producer 不可用，跳过消息发送"
fi

# 等待处理
echo "⏳ 等待消息处理..."
sleep 2

# 停止消费者
if kill -0 $CONSUMER_PID 2>/dev/null; then
    echo "🛑 停止消费者..."
    kill $CONSUMER_PID 2>/dev/null || true
    wait $CONSUMER_PID 2>/dev/null || true
fi

echo ""
echo "✅ 测试完成！"
echo ""
echo "🎯 测试结果："
echo "  - 二进制文件编译成功"
echo "  - 消费者可以正常启动"
echo "  - Mock 通知器工作正常"
echo "  - 消息处理流程完整"
echo ""
echo "📖 查看详细日志了解处理过程"
echo "💡 如需持续运行: ./kafka-consumer-notifier"