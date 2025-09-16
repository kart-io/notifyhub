#!/bin/bash

# NotifyHub HTTP Service API 测试脚本
# 使用方法: ./test_api.sh [base_url]

set -e

# 默认服务地址
BASE_URL=${1:-"http://localhost:8080"}
API_BASE="$BASE_URL/api/v1"

echo "🧪 测试 NotifyHub HTTP Service API"
echo "📡 服务地址: $BASE_URL"
echo "=================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -e "${BLUE}测试: $description${NC}"
    echo "请求: $method $endpoint"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$endpoint")
    fi
    
    # 分离响应体和状态码
    body=$(echo "$response" | sed '$d')
    status_code=$(echo "$response" | tail -n 1)
    
    if [[ $status_code -ge 200 && $status_code -lt 300 ]]; then
        echo -e "${GREEN}✅ 成功 (HTTP $status_code)${NC}"
        echo "响应: $(echo "$body" | jq '.' 2>/dev/null || echo "$body")"
    else
        echo -e "${RED}❌ 失败 (HTTP $status_code)${NC}"
        echo "响应: $body"
    fi
    
    echo "---"
}

# 检查服务是否运行
echo "📋 检查服务状态..."
if ! curl -s "$BASE_URL" > /dev/null; then
    echo -e "${RED}❌ 服务未运行或无法连接到 $BASE_URL${NC}"
    echo "请先启动服务: make run 或 go run main.go"
    exit 1
fi

echo -e "${GREEN}✅ 服务运行正常${NC}"
echo ""

# 1. 健康检查
test_endpoint "GET" "$API_BASE/health" "" "健康检查"

# 2. 指标监控
test_endpoint "GET" "$API_BASE/metrics" "" "获取监控指标"

# 3. 发送简单通知
test_endpoint "POST" "$API_BASE/notifications" '{
    "type": "notice",
    "title": "API 测试通知",
    "message": "这是一条来自测试脚本的通知",
    "targets": [
        {
            "type": "email",
            "value": "test@example.com"
        }
    ]
}' "发送简单通知"

# 4. 发送告警
test_endpoint "POST" "$API_BASE/alert" '{
    "title": "🚨 测试告警",
    "message": "这是一个测试告警消息",
    "priority": 4,
    "targets": [
        {
            "type": "email",
            "value": "alert@example.com"
        }
    ],
    "variables": {
        "server": "test-server",
        "error": "test error"
    },
    "metadata": {
        "environment": "test"
    }
}' "发送告警"

# 5. 发送报告
test_endpoint "POST" "$API_BASE/report" '{
    "title": "📊 测试报告",
    "message": "系统运行正常",
    "targets": [
        {
            "type": "email",
            "value": "report@example.com"
        }
    ],
    "variables": {
        "uptime": "99.9%",
        "requests": 1000
    }
}' "发送报告"

# 6. 异步发送
test_endpoint "POST" "$API_BASE/notifications" '{
    "type": "notice",
    "title": "异步测试",
    "message": "这是一条异步发送的消息",
    "async": true,
    "targets": [
        {
            "type": "email",
            "value": "async@example.com"
        }
    ]
}' "异步发送通知"

# 7. 复杂通知（多目标、模板变量）
test_endpoint "POST" "$API_BASE/notifications" '{
    "type": "alert",
    "title": "复杂通知测试",
    "message": "这是一个包含多个目标和变量的复杂通知",
    "priority": 5,
    "targets": [
        {
            "type": "email",
            "value": "admin@example.com"
        },
        {
            "type": "group",
            "value": "test-group",
            "platform": "feishu"
        }
    ],
    "variables": {
        "server": "prod-web-01",
        "cpu_usage": "85%",
        "memory_usage": "78%",
        "timestamp": "2024-01-01T10:00:00Z"
    },
    "metadata": {
        "severity": "high",
        "category": "system",
        "environment": "production"
    },
    "retry_count": 3,
    "timeout_seconds": 30
}' "复杂通知（多目标+变量）"

# 8. 测试错误情况
echo -e "${YELLOW}🔍 测试错误处理...${NC}"

# 无效请求
test_endpoint "POST" "$API_BASE/notifications" '{
    "invalid": "request"
}' "无效请求测试"

# 缺少必填字段
test_endpoint "POST" "$API_BASE/notifications" '{
    "type": "notice"
}' "缺少必填字段测试"

echo ""
echo "=================================="
echo -e "${GREEN}🎉 API 测试完成！${NC}"
echo ""
echo "💡 提示："
echo "  - 查看服务日志了解详细处理情况"
echo "  - 检查健康状态: curl $API_BASE/health"
echo "  - 查看指标: curl $API_BASE/metrics"
echo ""
echo "📖 更多 API 文档请参考 README.md"