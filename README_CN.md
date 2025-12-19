# LangGraphGo 聊天智能体 - 智能对话应用

一个基于 Go 和 LangGraphGo 构建的现代化智能聊天应用，集成AI智能体、多会话管理、工具支持和本地持久化存储。

[![License](https://img.shields.io/:license-MIT-blue.svg)](https://opensource.org/license/apache-2-0) [![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/smallnest/langchat) [![github actions](https://github.com/smallnest/langchat/actions/workflows/go.yaml/badge.svg)](https://github.com/smallnest/langchat/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/smallnest/langchat)](https://goreportcard.com/report/github.com/smallnest/langchat) 

[English](./README.md) | [简体中文](./README_CN.md)


## ✨ 核心特性

### 🤖 智能对话功能
- **多会话支持**: 创建和管理多个独立的聊天会话
- **AI 智能体**: 基于 LangGraphGo 的先进对话智能体
- **上下文记忆**: 自动维护对话历史和上下文
- **多模型支持**: 支持 OpenAI、Azure OpenAI、百度千帆等主流模型
- **实时流式响应**: 基于 Server-Sent Events 的流式聊天体验

### 🛠️ 工具生态集成
- **智能工具选择**: LLM驱动的自动工具选择机制
- **Skills 工具系统**: 可扩展的技能包管理和加载
- **MCP 协议支持**: Model Context Protocol 工具集成
- **工具进度跟踪**: 实时显示工具执行进度和状态

### 🔐 企业级功能
- **JWT 认证授权**: 基于角色的访问控制系统
- **用户管理**: 完整的用户注册、登录、会话管理
- **速率限制**: API 请求保护和 DDoS 防护
- **安全中间件**: CORS、安全头设置等安全防护

### 📊 监控运维体系
- **Prometheus 指标**: HTTP请求、Agent状态、LLM调用全方位监控
- **健康检查**: `/health`、`/ready`、`/info` 多层次健康检查端点
- **配置热重载**: 支持 JSON/YAML 配置文件实时监听和重载
- **优雅关闭**: 完善的资源清理和超时处理机制

### 🎨 现代化用户界面
- **响应式 Web UI**: 支持深色/浅色主题切换
- **会话管理**: 直观的会话创建、查看、清空、删除操作
- **用户反馈**: 消息质量评估和智能反馈收集
- **实时更新**: 无需刷新页面的实时界面更新

## 🏗️ 项目架构

```
langchat/
├── main.go                     # 应用程序入口点
├── pkg/                        # Go 核心业务包
│   ├── agent/                  # 智能体管理模块
│   │   └── agent.go           # 智能体生命周期和状态管理
│   ├── api/                    # HTTP API 处理器
│   │   ├── auth.go            # 认证相关 API 接口
│   │   └── static.go          # 静态文件服务和处理
│   ├── auth/                   # 认证服务核心
│   │   └── auth.go            # JWT 用户认证实现
│   ├── chat/                   # 聊天核心功能
│   │   └── chat.go            # 聊天服务器和流式响应处理
│   ├── config/                 # 配置管理系统
│   │   └── config.go          # 热重载配置系统实现
│   ├── middleware/             # HTTP 中间件
│   │   └── auth.go            # JWT 认证中间件
│   ├── monitoring/             # 监控指标收集
│   │   └── metrics.go         # Prometheus 指标收集器
│   └── session/                # 会话管理模块
│       └── session.go         # 会话持久化存储实现
├── static/                     # 前端静态资源
│   ├── index.html             # 主应用页面
│   ├── css/                   # 样式文件目录
│   ├── js/                    # JavaScript 脚本文件
│   ├── images/                # 图片资源库
│   └── lib/                   # 第三方 JavaScript 库
├── configs/                    # 应用配置文件
│   ├── config.json            # JSON 格式配置文件
│   └── config.yaml            # YAML 格式配置文件
├── sessions/                   # 本地会话存储目录（自动创建）
├── deployments/                # 部署配置文件
├── scripts/                    # 构建和部署脚本
├── docs/                      # 项目文档中心
├── Dockerfile                 # Docker 容器构建配置
├── Makefile                   # 构建自动化脚本
├── go.mod                     # Go 模块依赖定义
└── go.sum                     # 依赖版本锁定文件
```

## 🚀 快速开始

### 环境要求
- Go 1.19 或更高版本
- OpenAI API Key 或兼容的 LLM 服务

### 方式一：使用 Makefile（推荐方式）

```bash
# 克隆项目到本地
git clone https://github.com/your-repo/langchat.git
cd langchat

# 安装项目依赖
go mod download

# 配置环境变量
cp configs/config.json.example configs/config.json
# 编辑 configs/config.json，添加你的 API Key

# 启动开发服务器（支持热重载）
make dev

# 或者构建并运行生产版本
make build
./bin/langchat
```

### 方式二：标准 Go 命令行

```bash
# 安装依赖包
go mod download

# 配置环境变量
export OPENAI_API_KEY="your-api-key-here"
export PORT="8080"

# 直接运行应用
go run main.go
```

### 访问应用
- **应用主页**: http://localhost:8080
- **登录页面**: http://localhost:8080/login
- **演示账号**:
  - 管理员: 用户名 `admin`，密码 `admin123`
  - 普通用户: 用户名 `user`，密码 `user123`

## ⚙️ 配置说明

### 配置文件结构

应用同时支持 JSON 和 YAML 两种格式的配置文件：

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

### 支持的 LLM 提供商配置

#### OpenAI 配置
```json
{
  "llm": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "sk-your-openai-key"
  }
}
```

#### Azure OpenAI 配置
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

#### 百度千帆配置
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

#### 本地模型 (Ollama) 配置
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

## 📡 API 接口文档

### 用户认证相关
- `POST /api/auth/login` - 用户登录接口
- `POST /api/auth/register` - 用户注册接口
- `POST /api/auth/refresh` - 刷新访问令牌
- `POST /api/auth/logout` - 用户登出接口
- `GET /api/auth/me` - 获取当前用户信息

### 会话管理接口
- `POST /api/sessions/new` - 创建新的聊天会话
- `GET /api/sessions` - 获取用户所有会话列表
- `DELETE /api/sessions/:id` - 删除指定会话
- `GET /api/sessions/:id/history` - 获取会话历史消息

### 聊天功能接口
- `POST /api/chat` - 发送聊天消息（支持流式响应）
- `POST /api/feedback` - 提交消息质量反馈

### 工具和配置接口
- `GET /api/mcp/tools` - 获取 MCP 工具列表
- `GET /api/tools/hierarchical` - 获取分层工具结构
- `GET /api/config` - 获取应用配置信息

### 监控和健康检查接口
- `GET /health` - 应用健康状态检查
- `GET /ready` - 应用就绪状态检查
- `GET /info` - 服务器详细信息
- `GET /metrics` - Prometheus 监控指标

## 🧩 核心组件详解

### ChatServer 聊天服务器
- **智能体生命周期管理**: 基于状态的智能体生命周期控制
- **流式响应处理**: 基于 SSE 的实时数据流处理
- **会话隔离机制**: 基于客户端的会话分离和安全隔离
- **工具集成框架**: Skills 和 MCP 工具的无缝集成机制

### SimpleChatAgent 智能体
- **上下文管理**: 自动维护对话历史和上下文连贯性
- **智能工具选择**: 基于 LLM 推理的智能工具选择算法
- **异步初始化**: 后台工具预加载，避免首次请求延迟
- **错误恢复机制**: 健壮的错误处理和自动重试策略

### 认证授权系统
- **JWT 认证**: 无状态的用户认证和令牌管理
- **角色权限管理**: 支持管理员和普通用户权限控制
- **会话管理**: 基于 Cookie 的安全会话管理
- **演示账号**: 内置开发和测试用演示账号

### 监控运维系统
- **多维度指标监控**: HTTP、Agent、LLM、系统资源全方位监控
- **健康状态检查**: 多层次的应用健康状态监控机制
- **性能追踪**: 请求响应时间和处理量性能监控
- **Prometheus 集成**: 标准化的监控指标输出和集成

## 🐳 Docker 容器化部署

### 构建镜像
```bash
# 构建 Docker 镜像
docker build -t langchat .
```

### 运行容器
```bash
# 运行容器实例
docker run -p 8080:8080 \
  -e OPENAI_API_KEY="your-api-key" \
  -e OPENAI_MODEL="gpt-4" \
  -v $(pwd)/sessions:/app/sessions \
  langchat
```

### Docker Compose 编排部署
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

### 项目模块结构说明

#### pkg/agent/ - 智能体管理
- 智能体状态机：uninitialized → initializing → ready → running → stopped
- 健康检查和生命周期管理机制
- 并发控制和资源管理策略

#### pkg/chat/ - 聊天核心功能
- HTTP 路由和中间件配置
- 流式响应处理和 SSE 实现
- 会话管理和消息持久化

#### pkg/config/ - 配置管理系统
- 支持热重载的动态配置系统
- 环境变量和配置文件的双重支持
- 配置验证和类型转换机制

#### pkg/session/ - 会话持久化
- JSON 格式的会话数据存储
- 用户反馈收集和智能评估
- 会话历史管理和限制控制

### 开发工作流程

```bash
# 安装开发工具和依赖
make setup-dev

# 启用热重载开发模式
make dev

# 执行代码质量检查
make check

# 运行测试套件
make test

# 构建生产版本
make build
```

### 新功能开发指南

1. **新增 API 端点**: 在 `pkg/chat/chat.go` 中添加路由处理逻辑
2. **新增配置选项**: 在 `pkg/config/config.go` 中扩展配置结构
3. **新增前端功能**: 修改 `static/` 目录下的相关文件
4. **新增工具集成**: 在 `pkg/agent/agent.go` 中集成工具逻辑

## 🧪 测试策略

```bash
# 运行完整测试套件
make test

# 运行测试并生成覆盖率报告
make coverage

# 运行特定模块的测试
go test ./pkg/chat -v

# 运行性能基准测试
go test -bench=. ./pkg/...
```

## 📦 构建和发布

### 本地构建
```bash
# 构建当前平台的可执行文件
make build

# 交叉编译构建多平台版本
make build-all
```

### 版本发布
```bash
# 创建完整的发布包
make release

# 发布包输出目录：build/release/
```

## 🔍 故障排除指南

### 常见问题解决

**"API key not configured" 问题**
```bash
# 检查配置文件中的 API key 设置
cat configs/config.json | grep api_key
# 或者通过环境变量设置
export OPENAI_API_KEY="your-api-key"
```

**"Port already in use" 端口占用问题**
```bash
# 使用不同的端口号
export PORT=3000
go run main.go
```

**"Tools not loading" 工具加载失败**
- 检查 MCP 配置路径是否正确
- 验证 Skills 目录权限设置
- 查看应用日志中的详细错误信息

**"High memory usage" 内存使用过高**
```bash
# 调整会话历史限制
export MAX_HISTORY=20

# 调整最大并发数限制
export AGENT_MAX_CONCURRENT=10
```

### 调试模式
```bash
# 启用详细日志输出
export LOG_LEVEL=debug
go run main.go
```

## 📈 性能优化策略

- **会话懒加载**: 仅在需要时加载会话历史数据
- **工具异步初始化**: 后台预加载避免首次请求延迟
- **内存管理优化**: LRU 缓存策略和定期清理机制
- **高并发处理**: 基于 Goroutine 的高效并发请求处理

## 🔒 安全防护特性

- JWT 令牌认证和自动刷新机制
- CORS 跨域请求防护
- 输入数据验证和安全清理
- 速率限制和 DDoS 攻击防护
- 安全的配置管理和敏感信息保护

## 🤝 贡献指南

我们欢迎社区贡献！请遵循以下步骤：

1. Fork 本项目到你的 GitHub 账户
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交你的更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request 并详细描述你的更改

## 📄 开源许可

本项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 📚 相关文档资源

- [📖 项目文档中心](./docs/) - 完整的设计文档和实施计划
- [🚀 部署运维指南](./docs/DEPLOYMENT.md) - 详细的部署和运维最佳实践
- [🔧 API 接口参考](./docs/API_REFERENCE.md) - 完整的 API 接口文档
- [🧪 测试策略指南](./docs/TESTING.md) - 测试策略和最佳实践指南

## 🆘 技术支持

如果你在使用过程中遇到问题或有任何疑问，请：

1. 首先查看 [常见问题解答](./docs/FAQ.md)
2. 搜索现有的 [Issues](https://github.com/your-repo/langchat/issues)
3. 创建新的 Issue 并提供详细的问题描述和复现步骤

---

**🎯 LangGraphGo 聊天智能体 - 构建下一代智能对话应用的完整解决方案！**

*📚 文档维护: LangGraphGo 开发团队*
*🕒 最后更新: 2025-12-19*
*📧 技术支持: 请通过 GitHub Issues 提交反馈*