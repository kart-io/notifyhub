#!/bin/bash

# Gin Kafka Producer - cURL Examples
# This script demonstrates various API calls to the gin-kafka-producer service

BASE_URL="http://localhost:8080"

echo "🚀 Gin Kafka Producer - cURL Examples"
echo "======================================"

# Check if service is running
echo ""
echo "1. 💚 Health Check"
echo "-------------------"
curl -s "$BASE_URL/health" | jq .

echo ""
echo "2. ℹ️  Service Information"
echo "-------------------------"
curl -s "$BASE_URL/" | jq .

echo ""
echo "3. 📧 Simple Email Notification"
echo "-------------------------------"
curl -X POST "$BASE_URL/api/v1/notifications" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Simple Email Alert",
    "body": "This is a simple email notification test",
    "priority": 3,
    "targets": [
      {"type": "email", "value": "admin@example.com"}
    ]
  }' | jq .

echo ""
echo "4. 🚨 High Priority System Alert"
echo "--------------------------------"
curl -X POST "$BASE_URL/api/v1/notifications" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "CRITICAL: System Alert",
    "body": "High CPU usage detected on server {{server}}. Current usage: {{cpu_usage}}",
    "priority": 5,
    "targets": [
      {"type": "email", "value": "sre@company.com"},
      {"type": "user", "value": "oncall", "platform": "slack"},
      {"type": "channel", "value": "alerts", "platform": "slack"}
    ],
    "variables": {
      "cpu_usage": "94%",
      "server": "web-01",
      "threshold": "90%"
    },
    "metadata": {
      "alert_type": "system",
      "severity": "critical",
      "source": "monitoring",
      "environment": "production"
    }
  }' | jq .

echo ""
echo "5. 📱 Multi-Platform Notification"
echo "---------------------------------"
curl -X POST "$BASE_URL/api/v1/notifications" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Deployment Complete",
    "body": "Version {{version}} has been successfully deployed to {{environment}}",
    "priority": 2,
    "targets": [
      {"type": "email", "value": "devops@company.com"},
      {"type": "email", "value": "product@company.com"},
      {"type": "channel", "value": "deployments", "platform": "slack"},
      {"type": "group", "value": "dev-team", "platform": "feishu"},
      {"type": "user", "value": "tech-lead", "platform": "slack"}
    ],
    "variables": {
      "version": "v2.1.0",
      "environment": "production",
      "deployer": "jenkins",
      "duration": "3m 45s"
    },
    "template": "deployment_success",
    "metadata": {
      "deployment_id": "dep_12345",
      "commit_sha": "abc123def456",
      "branch": "main"
    }
  }' | jq .

echo ""
echo "6. 🔧 Notification with Kafka Options"
echo "-------------------------------------"
curl -X POST "$BASE_URL/api/v1/notifications" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Custom Kafka Routing",
    "body": "This message uses custom Kafka options for routing",
    "priority": 3,
    "targets": [
      {"type": "email", "value": "test@example.com"}
    ],
    "kafka_options": {
      "key": "user_notifications",
      "partition": 0,
      "headers": {
        "source": "api",
        "routing_key": "email",
        "priority": "normal"
      }
    }
  }' | jq .

echo ""
echo "7. 📊 Service Metrics"
echo "--------------------"
curl -s "$BASE_URL/metrics" | jq .

echo ""
echo "8. 🔍 Service Status"
echo "-------------------"
curl -s "$BASE_URL/status" | jq .

echo ""
echo "9. ❌ Error Example - Invalid Request"
echo "------------------------------------"
curl -X POST "$BASE_URL/api/v1/notifications" \
  -H "Content-Type: application/json" \
  -d '{
    "body": "Missing title field",
    "targets": []
  }' | jq .

echo ""
echo "10. ❌ Error Example - Invalid Target Type"
echo "------------------------------------------"
curl -X POST "$BASE_URL/api/v1/notifications" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test",
    "body": "Test message",
    "targets": [
      {"type": "invalid_type", "value": "test"}
    ]
  }' | jq .

echo ""
echo "✅ All examples completed!"
echo ""
echo "💡 Tips:"
echo "- Make sure the service is running: ./gin-kafka-producer or go run main.go"
echo "- Make sure Kafka is running: docker run -d --name kafka-test -p 9092:9092 confluentinc/cp-kafka:latest"
echo "- Install jq for JSON formatting: brew install jq (macOS) or apt-get install jq (Ubuntu)"
echo "- Check service logs for Kafka message details"
echo ""
echo "🔗 Next Steps:"
echo "- Run the kafka-consumer-notifier to process these messages"
echo "- Monitor Kafka topic: kafka-console-consumer --bootstrap-server localhost:9092 --topic notifications --from-beginning"