# LangGraphGo Chat - 智能聊天应用

一个基于 Go 和 LangGraphGo 的现代化智能聊天应用框架，集成了AI智能体、多会话管理、工具支持和本地持久化存储。

[![License](https://img.shields.io/:license-MIT-blue.svg)](https://opensource.org/license/apache-2-0) [![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/smallnest/langchat) [![github actions](https://github.com/smallnest/langchat/actions/workflows/go.yaml/badge.svg)](https://github.com/smallnest/langchat/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/smallnest/langchat)](https://goreportcard.com/report/github.com/smallnest/langchat) 

[English](./README.md) | [简体中文](./README_CN.md)


## ✨ 核心特性

### 🤖 智能聊天功能
- **多会话支持**: 创建和管理多个独立的聊天会话
- **AI 智能体**: 基于 LangGraphGo 的先进对话智能体
- **上下文记忆**: 自动维护对话历史和上下文
- **多模型支持**: 支持 OpenAI、Azure OpenAI、百度千帆等
- **实时流式响应**: 基于 Server-Sent Events 的流式聊天

### 🛠️ 工具集成
- **智能工具选择**: LLM驱动的自动工具选择
- **Skills 工具系统**: 可扩展的技能包管理
- **MCP 协议支持**: Model Context Protocol 工具集成
- **工具进度跟踪**: 实时显示工具执行进度

### 🔐 企业级功能
- **JWT 认证授权**: 基于角色的访问控制
- **用户管理**: 注册、登录、会话管理
- **速率限制**: API 请求保护机制
- **安全中间件**: CORS、安全头设置

### 📊 监控运维
- **Prometheus 指标**: HTTP请求、Agent状态、LLM调用监控
- **健康检查**: `/health`、`/ready`、`/info` 端点
- **配置热重载**: 支持 JSON/YAML 配置文件监听
- **优雅关闭**: 完善的资源清理和超时处理

### 🎨 用户界面
- **现代化 Web UI**: 响应式设计，支持深色/浅色主题
- **会话管理**: 创建、查看、清空、删除会话
- **用户反馈**: 消息质量评估和收集
- **实时更新**: 无需刷新的实时界面更新

## 🏗️ 项目架构

```
langchat/
├── main.go                     # 应用程序入口
├── pkg/                        # Go 核心包
│   ├── agent/                  # 智能体管理
│   │   └── agent.go           # 智能体生命周期和状态管理
│   ├── api/                    # HTTP API 处理器
│   │   ├── auth.go            # 认证相关 API
│   │   └── static.go          # 静态文件服务
│   ├── auth/                   # 认证服务
│   │   └── auth.go            # JWT 用户认证
│   ├── chat/                   # 聊天核心功能
│   │   └── chat.go            # 聊天服务器和流式响应
│   ├── config/                 # 配置管理
│   │   └── config.go          # 热重载配置系统
│   ├── middleware/             # HTTP 中间件
│   │   └── auth.go            # JWT 认证中间件
│   ├── monitoring/             # 监控指标
│   │   └── metrics.go         # Prometheus 指标收集
│   └── session/                # 会话管理
│       └── session.go         # 会话持久化存储
├── static/                     # 前端静态资源
│   ├── index.html             # 主页面
│   ├── css/                   # 样式文件
│   ├── js/                    # JavaScript 文件
│   ├── images/                # 图片资源
│   └── lib/                   # 第三方库
├── configs/                    # 配置文件
│   ├── config.json            # JSON 格式配置
│   └── config.yaml            # YAML 格式配置
├── sessions/                   # 本地会话存储（自动创建）
├── deployments/                # 部署配置
├── scripts/                    # 构建和部署脚本
├── docs/                      # 项目文档
├── Dockerfile                 # Docker 容器配置
├── Makefile                   # 构建自动化
├── go.mod                     # Go 模块定义
└── go.sum                     # 依赖版本锁定
```

## 🚀 快速开始

### 环境要求
- Go 1.19 或更高版本
- OpenAI API Key 或兼容的 LLM 服务

### 方式一：使用 Makefile（推荐）

```bash
# 克隆项目
git clone https://github.com/your-repo/langchat.git
cd langchat

# 安装依赖
go mod download

# 配置环境变量
cp configs/config.json.example configs/config.json
# 编辑 configs/config.json，添加你的 API Key

# 运行开发服务器
make dev

# 或者构建并运行
make build
./bin/langchat
```

### 方式二：标准 Go 命令

```bash
# 安装依赖
go mod download

# 配置环境变量
export OPENAI_API_KEY="your-api-key-here"
export PORT="8080"

# 运行应用
go run main.go
```

### 访问应用
- 应用地址: http://localhost:8080
- 登录页面: http://localhost:8080/login
- 演示账号:
  - 管理员: `admin` / `admin123`
  - 普通用户: `user` / `user123`

## ⚙️ 配置说明

### 配置文件结构

应用支持 JSON 和 YAML 两种格式的配置文件：

```json
{
  "server": {
    "host": "localhost",
    "port": 8080,
    "read_timeout": 30000000000,
    "write_timeout": 30000000000
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "your-api-key-here",
    "temperature": 0.7,
    "max_tokens": 4096
  },
  "auth": {
    "jwt_secret": "your-secret-key",
    "session_timeout": 86400000000000,
    "rate_limit_enabled": true,
    "rate_limit_rps": 10
  },
  "agent": {
    "max_concurrent": 50,
    "max_idle_time": 1800000000000,
    "health_check_interval": 30000000000,
    "session_timeout": 3600000000000,
    "max_history": 100
  },
  "monitoring": {
    "enabled": true,
    "metrics_port": 9090,
    "health_check_enabled": true
  }
}
```

### 支持的 LLM 提供商

#### OpenAI
```json
{
  "llm": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "sk-your-openai-key"
  }
}
```

#### Azure OpenAI
```json
{
  "llm": {
    "provider": "azure",
    "model": "your-deployment-name",
    "api_key": "your-azure-key",
    "base_url": "https://your-resource.openai.azure.com/"
  }
}
```

#### 百度千帆
```json
{
  "llm": {
    "provider": "baidu",
    "model": "ERNIE-Bot",
    "api_key": "your-baidu-token",
    "base_url": "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions"
  }
}
```

#### 本地模型 (Ollama)
```json
{
  "llm": {
    "provider": "ollama",
    "model": "llama2",
    "api_key": "not-needed",
    "base_url": "http://localhost:11434/v1"
  }
}
```

## 📡 API 接口

### 认证相关
- `POST /api/auth/login` - 用户登录
- `POST /api/auth/register` - 用户注册
- `POST /api/auth/refresh` - 刷新访问令牌
- `POST /api/auth/logout` - 用户登出
- `GET /api/auth/me` - 获取当前用户信息

### 会话管理
- `POST /api/sessions/new` - 创建新会话
- `GET /api/sessions` - 获取所有会话
- `DELETE /api/sessions/:id` - 删除会话
- `GET /api/sessions/:id/history` - 获取会话历史

### 聊天功能
- `POST /api/chat` - 发送消息（支持流式响应）
- `POST /api/feedback` - 提交消息反馈

### 工具和配置
- `GET /api/mcp/tools` - 获取 MCP 工具列表
- `GET /api/tools/hierarchical` - 获取分层工具结构
- `GET /api/config` - 获取应用配置

### 监控和健康检查
- `GET /health` - 健康检查
- `GET /ready` - 就绪检查
- `GET /info` - 服务器信息
- `GET /metrics` - Prometheus 指标

## 🧩 核心组件

### ChatServer
- **智能体生命周期管理**: 状态驱动的智能体生命周期
- **流式响应处理**: 基于 SSE 的实时响应流
- **会话隔离**: 基于客户端的会话分离
- **工具集成**: Skills 和 MCP 工具的无缝集成

### SimpleChatAgent
- **上下文管理**: 自动维护对话历史和上下文
- **智能工具选择**: 基于 LLM 推理的工具选择
- **异步初始化**: 后台工具预加载，避免首次请求延迟
- **错误恢复**: 健壮的错误处理和自动重试

### 认证系统
- **JWT 认证**: 无状态的用户认证机制
- **角色权限**: 支持管理员和普通用户角色
- **会话管理**: 基于 Cookie 的会话管理
- **演示账号**: 内置开发测试账号

### 监控系统
- **多维度指标**: HTTP、Agent、LLM、系统资源指标
- **健康检查**: 全面的应用健康状态监控
- **性能追踪**: 请求响应时间和处理量监控
- **Prometheus 集成**: 标准化的指标输出

## 🐳 Docker 部署

### 构建镜像
```bash
docker build -t langchat .
```

### 运行容器
```bash
docker run -p 8080:8080 \
  -e OPENAI_API_KEY="your-api-key" \
  -e OPENAI_MODEL="gpt-4" \
  -v $(pwd)/sessions:/app/sessions \
  langchat
```

### Docker Compose
```yaml
version: '3.8'
services:
  langchat:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OPENAI_MODEL=gpt-4
    volumes:
      - ./sessions:/app/sessions
      - ./configs:/app/configs
    restart: unless-stopped
```

## 🔧 开发指南

### 项目结构说明

#### pkg/agent/ - 智能体管理
- 智能体状态机：uninitialized → initializing → ready → running → stopped
- 健康检查和生命周期管理
- 并发控制和资源管理

#### pkg/chat/ - 聊天核心
- HTTP 路由和中间件配置
- 流式响应处理和 SSE 实现
- 会话管理和消息存储

#### pkg/config/ - 配置管理
- 支持热重载的配置系统
- 环境变量和配置文件的双重支持
- 配置验证和类型转换

#### pkg/session/ - 会话持久化
- JSON 格式的会话数据存储
- 用户反馈收集和评估
- 会话历史管理和限制

### 开发工作流

```bash
# 安装开发工具
make setup-dev

# 启用热重载开发
make dev

# 代码质量检查
make check

# 运行测试
make test

# 构建生产版本
make build
```

### 添加新功能

1. **新的 API 端点**: 在 `pkg/chat/chat.go` 中添加路由处理
2. **新的配置选项**: 在 `pkg/config/config.go` 中添加配置结构
3. **新的前端功能**: 修改 `static/` 目录下的文件
4. **新的工具集成**: 在 `pkg/agent/agent.go` 中添加工具逻辑

## 🧪 测试

```bash
# 运行所有测试
make test

# 运行测试并生成覆盖率报告
make coverage

# 运行特定包的测试
go test ./pkg/chat -v

# 运行性能基准测试
go test -bench=. ./pkg/...
```

## 📦 构建和发布

### 本地构建
```bash
# 构建当前平台
make build

# 交叉编译构建
make build-all
```

### 发布版本
```bash
# 创建发布包
make release

# 输出目录：build/release/
```

## 🔍 故障排除

### 常见问题

**"API key not configured"**
```bash
# 检查配置文件
cat configs/config.json | grep api_key
# 或设置环境变量
export OPENAI_API_KEY="your-key"
```

**"Port already in use"**
```bash
# 使用不同端口
export PORT=3000
go run main.go
```

**"Tools not loading"**
- 检查 MCP 配置路径
- 验证 Skills 目录权限
- 查看应用日志中的错误信息

**"High memory usage"**
```bash
# 调整会话历史限制
export MAX_HISTORY=20

# 调整最大并发数
export AGENT_MAX_CONCURRENT=10
```

### 调试模式
```bash
# 启用详细日志
export LOG_LEVEL=debug
go run main.go
```

## 📈 性能优化

- **会话懒加载**: 仅在需要时加载会话历史
- **工具异步初始化**: 后台预加载避免首次请求延迟
- **内存管理**: LRU 缓存和定期清理
- **并发处理**: 基于 Goroutine 的高并发请求处理

## 🔒 安全特性

- JWT 令牌认证和刷新机制
- CORS 跨域请求保护
- 输入验证和清理
- 速率限制和 DDoS 防护
- 安全的配置管理

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 📚 相关文档

- [📖 项目文档中心](./docs/) - 完整的设计文档和实施计划
- [🚀 部署指南](./docs/DEPLOYMENT.md) - 详细的部署和运维指南
- [🔧 API 参考](./docs/API_REFERENCE.md) - 完整的 API 文档
- [🧪 测试指南](./docs/TESTING.md) - 测试策略和最佳实践

## 🆘 支持

如果您遇到问题或有任何疑问，请：

1. 查看 [FAQ](./docs/FAQ.md)
2. 搜索现有的 [Issues](https://github.com/your-repo/langchat/issues)
3. 创建新的 Issue 并提供详细信息

---

**🎯 LangGraphGo Chat - 构建下一代智能聊天应用的完整解决方案！**