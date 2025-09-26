#!/bin/bash

# 企业微信推送示例设置脚本
# WeChat Work Push Example Setup Script

set -e

echo "=== 企业微信推送示例设置 / WeChat Work Push Example Setup ==="

# 检查 Go 版本
echo "检查 Go 环境 / Checking Go environment..."
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go 1.24+ / Go not installed, please install Go 1.24+ first"
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | cut -c 3-)
echo "✅ Go 版本: $GO_VERSION / Go version: $GO_VERSION"

# 检查环境变量
echo "检查环境变量 / Checking environment variables..."

if [[ -z "${WECHATWORK_WEBHOOK_URL}" ]]; then
    echo "⚠️  WECHATWORK_WEBHOOK_URL 未设置，将使用测试地址"
    echo "⚠️  WECHATWORK_WEBHOOK_URL not set, will use test endpoint"
    echo "   设置方法 / How to set:"
    echo "   export WECHATWORK_WEBHOOK_URL=\"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key\""
else
    echo "✅ WECHATWORK_WEBHOOK_URL: ${WECHATWORK_WEBHOOK_URL:0:50}..."
fi

if [[ -z "${WECHATWORK_SECRET}" ]]; then
    echo "ℹ️  WECHATWORK_SECRET 未设置（可选，用于签名验证）"
    echo "ℹ️  WECHATWORK_SECRET not set (optional, for signature verification)"
else
    echo "✅ WECHATWORK_SECRET: 已设置 / Set"
fi

if [[ -z "${WECHATWORK_KEYWORDS}" ]]; then
    echo "ℹ️  WECHATWORK_KEYWORDS 未设置（可选，用于关键词验证）"
    echo "ℹ️  WECHATWORK_KEYWORDS not set (optional, for keyword verification)"
else
    echo "✅ WECHATWORK_KEYWORDS: ${WECHATWORK_KEYWORDS}"
fi

# 构建项目
echo "构建项目 / Building project..."
if go build -o wechatwork-example .; then
    echo "✅ 构建成功 / Build successful"
else
    echo "❌ 构建失败 / Build failed"
    exit 1
fi

# 检查可执行文件
if [[ -f "./wechatwork-example" ]]; then
    echo "✅ 可执行文件创建成功 / Executable created successfully"
    echo "   文件大小 / File size: $(du -h ./wechatwork-example | cut -f1)"
else
    echo "❌ 可执行文件创建失败 / Executable creation failed"
    exit 1
fi

echo ""
echo "=== 设置完成 / Setup Complete ==="
echo ""
echo "运行示例 / Run example:"
echo "  ./wechatwork-example"
echo ""
echo "或直接运行 / Or run directly:"
echo "  go run main.go"
echo ""
echo "环境变量配置 / Environment variable configuration:"
echo "  export WECHATWORK_WEBHOOK_URL=\"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key\""
echo "  export WECHATWORK_SECRET=\"your_secret\"           # 可选 / Optional"
echo "  export WECHATWORK_KEYWORDS=\"通知\"                # 可选 / Optional"
echo ""
echo "使用配置文件 / Using config file:"
echo "  编辑 config.yaml 然后运行 / Edit config.yaml then run"
echo ""
echo "✅ 企业微信外部平台集成示例已准备就绪！"
echo "✅ WeChat Work external platform integration example is ready!"