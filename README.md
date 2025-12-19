# LangChat Application

A sophisticated web-based multi-session chat application with AI agent integration, tool support, and persistent local storage.

## ✨ Features

### Core Chat Features
- 🔄 **Multi-Session Support**: Create and manage multiple independent chat sessions
- 💾 **Persistent Storage**: All conversations automatically saved to local disk
- 🌐 **Modern Web Interface**: Clean, responsive web UI with real-time updates
- 🤖 **AI Chat Agent**: Advanced agent with conversation history management
- 🔧 **Tool Integration**: Support for Skills and MCP (Model Context Protocol) tools
- 🔌 **Multi-Provider Support**: Works with OpenAI, Baidu, Azure, and any OpenAI-compatible API
- 🎨 **Beautiful UI**: Dark/light theme support with smooth animations
- 📝 **Session Management**: Create, view, clear, and delete sessions
- ⚡ **Hot Reload**: Development mode with automatic code reloading
- 🐳 **Docker Support**: Containerized deployment ready

### Enterprise Features
- 🔐 **Authentication & Authorization**: JWT-based auth with user roles and protected endpoints
- 📊 **Monitoring & Metrics**: Prometheus metrics collection for HTTP requests, agents, and LLM calls
- 🏥 **Health Checks**: Comprehensive health monitoring with `/health`, `/ready`, and `/info` endpoints
- ⚙️ **Configuration Management**: Hot-reloadable configuration with file watching and env var support
- 🔄 **Streaming Responses**: Real-time chat streaming with Server-Sent Events (SSE)
- 🛡️ **Graceful Shutdown**: Proper resource cleanup and timeout handling
- 🚦 **Rate Limiting**: Configurable request rate limiting for API protection
- 📈 **Performance Monitoring**: System metrics tracking and agent lifecycle management

### Advanced Agent Features
- 🎯 **Agent Lifecycle Management**: State-based agent lifecycle with health monitoring
- 🔍 **Tool Selection**: Intelligent tool and skill selection using LLM-based reasoning
- ⚡ **Tool Pre-warming**: Asynchronous tool loading to prevent first-request delays
- 🎛️ **Session Isolation**: Client-based session separation with cookie management
- 💬 **Message Feedback**: User feedback system for message quality assessment
- 🔧 **Error Recovery**: Robust error handling with automatic retries and fallbacks

## 🏗️ Architecture

```
showcases/chat/
├── main.go                 # Application entry point and server bootstrap
├── pkg/                    # Go packages
│   ├── agent/             # Agent lifecycle management
│   │   └── agent.go       # Agent state and health monitoring
│   ├── api/               # API handlers and static content
│   │   ├── auth.go        # Authentication API endpoints
│   │   └── static.go      # Static file serving
│   ├── auth/              # Authentication service
│   │   └── auth.go        # JWT-based user authentication
│   ├── chat/              # Chat server and agent logic
│   │   └── chat.go        # Core chat functionality with streaming
│   ├── config/            # Configuration management
│   │   └── config.go      # Hot-reloadable configuration
│   ├── middleware/        # HTTP middleware
│   │   └── auth.go        # JWT authentication middleware
│   ├── monitoring/        # Monitoring and metrics
│   │   └── metrics.go     # Prometheus metrics and health checks
│   └── session/           # Session management
│       └── session.go     # Session persistence with feedback
├── static/
│   ├── index.html        # Web frontend
│   ├── style.css         # UI styles
│   └── script.js         # Frontend logic with streaming support
├── sessions/             # Local session storage (auto-created)
├── configs/              # Configuration files (optional)
│   └── config.json       # Hot-reloadable config
├── build/                # Build output directory
├── Makefile              # Build automation
├── Dockerfile            # Docker configuration
├── .air.toml            # Hot reload configuration
├── go.mod
├── go.sum
├── .env                 # Configuration (create from .env.example)
└── README.md
```

## 🚀 Quick Start

### Option 1: Using Makefile (Recommended)

```bash
# Clone and navigate to the project
cd showcases/chat

# Install development tools
make setup-dev

# Copy environment template
cp .env.example .env

# Edit .env and add your OpenAI API key
# OPENAI_API_KEY=sk-...

# Run with hot reload (development mode)
make dev

# Or run normally
make run-dev
```

### Option 2: Standard Go Commands

```bash
cd showcases/chat

# Install dependencies
go mod download

# Copy environment template
cp .env.example .env

# Edit .env and add your OpenAI API key
# OPENAI_API_KEY=sk-...

# Build and run
go run main.go
```

The server will start at `http://localhost:8080`

## 🛠️ Development Workflow

### Using Makefile

```bash
# Install development tools (air, golangci-lint, etc.)
make setup-dev

# Run with hot reload
make dev

# Run all checks (format, lint, vet, test)
make check

# Build for production
make build

# Build for all platforms
make build-all
```

### Common Makefile Targets

| Target           | Description              |
| ---------------- | ------------------------ |
| `make dev`       | Run with hot reload      |
| `make run-dev`   | Run with dev environment |
| `make build`     | Build the application    |
| `make test`      | Run tests                |
| `make coverage`  | Run tests with coverage  |
| `make format`    | Format code              |
| `make vet`       | Vet code                 |
| `make lint`      | Lint code                |
| `make docker-up` | Build and run Docker     |
| `make clean`     | Clean build artifacts    |
| `make help`      | Show all targets         |

## ⚙️ Configuration

Environment variables (in `.env`):

```env
# Required: Your API key
OPENAI_API_KEY=your-api-key-here

# Optional: Model name (default: gpt-4o-mini)
OPENAI_MODEL=gpt-4o-mini

# Optional: Base URL for OpenAI-compatible APIs
# Examples:
#   Baidu: https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions
#   Azure: https://your-resource.openai.azure.com/
#   Ollama: http://localhost:11434/v1
OPENAI_BASE_URL=

# Optional: Server port (default: 8080)
PORT=8080

# Optional: Session storage directory (default: ./sessions)
SESSION_DIR=./sessions

# Optional: Maximum messages per session (default: 50)
MAX_HISTORY_SIZE=50

# Optional: Skills directory (for tool integration)
SKILLS_DIR=../../testdata/skills

# Optional: MCP configuration path
MCP_CONFIG_PATH=../../testdata/mcp/mcp.json

# Optional: Chat title
CHAT_TITLE=LangChat

# Optional: Authentication settings
JWT_SECRET=your-jwt-secret-key-here
SESSION_TIMEOUT=24h

# Optional: Monitoring settings
MONITORING_ENABLED=true
METRICS_PORT=9090
HEALTH_CHECK_ENABLED=true

# Optional: Rate limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=10

# Optional: Agent settings
AGENT_MAX_CONCURRENT=50
AGENT_MAX_IDLE_TIME=30m
AGENT_HEALTH_CHECK_INTERVAL=30s

# Optional: Feature flags
FEATURES_FEEDBACK=true
FEATURES_WEBSOCKET=true
FEATURES_MCP=true
```

### LLM Provider Examples

**OpenAI**:
```env
OPENAI_API_KEY=sk-your-openai-key
OPENAI_MODEL=gpt-4o
```

**Baidu Qianfan**:
```env
OPENAI_API_KEY=your-baidu-token
OPENAI_BASE_URL=https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions
OPENAI_MODEL=ERNIE-Bot
```

**Azure OpenAI**:
```env
OPENAI_API_KEY=your-azure-key
OPENAI_BASE_URL=https://your-resource.openai.azure.com/
OPENAI_MODEL=your-deployment-name
```

**Local Models (Ollama, LM Studio)**:
```env
OPENAI_API_KEY=not-needed
OPENAI_BASE_URL=http://localhost:11434/v1
OPENAI_MODEL=llama2
```

## 📡 API Endpoints

### Authentication
- `POST /api/auth/login` - User login with credentials
- `POST /api/auth/register` - User registration
- `POST /api/auth/refresh` - Refresh access token
- `POST /api/auth/logout` - User logout
- `GET /api/auth/me` - Get current user info

### Sessions
- `POST /api/sessions/new` - Create a new session
- `GET /api/sessions` - List all sessions
- `DELETE /api/sessions/:id` - Delete a session
- `GET /api/sessions/:id/history` - Get session messages
- `GET /api/client-id` - Get current client ID

### Chat
- `POST /api/chat` - Send a message (supports streaming)
  ```json
  {
    "session_id": "uuid",
    "message": "your message",
    "user_settings": {
      "enable_skills": true,
      "enable_mcp": true
    },
    "stream": false
  }
  ```
  Response (non-streaming):
  ```json
  {
    "response": "AI response text",
    "message_id": "uuid"
  }
  ```

  Streaming response uses Server-Sent Events (SSE).

### Tools
- `GET /api/mcp/tools?session_id=:id` - List available MCP tools
- `GET /api/tools/hierarchical?session_id=:id` - Get tools in hierarchical structure
- `GET /api/config` - Get chat configuration

### Feedback
- `POST /api/feedback` - Submit feedback on messages
  ```json
  {
    "session_id": "uuid",
    "message_id": "uuid",
    "feedback": "like|dislike"
  }
  ```

### Monitoring & Health
- `GET /health` - Health check endpoint
- `GET /ready` - Readiness probe
- `GET /info` - Server information and status
- `GET /metrics` - Prometheus metrics (redirects to metrics port)

## 🧩 Components

### ChatAgent

The `SimpleChatAgent` provides:
- Automatic conversation context management
- Intelligent tool and skill selection using LLM reasoning
- Tool integration (Skills and MCP) with progress tracking
- Support for OpenAI-compatible APIs
- Thread-safe conversation history with message feedback
- Asynchronous tool loading with pre-warming
- Streaming response support with real-time updates
- Graceful error handling and recovery mechanisms

### Authentication System

The JWT-based authentication includes:
- User registration and login with role-based access control
- Access and refresh token management with configurable timeouts
- Protected API endpoints with middleware enforcement
- Cookie-based session management for browser clients
- Demo user accounts for testing and development

### Monitoring & Metrics

The monitoring system provides:
- Prometheus metrics for HTTP requests, response times, and sizes
- Agent lifecycle metrics (sessions, messages, errors, token usage)
- LLM provider metrics (requests, duration, token usage, errors)
- System metrics (memory, CPU, goroutine counts)
- Health check endpoints with configurable probes
- Separate metrics server for production deployments

### Configuration Management

The configuration system supports:
- Hot-reloadable JSON/YAML configuration files
- Environment variable overrides with type conversion
- Configuration validation with error reporting
- File system watching for automatic reloads
- Environment-specific settings (development, staging, production)

### Session Management

Each session includes:
- Unique UUID identifier with client-based isolation
- Complete message history with role-based organization
- Persistent JSON storage with feedback support
- Client-based separation using cookies and IP tracking
- Automatic saving and loading with configurable history limits
- User feedback collection for message quality assessment

### Tool Integration

The application supports two types of tools:

1. **Skills**: Pre-defined tool packages loaded from `SKILLS_DIR`
   - Intelligent skill selection using LLM reasoning
   - Lazy loading with caching for performance
   - Hierarchical organization with descriptions

2. **MCP Tools**: Dynamic tools from Model Context Protocol servers
   - Automatic tool discovery and categorization
   - Error recovery and timeout handling
   - Real-time execution progress tracking

### Agent Lifecycle Management

The agent lifecycle system provides:
- State-based agent management (uninitialized → initializing → ready → running → stopped)
- Health monitoring with configurable check intervals
- Automatic idle timeout handling with graceful shutdown
- Retry policies with exponential backoff
- Resource cleanup and memory management
- Lifecycle event callbacks and metrics collection

## 🐳 Docker Deployment

```bash
# Build and run with Docker Compose
make docker-up

# Or manually:
docker build -t chat-app .
docker run -p 8080:8080 -e OPENAI_API_KEY=your-key chat-app
```

### Docker Compose

```yaml
version: '3.8'
services:
  chat:
    build: .
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OPENAI_MODEL=gpt-4o-mini
    volumes:
      - ./sessions:/app/sessions
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run specific test
go test ./pkg/session -v
```

## 📦 Building

### Build for Current Platform
```bash
make build
```

### Cross-Platform Builds
```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux
make build-darwin
make build-windows
```

### Release Packages
```bash
# Create release packages
make release
```

Outputs will be in `build/release/`.

## 🔧 Customization

### Change System Prompt

Edit `pkg/chat/chat.go` in the `NewSimpleChatAgent` function:
```go
systemMsg := llms.MessageContent{
    Role:  llms.ChatMessageTypeSystem,
    Parts: []llms.ContentPart{llms.TextPart("Your custom system message here")},
}
```

### Add Custom Tools

1. Create a skill package in your skills directory
2. Follow the skill package structure from the examples
3. Tools will be automatically loaded

### Modify UI

Edit files in `static/`:
- `index.html` - Main HTML structure
- `style.css` - Styles and themes
- `script.js` - Frontend logic

## 🚀 Streaming Chat

The application supports real-time streaming responses:

### Client-side Streaming
```javascript
const response = await fetch('/api/chat', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    session_id: 'uuid',
    message: 'your message',
    stream: true
  })
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;

  const chunk = decoder.decode(value);
  // Process SSE events
}
```

### Server-sent Events Format
- `event: start` - Stream initiation
- `event: chunk` - Response content chunks
- `event: end` - Stream completion with full response
- `event: error` - Error events with details

## 🏢 Enterprise Features

### Authentication & Authorization
- JWT-based authentication with configurable secrets
- Role-based access control (admin, user roles)
- Session management with timeout policies
- Protected API endpoints with middleware

### Monitoring & Observability
- Prometheus metrics collection on configurable port
- Health check endpoints (`/health`, `/ready`, `/info`)
- System metrics (memory, CPU, goroutines)
- Request tracing and performance monitoring
- Error tracking and alerting integration

### Configuration Management
- Hot-reloadable configuration (JSON/YAML)
- Environment variable overrides
- Configuration validation and watching
- Environment-specific settings
- Runtime configuration updates

### Graceful Shutdown
- Configurable shutdown timeouts
- Resource cleanup and connection management
- In-flight request completion
- Agent state preservation

## 🔍 Development

### Project Structure

- **main.go**: Application entry point, bootstrap, and graceful shutdown
- **pkg/agent/**: Agent lifecycle management and health monitoring
- **pkg/api/**: HTTP handlers for authentication and static content
- **pkg/auth/**: JWT-based user authentication service
- **pkg/chat/**: Core chat functionality with streaming support
- **pkg/config/**: Hot-reloadable configuration management
- **pkg/middleware/**: HTTP middleware for authentication
- **pkg/monitoring/**: Prometheus metrics and health checks
- **pkg/session/**: Session persistence with feedback support
- **static/**: Web frontend assets with streaming UI
- **configs/**: Configuration files (optional)

### Adding Features

1. **New API endpoints**: Add to `pkg/chat/chat.go`
2. **New session fields**: Update `pkg/session/session.go`
3. **Frontend changes**: Modify `static/` files
4. **Configuration**: Add to environment variables

### Code Quality

The project uses:
- `go fmt` for formatting
- `go vet` for static analysis
- `golangci-lint` for comprehensive linting
- Tests for critical functionality

Run `make check` to run all quality checks.

## 🐛 Troubleshooting

### Common Issues

**"OPENAI_API_KEY environment variable not set"**
```bash
cp .env.example .env
# Edit .env and add your key
```

**Port already in use**
```bash
PORT=3000 make run-dev
```

**Tools not loading**
- Check `SKILLS_DIR` environment variable
- Verify MCP configuration path
- Check logs for error messages

**Build errors**
```bash
make clean
make deps
make build
```

### Debug Mode

Enable verbose logging:
```env
LOG_LEVEL=debug
```

## 📈 Performance

- **Session Loading**: Lazy loading of session history
- **Tool Initialization**: Asynchronous background loading
- **Memory Management**: LRU-based session caching
- **Concurrent Requests**: Goroutine-based request handling

## 🔒 Security

- No user authentication (single-user mode)
- Local storage only (no cloud dependencies)
- Input validation and sanitization
- CORS configuration for API access

## 🗺️ Roadmap

### ✅ Completed Features
- [x] **Streaming chat responses** - Server-Sent Events (SSE) implementation
- [x] **Multi-user support with authentication** - JWT-based auth with roles
- [x] **Enterprise monitoring** - Prometheus metrics and health checks
- [x] **Hot configuration reloading** - File-based configuration watching
- [x] **Agent lifecycle management** - State-based agent monitoring
- [x] **Message feedback system** - User feedback collection
- [x] **Graceful shutdown handling** - Proper resource cleanup
- [x] **Rate limiting** - Configurable request throttling
- [x] **Health check endpoints** - `/health`, `/ready`, `/info` endpoints

### 🚧 In Progress
- [ ] **Session export/import functionality** - Backup and restore conversations
- [ ] **Advanced tool management UI** - Interactive tool configuration
- [ ] **Database integration** - PostgreSQL/MySQL support for production
- [ ] **WebSocket support** - Real-time bidirectional communication

### 📋 Planned Features
- [ ] **Voice input/output support** - Speech-to-text and text-to-speech
- [ ] **Plugin system for custom tools** - Dynamic tool loading
- [ ] **Real-time collaboration features** - Multi-user sessions
- [ ] **File upload capabilities** - Document and image processing
- [ ] **Advanced analytics dashboard** - Usage insights and trends
- [ ] **API rate limiting per user** - Individual user quotas
- [ ] **Message search and filtering** - Content discovery within sessions
- [ ] **Custom branding support** - White-label customization options

## 📄 License

This project is part of LangGraphGo and follows the same license.

## 📚 文档和指南

### 核心文档
- **[📖 文档中心](./docs/)** - 完整的设计文档、实施计划和总结报告
  - [智能体开发最佳实践指南](./docs/AGENT_TEMPLATE_GUIDE.md) - 架构设计和最佳实践
  - [智能体模板改进计划](./docs/AGENT_TEMPLATE_IMPROVEMENTS.md) - 实施路线图（90%完成）
  - [最终完成报告](./docs/FINAL_COMPLETION_REPORT.md) - 100%诺言兑现验证
  - [集成完成总结](./docs/INTEGRATION_SUMMARY.md) - 企业级功能集成记录

### Roadmap & Tasks
- **[📋 优化任务清单](./docs/TODOs.md)** - 详细的优化路线图和任务清单
  - **多模态支持**（图像、音频、文档处理）
  - **高级智能体功能**（记忆系统、规划系统、自适应学习）
  - **分布式智能体协作**（Agent间通信、任务协调）
  - **性能监控增强**（分布式追踪、可视化Dashboard）
  - **企业级功能**（数据库集成、消息队列）
  - **开发工具**（SDK、CLI、调试工具）

### 开发指南
- [LangGraphGo Documentation](https://github.com/smallnest/langgraphgo)
- [Makefile Guide](./Makefile.README.md)
- [LangChain Go](https://github.com/tmc/langchaingo)
- [MCP Specification](https://modelcontextprotocol.io/)