#!/bin/bash

# 飞书通知示例演示脚本
# 使用模拟的 Webhook URL 进行演示

echo "🚀 飞书通知示例演示"
echo "=================="
echo ""
echo "注意：这是一个演示脚本，使用模拟的 Webhook URL"
echo "实际使用时请设置真实的飞书 Webhook URL"
echo ""

# 设置模拟的环境变量
export FEISHU_WEBHOOK_URL=""
export FEISHU_SECRET=""

echo "环境配置："
echo "  FEISHU_WEBHOOK_URL: $FEISHU_WEBHOOK_URL"
echo "  FEISHU_SECRET: $FEISHU_SECRET"
echo ""

echo "正在运行飞书通知示例..."
echo "========================"

# 运行示例程序
cd ../cmd/example && go run main.go

echo ""
echo "演示完成！"
echo ""
echo "要使用真实的飞书 Webhook："
echo "1. 在飞书群聊中创建自定义机器人"
echo "2. 获取 Webhook URL"
echo "3. 设置环境变量："
echo "   export FEISHU_WEBHOOK_URL=\"https://open.feishu.cn/open-apis/bot/v2/hook/your-token\""
echo "   export FEISHU_SECRET=\"your-secret\""
echo "4. 运行示例：go run main.go"