#!/bin/bash

# 钉钉推送示例环境变量设置脚本

echo "=== 钉钉推送示例配置 ==="
echo ""

# 检查是否已设置环境变量
if [ -n "$DINGTALK_WEBHOOK_URL" ]; then
    echo "✅ DINGTALK_WEBHOOK_URL 已设置"
else
    echo "❌ DINGTALK_WEBHOOK_URL 未设置"
    echo ""
    echo "请按以下步骤获取钉钉 Webhook URL："
    echo "1. 在钉钉群中点击群设置"
    echo "2. 选择 '智能群助手'"
    echo "3. 点击 '添加机器人'"
    echo "4. 选择 '自定义' 机器人"
    echo "5. 设置机器人名称和头像"
    echo "6. 选择安全设置（推荐：加签）"
    echo "7. 复制生成的 Webhook URL"
    echo ""
    echo "然后设置环境变量："
    echo "export DINGTALK_WEBHOOK_URL='https://oapi.dingtalk.com/robot/send?access_token=your_token'"
    echo ""
fi

if [ -n "$DINGTALK_SECRET" ]; then
    echo "✅ DINGTALK_SECRET 已设置 (加签验证)"
else
    echo "⚠️  DINGTALK_SECRET 未设置 (将使用无安全验证模式)"
    echo ""
    echo "如果启用了加签验证，请设置："
    echo "export DINGTALK_SECRET='your_secret_key'"
    echo ""
fi

if [ -n "$DINGTALK_KEYWORDS" ]; then
    echo "✅ DINGTALK_KEYWORDS 已设置: $DINGTALK_KEYWORDS"
else
    echo "⚠️  DINGTALK_KEYWORDS 未设置"
    echo ""
    echo "如果启用了自定义关键词验证，请设置："
    echo "export DINGTALK_KEYWORDS='通知'"
    echo ""
fi

echo "=== 钉钉安全模式说明 ==="
echo ""
echo "钉钉机器人支持三种安全设置："
echo ""
echo "1. 无安全验证："
echo "   - 仅需要 DINGTALK_WEBHOOK_URL"
echo "   - 安全性较低，不推荐生产环境使用"
echo ""
echo "2. 加签验证（推荐）："
echo "   - 需要设置 DINGTALK_WEBHOOK_URL 和 DINGTALK_SECRET"
echo "   - 使用 HMAC-SHA256 签名验证"
echo "   - 安全性高，推荐生产环境使用"
echo ""
echo "3. 自定义关键词："
echo "   - 需要设置 DINGTALK_WEBHOOK_URL 和 DINGTALK_KEYWORDS"
echo "   - 消息必须包含指定关键词"
echo "   - 如果消息不包含关键词，会自动添加"
echo ""
echo "4. 加签+关键词（最高安全性）："
echo "   - 同时设置 DINGTALK_WEBHOOK_URL、DINGTALK_SECRET 和 DINGTALK_KEYWORDS"
echo ""

if [ -n "$DINGTALK_WEBHOOK_URL" ]; then
    echo "=== 准备运行示例 ==="
    echo ""
    echo "环境变量已配置，可以运行示例："
    echo "go run main.go"
    echo ""
else
    echo "=== 配置环境变量后运行 ==="
    echo ""
    echo "配置完环境变量后，运行："
    echo "go run main.go"
    echo ""
fi