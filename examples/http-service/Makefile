# NotifyHub HTTP Service Makefile

.PHONY: help run build test clean docker-build docker-run dev deps

# 默认目标
help: ## 显示帮助信息
	@echo "NotifyHub HTTP Service - 可用命令:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# 开发相关
dev: deps ## 开发模式运行（自动重启）
	@echo "🔧 开发模式启动..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "📦 安装 air 工具..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

run: ## 直接运行服务
	@echo "🚀 启动 NotifyHub HTTP Service..."
	@go run main.go

deps: ## 安装依赖
	@echo "📦 安装依赖..."
	@go mod tidy
	@go mod download

# 构建相关
build: ## 构建二进制文件
	@echo "🔨 构建应用..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/notifyhub-http-service main.go
	@echo "✅ 构建完成: bin/notifyhub-http-service"

build-local: ## 构建本地版本
	@echo "🔨 构建本地版本..."
	@go build -o bin/notifyhub-http-service main.go
	@echo "✅ 构建完成: bin/notifyhub-http-service"

# 测试相关
test: ## 运行测试
	@echo "🧪 运行测试..."
	@go test -v ./...

test-coverage: ## 运行测试并生成覆盖率报告
	@echo "📊 生成测试覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告: coverage.html"

# Docker 相关
docker-build: ## 构建 Docker 镜像
	@echo "🐳 构建 Docker 镜像..."
	@docker build -t notifyhub-http-service:latest .
	@echo "✅ Docker 镜像构建完成"

docker-run: ## 运行 Docker 容器
	@echo "🐳 运行 Docker 容器..."
	@docker run -p 8080:8080 --env-file .env notifyhub-http-service:latest

# 服务管理
start: build-local ## 构建并启动服务
	@echo "🚀 启动服务..."
	@./bin/notifyhub-http-service

stop: ## 停止服务（如果在后台运行）
	@echo "🛑 停止服务..."
	@pkill -f notifyhub-http-service || true

restart: stop start ## 重启服务

# 工具命令
fmt: ## 格式化代码
	@echo "🎨 格式化代码..."
	@go fmt ./...

lint: ## 运行代码检查
	@echo "🔍 运行代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "📦 请安装 golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 清理
clean: ## 清理构建文件
	@echo "🧹 清理构建文件..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✅ 清理完成"

# 示例和测试
example-curl: ## 运行 curl 示例测试
	@echo "📡 测试 API 接口..."
	@echo "健康检查:"
	@curl -s http://localhost:8080/api/v1/health | jq '.' || curl -s http://localhost:8080/api/v1/health
	@echo ""
	@echo "发送测试通知:"
	@curl -s -X POST http://localhost:8080/api/v1/notifications \
		-H "Content-Type: application/json" \
		-d '{"type":"notice","title":"测试","message":"Hello NotifyHub!","targets":[{"type":"email","value":"test@example.com"}]}' \
		| jq '.' || echo "需要安装 jq 来格式化 JSON 输出"

setup: ## 初始化项目环境
	@echo "🔧 初始化项目环境..."
	@cp .env.example .env || true
	@echo "✅ 请编辑 .env 文件配置您的通知渠道"
	@echo "📖 然后运行: make run"

# 生产部署
deploy-prep: test build ## 准备生产部署
	@echo "🚀 准备生产部署..."
	@echo "✅ 测试通过，二进制文件已构建"

# 版本信息
version: ## 显示版本信息
	@echo "NotifyHub HTTP Service v1.2.0"
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# 默认目标
.DEFAULT_GOAL := help