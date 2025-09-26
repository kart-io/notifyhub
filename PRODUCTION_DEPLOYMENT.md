# NotifyHub v3.0 生产环境部署指南

## 概述

本文档提供NotifyHub v3.0在生产环境中部署、配置和运维的完整指南。涵盖了从单机部署到大规模集群部署的各种场景，以及性能调优、监控告警和故障处理的最佳实践。

## 部署架构选择

### 场景1: 小型团队/初创公司（<10万条消息/天）

**推荐架构**：单机部署 + Redis
```
[Load Balancer] → [NotifyHub Instance] → [Redis] → [External Services]
```

**资源配置**：
- CPU: 2核心
- 内存: 4GB
- 存储: 50GB SSD
- 网络: 100Mbps

**部署成本**：约$50-100/月

### 场景2: 中型企业（10万-1000万条消息/天）

**推荐架构**：多实例 + Redis集群
```
[Load Balancer] → [NotifyHub Cluster] → [Redis Cluster] → [External Services]
                  └─ Instance 1
                  └─ Instance 2
                  └─ Instance 3
```

**资源配置**：
- 每实例: 4核心, 8GB内存
- 实例数量: 3-5个
- Redis: 3节点集群
- 存储: 100GB SSD
- 网络: 1Gbps

**部署成本**：约$300-500/月

### 场景3: 大型企业（>1000万条消息/天）

**推荐架构**：Kubernetes + 微服务
```
[Ingress] → [NotifyHub Pods] → [Redis Cluster] → [External Services]
           └─ HPA自动伸缩       └─ Persistent Volumes
           └─ Service Mesh     └─ Backup Strategy
```

**资源配置**：
- Pod资源: 2核心, 4GB内存
- 副本数: 5-20个（自动伸缩）
- Redis: 6节点集群（3主3从）
- 存储: 500GB+ SSD
- 网络: 10Gbps+

**部署成本**：约$1000-3000/月

## Docker部署

### 1. 基础Docker镜像

**Dockerfile**：
```dockerfile
# 多阶段构建
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o notifyhub ./cmd/server

# 运行时镜像
FROM alpine:latest

# 安装CA证书和时区数据
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 复制二进制文件
COPY --from=builder /app/notifyhub .

# 创建配置目录
RUN mkdir -p /etc/notifyhub

# 非root用户
RUN addgroup -S notifyhub && adduser -S notifyhub -G notifyhub
USER notifyhub

EXPOSE 8080

CMD ["./notifyhub"]
```

### 2. Docker Compose部署

**docker-compose.yml**：
```yaml
version: '3.8'

services:
  # NotifyHub服务
  notifyhub:
    image: notifyhub:v3.0
    container_name: notifyhub-server
    ports:
      - "8080:8080"
    environment:
      # 基础配置
      - LOG_LEVEL=info
      - LOG_FORMAT=json
      - HTTP_PORT=8080

      # Redis配置
      - REDIS_URL=redis://redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - REDIS_DB=0

      # 邮件配置
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_PORT=${SMTP_PORT}
      - SMTP_USERNAME=${SMTP_USERNAME}
      - SMTP_PASSWORD=${SMTP_PASSWORD}
      - SMTP_FROM=${SMTP_FROM}
      - SMTP_TLS=true

      # 飞书配置
      - FEISHU_WEBHOOK_URL=${FEISHU_WEBHOOK_URL}
      - FEISHU_SECRET=${FEISHU_SECRET}

      # 性能配置
      - ASYNC_ENABLED=true
      - ASYNC_QUEUE_SIZE=2000
      - ASYNC_WORKERS=10
      - CONNECTION_TIMEOUT=30s
      - IDLE_TIMEOUT=90s

    depends_on:
      - redis
      - prometheus
    networks:
      - notifyhub-network
    volumes:
      - ./config:/etc/notifyhub:ro
      - ./logs:/var/log/notifyhub
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # Redis服务
  redis:
    image: redis:7-alpine
    container_name: notifyhub-redis
    ports:
      - "6379:6379"
    environment:
      - REDIS_PASSWORD=${REDIS_PASSWORD}
    volumes:
      - redis-data:/data
      - ./redis.conf:/usr/local/etc/redis/redis.conf:ro
    networks:
      - notifyhub-network
    restart: unless-stopped
    sysctls:
      - net.core.somaxconn=65535
    healthcheck:
      test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Prometheus监控
  prometheus:
    image: prom/prometheus:latest
    container_name: notifyhub-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - notifyhub-network
    restart: unless-stopped

  # Grafana仪表板
  grafana:
    image: grafana/grafana:latest
    container_name: notifyhub-grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources
    networks:
      - notifyhub-network
    restart: unless-stopped

  # Nginx反向代理
  nginx:
    image: nginx:alpine
    container_name: notifyhub-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    networks:
      - notifyhub-network
    depends_on:
      - notifyhub
    restart: unless-stopped

networks:
  notifyhub-network:
    driver: bridge

volumes:
  redis-data:
  prometheus-data:
  grafana-data:
```

### 3. 环境变量配置

**.env文件**：
```bash
# Redis配置
REDIS_PASSWORD=your-super-secure-redis-password

# 邮件配置
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@yourcompany.com

# 飞书配置
FEISHU_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook
FEISHU_SECRET=your-feishu-secret

# Grafana配置
GRAFANA_PASSWORD=admin-password
```

## Kubernetes部署

### 1. Namespace和ConfigMap

**namespace.yaml**：
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: notifyhub
  labels:
    name: notifyhub
```

**configmap.yaml**：
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: notifyhub-config
  namespace: notifyhub
data:
  app.yaml: |
    server:
      port: 8080
      read_timeout: 30s
      write_timeout: 30s
      idle_timeout: 90s

    logging:
      level: info
      format: json
      output: stdout

    async:
      enabled: true
      queue_size: 2000
      workers: 10

    health:
      enabled: true
      endpoint: "/health"

    metrics:
      enabled: true
      endpoint: "/metrics"
```

### 2. Secret管理

**secret.yaml**：
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: notifyhub-secrets
  namespace: notifyhub
type: Opaque
data:
  # base64编码的敏感信息
  redis-password: <base64-encoded-password>
  smtp-password: <base64-encoded-password>
  feishu-secret: <base64-encoded-secret>
```

### 3. Redis部署

**redis-deployment.yaml**：
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
  namespace: notifyhub
spec:
  serviceName: redis
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: notifyhub-secrets
              key: redis-password
        volumeMounts:
        - name: redis-data
          mountPath: /data
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 1Gi
  volumeClaimTemplates:
  - metadata:
      name: redis-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi

---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: notifyhub
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
  type: ClusterIP
```

### 4. NotifyHub主服务部署

**notifyhub-deployment.yaml**：
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notifyhub
  namespace: notifyhub
  labels:
    app: notifyhub
    version: v3.0
spec:
  replicas: 3
  selector:
    matchLabels:
      app: notifyhub
  template:
    metadata:
      labels:
        app: notifyhub
        version: v3.0
    spec:
      containers:
      - name: notifyhub
        image: notifyhub:v3.0
        ports:
        - containerPort: 8080
        env:
        # 基础配置
        - name: LOG_LEVEL
          value: "info"
        - name: HTTP_PORT
          value: "8080"

        # Redis配置
        - name: REDIS_URL
          value: "redis://redis:6379"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: notifyhub-secrets
              key: redis-password

        # 邮件配置
        - name: SMTP_HOST
          value: "smtp.gmail.com"
        - name: SMTP_PORT
          value: "587"
        - name: SMTP_PASSWORD
          valueFrom:
            secretKeyRef:
              name: notifyhub-secrets
              key: smtp-password

        # 资源限制
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 1Gi

        # 健康检查
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 3

        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 3
          successThreshold: 1
          failureThreshold: 3

        # 配置挂载
        volumeMounts:
        - name: config
          mountPath: /etc/notifyhub
          readOnly: true

      volumes:
      - name: config
        configMap:
          name: notifyhub-config

---
apiVersion: v1
kind: Service
metadata:
  name: notifyhub-service
  namespace: notifyhub
spec:
  selector:
    app: notifyhub
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  type: ClusterIP

---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: notifyhub-hpa
  namespace: notifyhub
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: notifyhub
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 5. Ingress配置

**ingress.yaml**：
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: notifyhub-ingress
  namespace: notifyhub
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "1000"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - notifyhub.yourdomain.com
    secretName: notifyhub-tls
  rules:
  - host: notifyhub.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: notifyhub-service
            port:
              number: 8080
```

## 生产环境配置

### 1. 性能调优配置

**config/production.yaml**：
```yaml
# 服务器配置
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 2m
  max_header_bytes: 1048576

# 日志配置
logging:
  level: info
  format: json
  output: stdout

# 异步处理配置
async:
  enabled: true
  queue_size: 5000        # 增大队列
  workers: 20             # 增加工作者
  batch_size: 50          # 批处理大小
  flush_interval: 1s      # 刷新间隔

# Redis配置
redis:
  url: ${REDIS_URL}
  password: ${REDIS_PASSWORD}
  db: 0
  max_retries: 3
  pool_size: 20           # 连接池大小
  min_idle_conns: 5       # 最小空闲连接
  max_conn_age: 30m       # 连接最大生命周期
  pool_timeout: 4s        # 获取连接超时
  idle_timeout: 5m        # 空闲连接超时

# 邮件配置
email:
  smtp_host: ${SMTP_HOST}
  smtp_port: ${SMTP_PORT}
  smtp_username: ${SMTP_USERNAME}
  smtp_password: ${SMTP_PASSWORD}
  smtp_from: ${SMTP_FROM}
  use_tls: true
  timeout: 30s
  pool_size: 10           # SMTP连接池

# 监控配置
monitoring:
  metrics_enabled: true
  metrics_path: /metrics
  health_path: /health
  profiling_enabled: false # 生产环境关闭

# 限流配置
rate_limiting:
  enabled: true
  global_limit: 10000     # 全局限制：10k req/min
  per_ip_limit: 1000      # 每IP限制：1k req/min
  burst: 100              # 突发请求

# 重试配置
retry:
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: 0.1
```

### 2. Nginx配置

**nginx.conf**：
```nginx
upstream notifyhub_backend {
    least_conn;
    server notifyhub:8080 max_fails=3 fail_timeout=30s;
    # 如果是集群，添加更多upstream
    # server notifyhub-2:8080 max_fails=3 fail_timeout=30s;
    # server notifyhub-3:8080 max_fails=3 fail_timeout=30s;
}

# 限流配置
limit_req_zone $binary_remote_addr zone=api:10m rate=100r/s;
limit_req_zone $binary_remote_addr zone=webhook:10m rate=1000r/s;

server {
    listen 80;
    server_name notifyhub.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name notifyhub.yourdomain.com;

    # SSL配置
    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers on;

    # 安全头
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload";

    # 日志配置
    access_log /var/log/nginx/notifyhub.access.log;
    error_log /var/log/nginx/notifyhub.error.log;

    # 限流
    limit_req zone=api burst=200 nodelay;

    # 健康检查
    location /health {
        proxy_pass http://notifyhub_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 健康检查不记录访问日志
        access_log off;
    }

    # API路由
    location /api/ {
        limit_req zone=api burst=100 nodelay;

        proxy_pass http://notifyhub_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 超时配置
        proxy_connect_timeout 5s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # 缓冲配置
        proxy_buffering on;
        proxy_buffer_size 128k;
        proxy_buffers 4 256k;
        proxy_busy_buffers_size 256k;
    }

    # Webhook路由（高流量）
    location /webhook/ {
        limit_req zone=webhook burst=2000 nodelay;

        proxy_pass http://notifyhub_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 监控和告警

### 1. Prometheus配置

**prometheus.yml**：
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "/etc/prometheus/rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  # NotifyHub指标
  - job_name: 'notifyhub'
    static_configs:
      - targets: ['notifyhub:8080']
    metrics_path: /metrics
    scrape_interval: 15s

  # Redis指标
  - job_name: 'redis'
    static_configs:
      - targets: ['redis:6379']

  # 系统指标
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']
```

### 2. 告警规则

**alert-rules.yml**：
```yaml
groups:
- name: notifyhub.rules
  rules:
  # 高错误率告警
  - alert: HighErrorRate
    expr: (rate(notifyhub_requests_total{status="error"}[5m]) / rate(notifyhub_requests_total[5m])) > 0.05
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "NotifyHub high error rate"
      description: "Error rate is {{ $value | humanizePercentage }} for the last 5 minutes"

  # 响应时间过长告警
  - alert: HighResponseTime
    expr: histogram_quantile(0.95, rate(notifyhub_request_duration_seconds_bucket[5m])) > 1
    for: 3m
    labels:
      severity: warning
    annotations:
      summary: "NotifyHub high response time"
      description: "95th percentile response time is {{ $value }}s"

  # 队列积压告警
  - alert: QueueBacklog
    expr: notifyhub_queue_depth > 1000
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "NotifyHub queue backlog"
      description: "Queue depth is {{ $value }}"

  # 内存使用过高告警
  - alert: HighMemoryUsage
    expr: (process_resident_memory_bytes{job="notifyhub"} / 1024 / 1024) > 1024
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "NotifyHub high memory usage"
      description: "Memory usage is {{ $value }}MB"

  # 服务不可用告警
  - alert: ServiceDown
    expr: up{job="notifyhub"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "NotifyHub service is down"
      description: "NotifyHub service has been down for more than 1 minute"
```

### 3. Grafana仪表板

**dashboard.json**（关键指标）：
```json
{
  "dashboard": {
    "id": null,
    "title": "NotifyHub v3.0 Dashboard",
    "tags": ["notifyhub"],
    "timezone": "browser",
    "panels": [
      {
        "title": "Requests Per Second",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(notifyhub_requests_total[1m])",
            "legendFormat": "RPS"
          }
        ]
      },
      {
        "title": "Success Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "(rate(notifyhub_requests_total{status=\"success\"}[5m]) / rate(notifyhub_requests_total[5m])) * 100",
            "legendFormat": "Success Rate %"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(notifyhub_request_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          },
          {
            "expr": "histogram_quantile(0.95, rate(notifyhub_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.99, rate(notifyhub_request_duration_seconds_bucket[5m]))",
            "legendFormat": "99th percentile"
          }
        ]
      }
    ]
  }
}
```

## 安全配置

### 1. 网络安全

**防火墙规则**：
```bash
# 只允许必要端口
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p tcp --dport 22 -j ACCEPT  # SSH
iptables -A INPUT -j DROP

# 限制连接数
iptables -A INPUT -p tcp --dport 80 -m connlimit --connlimit-above 100 -j DROP
iptables -A INPUT -p tcp --dport 443 -m connlimit --connlimit-above 100 -j DROP
```

### 2. 应用安全

**安全配置**：
```yaml
security:
  # API认证
  auth:
    enabled: true
    type: "bearer"  # or "basic", "jwt"

  # 访问控制
  access_control:
    enabled: true
    allow_origins: ["https://yourdomain.com"]
    allow_methods: ["GET", "POST", "PUT", "DELETE"]
    allow_headers: ["Content-Type", "Authorization"]

  # 请求验证
  validation:
    max_request_size: 10MB
    rate_limiting: true
    ip_whitelist: []
    ip_blacklist: []

  # 数据保护
  encryption:
    in_transit: true  # TLS
    at_rest: true     # 配置加密

  # 审计日志
  audit:
    enabled: true
    log_requests: true
    log_responses: false
    sensitive_fields: ["password", "token", "secret"]
```

## 备份和恢复

### 1. 数据备份

**Redis备份脚本**：
```bash
#!/bin/bash
# backup-redis.sh

REDIS_HOST=${REDIS_HOST:-localhost}
REDIS_PORT=${REDIS_PORT:-6379}
REDIS_PASSWORD=${REDIS_PASSWORD}
BACKUP_DIR="/var/backups/notifyhub"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份Redis数据
redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD --rdb $BACKUP_DIR/redis_$TIMESTAMP.rdb

# 压缩备份
gzip $BACKUP_DIR/redis_$TIMESTAMP.rdb

# 清理旧备份（保留7天）
find $BACKUP_DIR -name "redis_*.rdb.gz" -mtime +7 -delete

echo "Redis backup completed: redis_$TIMESTAMP.rdb.gz"
```

### 2. 配置备份

**配置备份脚本**：
```bash
#!/bin/bash
# backup-config.sh

CONFIG_DIR="/etc/notifyhub"
BACKUP_DIR="/var/backups/notifyhub/config"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# 备份配置文件
tar -czf $BACKUP_DIR/config_$TIMESTAMP.tar.gz -C $CONFIG_DIR .

# 备份Kubernetes配置
kubectl get all -n notifyhub -o yaml > $BACKUP_DIR/k8s_$TIMESTAMP.yaml

echo "Configuration backup completed"
```

### 3. 恢复程序

**恢复脚本**：
```bash
#!/bin/bash
# restore.sh

BACKUP_FILE=$1
BACKUP_TYPE=$2  # redis or config

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file> <redis|config>"
    exit 1
fi

case $BACKUP_TYPE in
    "redis")
        echo "Restoring Redis data..."
        gunzip -c $BACKUP_FILE | redis-cli -h localhost -p 6379 --pipe
        ;;
    "config")
        echo "Restoring configuration..."
        tar -xzf $BACKUP_FILE -C /etc/notifyhub/
        ;;
    *)
        echo "Unknown backup type: $BACKUP_TYPE"
        exit 1
        ;;
esac

echo "Restore completed"
```

## 故障处理

### 1. 常见故障排查

**内存泄漏排查**：
```bash
# 查看内存使用
docker stats notifyhub-server

# 获取heap dump
curl http://localhost:8080/debug/pprof/heap > heap.prof

# 分析内存profile
go tool pprof heap.prof
```

**性能问题排查**：
```bash
# CPU profiling
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# 查看goroutine
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

**网络问题排查**：
```bash
# 检查连接状态
netstat -tlnp | grep 8080

# 检查DNS解析
nslookup smtp.gmail.com

# 测试网络连通性
telnet smtp.gmail.com 587
```

### 2. 故障恢复程序

**自动故障恢复**：
```yaml
# 健康检查失败重启
spec:
  containers:
  - name: notifyhub
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 30
      timeoutSeconds: 5
      failureThreshold: 3

    # 资源限制防止资源耗尽
    resources:
      requests:
        cpu: 200m
        memory: 256Mi
      limits:
        cpu: 1000m
        memory: 1Gi
```

**手动恢复步骤**：
```bash
# 1. 检查服务状态
kubectl get pods -n notifyhub

# 2. 查看日志
kubectl logs -f deployment/notifyhub -n notifyhub

# 3. 重启服务
kubectl rollout restart deployment/notifyhub -n notifyhub

# 4. 检查恢复状态
kubectl rollout status deployment/notifyhub -n notifyhub
```

## 性能调优

### 1. 系统级优化

**内核参数优化**：
```bash
# /etc/sysctl.conf
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_keepalive_time = 300
net.ipv4.tcp_keepalive_intvl = 30
net.ipv4.tcp_keepalive_probes = 3
net.ipv4.ip_local_port_range = 1024 65535
```

**文件描述符限制**：
```bash
# /etc/security/limits.conf
notifyhub soft nofile 65535
notifyhub hard nofile 65535
```

### 2. 应用级优化

**Go运行时优化**：
```bash
# 环境变量
export GOGC=100
export GOMAXPROCS=4
export GOMEMLIMIT=1GiB
```

**配置优化**：
```yaml
# 连接池优化
redis:
  pool_size: 50
  min_idle_conns: 10

# 工作者优化
async:
  workers: 20
  queue_size: 10000
  batch_size: 100

# 超时优化
timeouts:
  connect: 5s
  read: 30s
  write: 30s
  idle: 90s
```

## 部署检查清单

### 部署前检查

- [ ] 环境变量配置正确
- [ ] 密钥和证书已准备
- [ ] 网络和防火墙配置
- [ ] 依赖服务（Redis）就绪
- [ ] 监控系统配置完成
- [ ] 备份策略制定

### 部署后验证

- [ ] 健康检查端点响应正常
- [ ] 日志输出正常
- [ ] 指标收集工作
- [ ] 发送功能测试通过
- [ ] 性能基准测试通过
- [ ] 告警规则触发测试

### 运维检查

- [ ] 监控仪表板配置
- [ ] 告警通知渠道测试
- [ ] 备份恢复流程测试
- [ ] 故障演练完成
- [ ] 文档更新完成

## 总结

NotifyHub v3.0的生产环境部署需要综合考虑性能、安全、可靠性和可维护性。通过合理的架构设计、完善的监控告警、自动化的运维流程，可以构建一个稳定高效的通知服务系统。

关键要点：
1. **渐进式部署**：从单机开始，逐步扩展到集群
2. **监控优先**：完善的监控是稳定运行的基础
3. **自动化运维**：减少人工操作，提高可靠性
4. **安全防护**：多层次的安全防护措施
5. **应急准备**：完善的故障处理和恢复流程

---

**文档版本**: 1.0
**适用版本**: NotifyHub v3.0+
**更新时间**: 2025年9月
**维护团队**: DevOps Team