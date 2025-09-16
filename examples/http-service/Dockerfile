# 多阶段构建 Dockerfile - 生产级最佳实践

# 构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用 - 生成静态链接的二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo -ldflags='-w -s -extldflags "-static"' \
    -o notifyhub-http-service main.go

# 运行阶段
FROM scratch

# 从构建阶段复制必要文件
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/notifyhub-http-service /notifyhub-http-service

# 设置时区
ENV TZ=Asia/Shanghai

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/notifyhub-http-service", "--health-check"] || exit 1

# 运行应用
ENTRYPOINT ["/notifyhub-http-service"]

# 元数据标签
LABEL maintainer="NotifyHub Team" \
      version="1.2.0" \
      description="NotifyHub HTTP Service - Optimized notification service" \
      org.opencontainers.image.title="NotifyHub HTTP Service" \
      org.opencontainers.image.description="Production-ready HTTP service for NotifyHub" \
      org.opencontainers.image.version="1.2.0" \
      org.opencontainers.image.source="https://github.com/kart-io/notifyhub"