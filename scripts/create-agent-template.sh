#!/bin/bash

# 创建智能体模板项目脚手架
# 用法: ./create-agent-template.sh <project-name>

set -e

PROJECT_NAME=${1:-"my-agent"}
echo "🚀 创建智能体项目: $PROJECT_NAME"

# 检查是否提供了项目名称
if [ -z "$PROJECT_NAME" ]; then
    echo "❌ 请提供项目名称"
    echo "用法: $0 <project-name>"
    exit 1
fi

# 创建项目目录
echo "📁 创建项目目录..."
mkdir -p "$PROJECT_NAME"
cd "$PROJECT_NAME"

# 创建目录结构
echo "📂 创建目录结构..."
mkdir -p {cmd/server,internal/{agent,config,monitoring,security,storage},pkg/{api,models,utils},configs,deployments/{docker,k8s},docs,scripts,tests,tools}

# 创建 Go 模块
echo "📦 初始化 Go 模块..."
go mod init "$PROJECT_NAME"

# 添加核心依赖
echo "📚 添加依赖包..."
go get github.com/gin-gonic/gin
go get github.com/golang-jwt/jwt/v5
go get github.com/prometheus/client_golang
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
go get github.com/go-redis/redis/v8
go get gorm.io/gorm
go get gorm.io/driver/sqlite
go get github.com/joho/godotenv
go get github.com/sirupsen/logrus
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock

# 创建主程序
echo "🔧 创建主程序..."
cat > cmd/server/main.go << 'EOF'
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"$PROJECT_NAME/internal/config"
	"$PROJECT_NAME/internal/agent"
	"$PROJECT_NAME/internal/monitoring"
	"$PROJECT_NAME/pkg/api"
)

func main() {
	// 加载配置
	cfgManager := config.NewManager(config.Development)
	if err := cfgManager.Load("configs/config.yaml"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg := cfgManager.Get()
	log.Printf("🚀 Starting %s in %s mode", "$PROJECT_NAME", cfg.Server.Host)

	// 初始化监控
	metrics := monitoring.NewMetricsCollector()
	healthChecker := monitoring.NewHealthChecker()

	// 注册健康检查
	healthChecker.RegisterCheck("server", func(ctx context.Context) error {
		return nil // 简单的健康检查
	})

	// 初始化智能体管理器
	agentManager := agent.NewManager(cfg)
	if err := agentManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize agent manager: %v", err)
	}

	// 初始化 HTTP 服务器
	server := api.NewServer(cfg, agentManager, metrics, healthChecker)

	// 启动监控服务器
	metricsServer := monitoring.NewMetricsServer(metrics, cfg.Monitoring.MetricsPort)
	go func() {
		if err := metricsServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 启动主服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("🌐 Server listening on %s", addr)
		if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// 关闭监控服务器
	if err := metricsServer.Stop(ctx); err != nil {
		log.Printf("Metrics server forced to shutdown: %v", err)
	}

	// 关闭智能体管理器
	if err := agentManager.Shutdown(); err != nil {
		log.Printf("Agent manager forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped")
}
EOF

# 创建配置文件
echo "⚙️ 创建配置文件..."
cat > configs/config.yaml << 'EOF'
server:
  host: "localhost"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

agent:
  max_concurrent: 50
  max_idle_time: 30m
  health_check_interval: 30s
  max_retries: 3
  retry_delay: 5s
  session_timeout: 60m
  max_history: 100

llm:
  provider: "openai"
  model: "gpt-4"
  api_key: ""
  temperature: 0.7
  max_tokens: 4096
  timeout: 60s

security:
  jwt_secret: "your-secret-key"
  session_timeout: 24h
  rate_limit_enabled: true
  rate_limit_rps: 10
  cors_enabled: true

monitoring:
  enabled: true
  metrics_port: 9090
  tracing_enabled: false
  health_check_enabled: true

logging:
  level: "info"
  format: "json"
  output: "stdout"

cache:
  type: "memory"
  ttl: 1h
  max_size: 1000

features:
  artifacts_enabled: true
  tools_enabled: true
  websocket_enabled: true
EOF

# 创建环境变量文件
echo "🔐 创建环境变量文件..."
cat > .env.example << 'EOF'
# 服务器配置
SERVER_HOST=localhost
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# LLM 配置
LLM_PROVIDER=openai
LLM_MODEL=gpt-4
LLM_API_KEY=your-openai-api-key
LLM_TEMPERATURE=0.7
LLM_MAX_TOKENS=4096

# 智能体配置
AGENT_MAX_CONCURRENT=50
AGENT_MAX_IDLE_TIME=30m
AGENT_HEALTH_CHECK_INTERVAL=30s

# 安全配置
JWT_SECRET=your-super-secret-jwt-key
SESSION_TIMEOUT=24h
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=10

# 监控配置
MONITORING_ENABLED=true
METRICS_PORT=9090
TRACING_ENABLED=false

# 日志配置
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout

# 缓存配置
CACHE_TYPE=memory
CACHE_TTL=1h
CACHE_MAX_SIZE=1000

# 功能开关
FEATURES_ARTIFACTS=true
FEATURES_TOOLS=true
FEATURES_WEBSOCKET=true
EOF

# 创建 Dockerfile
echo "🐳 创建 Dockerfile..."
cat > Dockerfile << 'EOF'
# 多阶段构建 Dockerfile
FROM golang:1.21-alpine AS builder

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

# 最终阶段
FROM alpine:latest

# 安装 ca-certificates
RUN apk --no-cache add ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 创建配置目录
RUN mkdir -p /app/configs

# 复制配置文件
COPY --from=builder /app/configs/config.yaml /app/configs/

# 更改文件所有者
RUN chown -R appuser:appgroup /app

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
CMD ["./main"]
EOF

# 创建 Kubernetes 部署文件
echo "☸️ 创建 Kubernetes 部署文件..."
mkdir -p deployments/k8s

cat > deployments/k8s/deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-template
  labels:
    app: agent-template
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agent-template
  template:
    metadata:
      labels:
        app: agent-template
    spec:
      containers:
      - name: agent-template
        image: agent-template:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: agent-template-service
spec:
  selector:
    app: agent-template
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: ClusterIP
---
apiVersion: v1
kind: Service
metadata:
  name: agent-template-service-loadbalancer
spec:
  selector:
    app: agent-template
  ports:
  - name: http
    port: 80
    targetPort: 8080
  type: LoadBalancer
EOF

# 创建 Makefile
echo "🔨 创建 Makefile..."
cat > Makefile << 'EOF'
.PHONY: build run test clean docker-build docker-run docker-push deploy-dev deploy-prod

# 变量定义
PROJECT_NAME := $(shell basename $(CURDIR))
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
DOCKER_REGISTRY := your-registry
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(PROJECT_NAME):$(VERSION)

# Go 相关命令
build:
	@echo "🔨 Building $(PROJECT_NAME)..."
	go build -o bin/$(PROJECT_NAME) cmd/server/main.go

run:
	@echo "🚀 Running $(PROJECT_NAME)..."
	go run cmd/server/main.go

test:
	@echo "🧪 Running tests..."
	go test -v ./...

test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	@echo "🔍 Running linter..."
	golangci-lint run

fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...

mod-tidy:
	@echo "📦 Tidying modules..."
	go mod tidy

mod-download:
	@echo "📦 Downloading modules..."
	go mod download

# Docker 相关命令
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	@echo "🐳 Running Docker container..."
	docker run -p 8080:8080 -p 9090:9090 $(DOCKER_IMAGE)

docker-push:
	@echo "🐳 Pushing Docker image..."
	docker push $(DOCKER_IMAGE)

# 部署命令
deploy-dev:
	@echo "🚀 Deploying to development..."
	kubectl apply -f deployments/k8s/ -n development

deploy-prod:
	@echo "🚀 Deploying to production..."
	kubectl apply -f deployments/k8s/ -n production

# 清理命令
clean:
	@echo "🧹 Cleaning up..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# 开发工具
install-tools:
	@echo "🛠️ Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 开发模式
dev:
	@echo "🔥 Starting development server with hot reload..."
	air -c .air.toml

# 生成文档
docs:
	@echo "📚 Generating documentation..."
	swag init -g cmd/server/main.go

# 帮助信息
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  mod-tidy       - Tidy Go modules"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-push    - Push Docker image"
	@echo "  deploy-dev     - Deploy to development"
	@echo "  deploy-prod    - Deploy to production"
	@echo "  clean          - Clean build artifacts"
	@echo "  install-tools  - Install development tools"
	@echo "  dev            - Start development server"
	@echo "  docs           - Generate documentation"
	@echo "  help           - Show this help message"
EOF

# 创建 Air 配置文件
echo "🔥 创建 Air 配置..."
cat > .air.toml << 'EOF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main cmd/server/main.go"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = true

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
EOF

# 创建 README 文件
echo "📖 创建 README 文件..."
cat > README.md << 'EOF'
# '$PROJECT_NAME'

基于 LangGraphGo 的智能体模板项目。

## 特性

- 🤖 完整的智能体生命周期管理
- ⚙️ 多环境配置支持
- 📊 完整的监控和指标收集
- 🔒 多层安全防护
- 🚀 高性能和可扩展
- 🐳 容器化部署
- ☸️ Kubernetes 支持

## 快速开始

### 本地开发

\`\`\`bash
# 安装依赖
make install-tools
make mod-download

# 复制环境变量
cp .env.example .env

# 编辑环境变量
vim .env

# 启动开发服务器
make dev
\`\`\`

### 生产部署

\`\`\`bash
# 构建 Docker 镜像
make docker-build

# 部署到开发环境
make deploy-dev

# 部署到生产环境
make deploy-prod
\`\`\`

## 项目结构

\`\`\`
.
├── cmd/server/          # 应用程序入口
├── internal/            # 私有代码
│   ├── agent/          # 智能体核心
│   ├── config/         # 配置管理
│   ├── monitoring/     # 监控系统
│   ├── security/       # 安全组件
│   └── storage/        # 存储抽象
├── pkg/                 # 公共代码
│   ├── api/           # API 定义
│   ├── models/        # 数据模型
│   └── utils/         # 工具函数
├── configs/             # 配置文件
├── deployments/         # 部署配置
├── docs/               # 文档
├── scripts/            # 脚本
├── tests/              # 测试
└── tools/              # 开发工具
\`\`\`

## API 文档

启动服务后访问：
- HTTP API: http://localhost:8080
- 监控指标: http://localhost:9090/metrics

## 开发指南

参见 [开发指南](docs/development.md) 了解详细的开发指南。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
EOF

# 创建 .gitignore
echo "📝 创建 .gitignore..."
cat > .gitignore << 'EOF'
# Binaries
bin/
tmp/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out
coverage.html

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
logs/
*.log

# Environment variables
.env
.env.local
.env.production

# Docker
.dockerignore

# Air
.air.toml

# Coverage
coverage.out
EOF

# 创建开发目录
echo "📂 创建开发目录..."
mkdir -p {docs,tools,logs,data}

# 创建 API 文档目录
mkdir -p docs/api
mkdir -p docs/architecture
mkdir -p docs/deployment

# 创建工具目录
mkdir -p tools/migration
mkdir -p tools/backup

echo "✅ 项目创建完成！"
echo ""
echo "📋 下一步操作："
echo "1. cd $PROJECT_NAME"
echo "2. 复制并编辑环境变量：cp .env.example .env"
echo "3. 安装开发工具：make install-tools"
echo "4. 启动开发服务器：make dev"
echo "5. 访问 http://localhost:8080"
echo ""
echo "📚 更多信息请查看 README.md"