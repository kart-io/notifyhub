#!/bin/bash

# 飞书推送示例设置脚本（模板集成版本）
# Feishu Push Example Setup Script (Template Integration Version)

set -e

echo "=== 飞书推送示例设置（模板集成）/ Feishu Push Example Setup (Template Integration) ==="

# 检查 Go 版本
echo "检查 Go 环境 / Checking Go environment..."
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go 1.24+ / Go not installed, please install Go 1.24+ first"
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | cut -c 3-)
echo "✅ Go 版本: $GO_VERSION / Go version: $GO_VERSION"

# 检查模板文件
echo "检查模板文件 / Checking template files..."

TEMPLATE_DIR="./templates"
if [[ ! -d "$TEMPLATE_DIR" ]]; then
    echo "❌ 模板目录不存在: $TEMPLATE_DIR"
    echo "❌ Template directory not found: $TEMPLATE_DIR"
    exit 1
fi

REQUIRED_TEMPLATES=("alert.tmpl" "system_status.tmpl" "deployment.tmpl" "user_activity.mustache")
MISSING_TEMPLATES=()

for template in "${REQUIRED_TEMPLATES[@]}"; do
    if [[ ! -f "$TEMPLATE_DIR/$template" ]]; then
        MISSING_TEMPLATES+=("$template")
    else
        echo "✅ 找到模板: $template / Found template: $template"
    fi
done

if [[ ${#MISSING_TEMPLATES[@]} -gt 0 ]]; then
    echo "❌ 缺少以下模板文件 / Missing template files:"
    for template in "${MISSING_TEMPLATES[@]}"; do
        echo "   - $template"
    done
    exit 1
fi

echo "✅ 所有模板文件存在 / All template files exist"

# 检查环境变量
echo "检查环境变量 / Checking environment variables..."

if [[ -z "${FEISHU_WEBHOOK_URL}" ]]; then
    echo "⚠️  FEISHU_WEBHOOK_URL 未设置，将使用测试地址"
    echo "⚠️  FEISHU_WEBHOOK_URL not set, will use test endpoint"
    echo "   设置方法 / How to set:"
    echo "   export FEISHU_WEBHOOK_URL=\"https://open.feishu.cn/open-apis/bot/v2/hook/your_hook_id\""
else
    echo "✅ FEISHU_WEBHOOK_URL: ${FEISHU_WEBHOOK_URL:0:50}..."
fi

if [[ -z "${FEISHU_SECRET}" ]]; then
    echo "ℹ️  FEISHU_SECRET 未设置（可选，用于签名验证）"
    echo "ℹ️  FEISHU_SECRET not set (optional, for signature verification)"
else
    echo "✅ FEISHU_SECRET: 已设置 / Set"
fi

if [[ -z "${FEISHU_KEYWORDS}" ]]; then
    echo "ℹ️  FEISHU_KEYWORDS 未设置（可选，用于关键词验证）"
    echo "ℹ️  FEISHU_KEYWORDS not set (optional, for keyword verification)"
else
    echo "✅ FEISHU_KEYWORDS: ${FEISHU_KEYWORDS}"
fi

# 验证模板语法
echo "验证模板语法 / Validating template syntax..."

# 检查 Go 模板语法（简单验证）
for template in "alert.tmpl" "system_status.tmpl" "deployment.tmpl"; do
    if grep -q "{{.*}}" "$TEMPLATE_DIR/$template"; then
        echo "✅ Go 模板语法检查通过: $template / Go template syntax OK: $template"
    else
        echo "⚠️  模板可能没有变量: $template / Template may have no variables: $template"
    fi
done

# 检查 Mustache 模板语法
if grep -q "{{[#^/].*}}" "$TEMPLATE_DIR/user_activity.mustache"; then
    echo "✅ Mustache 模板语法检查通过 / Mustache template syntax OK"
else
    echo "⚠️  Mustache 模板可能语法有误 / Mustache template may have syntax issues"
fi

# 构建项目
echo "构建项目 / Building project..."
if go build -o feishu-example .; then
    echo "✅ 构建成功 / Build successful"
else
    echo "❌ 构建失败 / Build failed"
    exit 1
fi

# 检查可执行文件
if [[ -f "./feishu-example" ]]; then
    echo "✅ 可执行文件创建成功 / Executable created successfully"
    echo "   文件大小 / File size: $(du -h ./feishu-example | cut -f1)"
else
    echo "❌ 可执行文件创建失败 / Executable creation failed"
    exit 1
fi

# 验证配置文件
echo "验证配置文件 / Validating configuration files..."

if [[ -f "./config.yaml" ]]; then
    echo "✅ 配置文件存在: config.yaml / Config file exists: config.yaml"
else
    echo "⚠️  配置文件不存在: config.yaml / Config file not found: config.yaml"
fi

if [[ -f "./template_vars.json" ]]; then
    echo "✅ 模板变量文档存在: template_vars.json / Template variables doc exists: template_vars.json"

    # 简单验证 JSON 格式
    if command -v jq &> /dev/null; then
        if jq empty template_vars.json 2>/dev/null; then
            echo "✅ JSON 格式验证通过 / JSON format validation passed"
        else
            echo "❌ JSON 格式验证失败 / JSON format validation failed"
        fi
    else
        echo "ℹ️  跳过 JSON 验证（jq 未安装）/ Skipping JSON validation (jq not installed)"
    fi
else
    echo "⚠️  模板变量文档不存在 / Template variables doc not found"
fi

echo ""
echo "=== 设置完成 / Setup Complete ==="
echo ""
echo "🎯 **模板功能特性 / Template Features:**"
echo "   ✅ Go template 引擎支持 / Go template engine support"
echo "   ✅ Mustache 模板引擎支持 / Mustache template engine support"
echo "   ✅ 动态变量替换 / Dynamic variable substitution"
echo "   ✅ 条件渲染和循环 / Conditional rendering and loops"
echo "   ✅ 多种消息类型模板 / Multiple message type templates"
echo ""
echo "运行示例 / Run example:"
echo "  ./feishu-example"
echo ""
echo "或直接运行 / Or run directly:"
echo "  go run main.go"
echo ""
echo "环境变量配置 / Environment variable configuration:"
echo "  export FEISHU_WEBHOOK_URL=\"https://open.feishu.cn/open-apis/bot/v2/hook/your_hook_id\""
echo "  export FEISHU_SECRET=\"your_secret\"             # 可选 / Optional"
echo "  export FEISHU_KEYWORDS=\"通知\"                  # 可选 / Optional"
echo ""
echo "模板文件位置 / Template file locations:"
echo "  - 告警模板 / Alert template: templates/alert.tmpl"
echo "  - 状态报告 / Status report: templates/system_status.tmpl"
echo "  - 部署通知 / Deployment: templates/deployment.tmpl"
echo "  - 用户活动 / User activity: templates/user_activity.mustache"
echo ""
echo "模板变量参考 / Template variables reference:"
echo "  - 查看 template_vars.json / See template_vars.json"
echo ""
echo "✅ 飞书模板集成示例已准备就绪！"
echo "✅ Feishu template integration example is ready!"