package chat

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/smallnest/goskills"
	mcpclient "github.com/smallnest/goskills/mcp"
	adaptergoskills "github.com/smallnest/langgraphgo/adapter/goskills"
	"github.com/smallnest/langgraphgo/adapter/mcp"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"

	agentpkg "github.com/smallnest/langchat/pkg/agent"
	"github.com/smallnest/langchat/pkg/api"
	"github.com/smallnest/langchat/pkg/auth"
	configpkg "github.com/smallnest/langchat/pkg/config"
	"github.com/smallnest/langchat/pkg/middleware"
	monitoringpkg "github.com/smallnest/langchat/pkg/monitoring"
	sessionpkg "github.com/smallnest/langchat/pkg/session"
)

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SkillInfo stores basic info about a skill
type SkillInfo struct {
	Name        string
	Description string
	Package     *goskills.SkillPackage
	Tools       []tools.Tool // Cached tools for the skill
	Loaded      bool         // Whether tools have been loaded
}

// ChatAgent interface defines the contract for chat agents
type ChatAgent interface {
	Chat(ctx context.Context, message string, enableSkills bool, enableMCP bool) (string, error)
	ChatStream(ctx context.Context, message string, enableSkills bool, enableMCP bool, onChunk func(context.Context, []byte) error) (string, error)
}

// SimpleChatAgent manages conversation history for a session
type SimpleChatAgent struct {
	llm           llms.Model
	messages      []llms.MessageContent
	mu            sync.RWMutex
	mcpClient     *mcpclient.Client
	mcpTools      []tools.Tool
	skills        []SkillInfo
	selectedSkill string // Currently selected skill name
	toolsEnabled  bool
	toolsLoading  bool // true when tools are being loaded asynchronously
	toolsLoaded   bool // true when tools have finished loading
}

// NewSimpleChatAgent creates a simple chat agent
func NewSimpleChatAgent(llm llms.Model, config configpkg.Config) *SimpleChatAgent {
	// Add system message
	systemMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextPart("You are a helpful AI assistant. Be concise and friendly.")},
	}

	agent := &SimpleChatAgent{
		llm:      llm,
		messages: []llms.MessageContent{systemMsg},
	}

	return agent
}

// InitializeToolsAsync asynchronously loads Skills and MCP tools in the background
// This prevents blocking server startup while tools are being loaded
func (a *SimpleChatAgent) InitializeToolsAsync() {
	// Mark as loading
	a.mu.Lock()
	a.toolsLoading = true
	a.toolsLoaded = false
	a.mu.Unlock()

	go func() {
		defer func() {
			// Mark as loaded regardless of success/failure to prevent blocking
			a.mu.Lock()
			a.toolsLoading = false
			a.toolsLoaded = true
			skillsCount := len(a.skills)
			mcpToolsCount := len(a.mcpTools)
			a.mu.Unlock()
			log.Printf("✓ Tools pre-warming complete: %d Skills, %d MCP tools loaded", skillsCount, mcpToolsCount)
		}()

		log.Println("Starting background tools initialization...")

		// Add recovery for any panics during tool loading
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic during tools initialization: %v", r)
			}
		}()

		// Load Skills
		skillsDir := os.Getenv("SKILLS_DIR")
		if skillsDir == "" {
			skillsDir = "../../testdata/skills"
		}

		if _, err := os.Stat(skillsDir); err == nil {
			packages, err := goskills.ParseSkillPackages(skillsDir)
			if err != nil {
				log.Printf("Failed to parse skills packages: %v", err)
			} else {
				a.mu.Lock()
				for _, skill := range packages {
					// Store skill info without converting to tools yet
					a.skills = append(a.skills, SkillInfo{
						Name:        skill.Meta.Name,
						Description: skill.Meta.Description,
						Package:     skill,
						Loaded:      false,
					})
				}
				a.toolsEnabled = true
				a.mu.Unlock()
				log.Printf("Loaded %d skills info", len(packages))

				// Pre-warm: Load tools for all skills
				log.Println("Pre-loading tools for all skills...")
				for i := range a.skills {
					skillName := a.skills[i].Name
					if _, err := a.loadSkillTools(skillName); err != nil {
						log.Printf("Failed to pre-load tools for skill '%s': %v", skillName, err)
					}
				}
				log.Printf("Pre-loaded tools for %d skills", len(a.skills))
			}
		} else {
			log.Printf("Skills directory not found at %s", skillsDir)
		}

		// Load MCP
		mcpConfigPath := os.Getenv("MCP_CONFIG_PATH")
		if mcpConfigPath == "" {
			mcpConfigPath = "../../testdata/mcp/mcp.json"
		}

		// Safely initialize MCP with error recovery
		if err := a.initializeMCP(mcpConfigPath); err != nil {
			log.Printf("MCP initialization failed (continuing without MCP): %v", err)
		}
	}()
}

// initializeMCP safely initializes MCP client with error recovery
func (a *SimpleChatAgent) initializeMCP(mcpConfigPath string) (err error) {
	// Add panic recovery to prevent crashes from MCP initialization
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during MCP initialization: %v", r)
			log.Printf("Recovered from MCP initialization panic: %v", r)
		}
	}()

	// Use a longer timeout for initialization as npx downloads may be slow
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Load MCP config
	config, err := mcpclient.LoadConfig(mcpConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	// Create MCP client with error handling
	client, err := mcpclient.NewClient(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Get tools from MCP with timeout
	toolsCtx, toolsCancel := context.WithTimeout(ctx, 30*time.Second)
	defer toolsCancel()

	tools, err := mcp.MCPToTools(toolsCtx, client)
	if err != nil {
		// Close client if tool loading fails
		if closeErr := a.closeMCPClient(client); closeErr != nil {
			log.Printf("Failed to close MCP client after error: %v", closeErr)
		}
		return fmt.Errorf("failed to get MCP tools: %w", err)
	}

	if len(tools) == 0 {
		log.Printf("No MCP tools found, closing client")
		if closeErr := a.closeMCPClient(client); closeErr != nil {
			log.Printf("Failed to close MCP client: %v", closeErr)
		}
		return nil
	}

	// Successfully initialized
	a.mu.Lock()
	a.mcpClient = client
	a.mcpTools = tools
	a.toolsEnabled = true
	a.mu.Unlock()
	log.Printf("Successfully loaded %d MCP tools", len(tools))

	return nil
}

// closeMCPClient safely closes an MCP client with panic recovery and timeout
func (a *SimpleChatAgent) closeMCPClient(client *mcpclient.Client) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during MCP client close: %v", r)
			log.Printf("Recovered from MCP client close panic: %v", r)
		}
	}()

	if client == nil {
		return nil
	}

	// Use a goroutine with timeout to prevent hanging on close
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic in close goroutine: %v", r)
			}
		}()
		done <- client.Close()
	}()

	// Wait for close with timeout
	select {
	case closeErr := <-done:
		if closeErr != nil {
			return fmt.Errorf("failed to close MCP client: %w", closeErr)
		}
		return nil
	case <-time.After(5 * time.Second):
		log.Printf("Warning: MCP client close timed out after 5 seconds")
		return fmt.Errorf("MCP client close timed out")
	}
}

// Close releases resources held by the agent
func (a *SimpleChatAgent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Printf("Closing agent and cleaning up resources...")

	if a.mcpClient != nil {
		log.Printf("Closing MCP client...")
		if err := a.closeMCPClient(a.mcpClient); err != nil {
			// Log error but don't return - we want to continue cleanup
			log.Printf("Error closing MCP client (continuing cleanup): %v", err)
		}
		a.mcpClient = nil
		a.mcpTools = nil
		log.Printf("MCP client closed and cleared")
	}

	return nil
}

// getToolsInfo returns a formatted string of available tools
func (a *SimpleChatAgent) getToolsInfo() string {
	if len(a.mcpTools) == 0 {
		return "No tools available."
	}

	var info strings.Builder
	for _, tool := range a.mcpTools {
		info.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}
	return info.String()
}

// GetAvailableTools returns the list of available skills and MCP tools
func (a *SimpleChatAgent) GetAvailableTools() []map[string]string {
	var tools []map[string]string

	// Add MCP tools
	for _, tool := range a.mcpTools {
		tools = append(tools, map[string]string{
			"name":        tool.Name(),
			"description": tool.Description(),
			"type":        "mcp",
		})
	}

	// Add skills (not loaded as tools yet)
	for _, skill := range a.skills {
		tools = append(tools, map[string]string{
			"name":        skill.Name,
			"description": skill.Description,
			"type":        "skill",
		})
	}

	return tools
}

// Chat sends a message and returns response
func (a *SimpleChatAgent) Chat(ctx context.Context, message string, enableSkills bool, enableMCP bool) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add user message
	userMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextPart(message)},
	}
	a.messages = append(a.messages, userMsg)

	toolUsed := false
	var toolResult string
	var toolName string

	if a.toolsEnabled {
		// Stage 1: Select skill if needed (only if user enables Skills)
		if enableSkills && len(a.skills) > 0 {
			selectedSkill, err := a.selectSkillForTask(ctx, message)
			if err != nil {
				log.Printf("Skill selection error: %v", err)
			} else if selectedSkill != "" {
				// Load tools for the selected skill
				skillTools, err := a.loadSkillTools(selectedSkill)
				if err != nil {
					log.Printf("Failed to load skill tools: %v", err)
				} else {
					a.selectedSkill = selectedSkill

					// Stage 2: Select specific tool from the skill
					tool, args, err := a.selectToolForTask(ctx, message, skillTools)
					if err != nil {
						log.Printf("Tool selection error: %v", err)
					} else if tool != nil {
						// Convert args to JSON string
						argsJSON, _ := json.Marshal(args)
						argsStr := string(argsJSON)
						if argsStr == "null" {
							argsStr = "{}"
						}

						// Call the tool
						result, err := (*tool).Call(ctx, argsStr)
						if err != nil {
							log.Printf("Tool %s call failed: %v", (*tool).Name(), err)
						} else {
							toolUsed = true
							toolResult = result
							toolName = (*tool).Name()
							log.Printf("Successfully used tool '%s' from skill '%s'", (*tool).Name(), selectedSkill)
						}
					}
				}
			}
		}

		// If no skill was selected, try MCP tools (only if user enables MCP)
		if !toolUsed && enableMCP && len(a.mcpTools) > 0 {
			tool, args, err := a.selectToolForTask(ctx, message, a.mcpTools)
			if err != nil {
				log.Printf("MCP tool selection error: %v", err)
			} else if tool != nil {
				// Convert args to JSON string
				argsJSON, _ := json.Marshal(args)
				argsStr := string(argsJSON)
				if argsStr == "null" {
					argsStr = "{}"
				}

				// Call the tool
				result, err := (*tool).Call(ctx, argsStr)
				if err != nil {
					log.Printf("MCP tool %s call failed: %v", (*tool).Name(), err)
				} else {
					toolUsed = true
					toolResult = result
					toolName = (*tool).Name()
					log.Printf("Successfully used MCP tool '%s'", (*tool).Name())
				}
			}
		}
	}

	// Add tool result to conversation if a tool was used
	if toolUsed && toolResult != "" {
		toolMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart(fmt.Sprintf("I used the '%s' tool to help with your request. Here's the result:\n\n%s", toolName, toolResult)),
			},
		}
		a.messages = append(a.messages, toolMsg)
	}

	// Call LLM with full history
	response, err := a.llm.GenerateContent(ctx, a.messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract response text
	var responseText string
	if response != nil && len(response.Choices) > 0 {
		responseText = response.Choices[0].Content
	}

	// Add assistant response to history
	assistantMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart(responseText)},
	}
	a.messages = append(a.messages, assistantMsg)

	return responseText, nil
}

// ChatStream sends a message and streams response
func (a *SimpleChatAgent) ChatStream(ctx context.Context, message string, enableSkills bool, enableMCP bool, onChunk func(context.Context, []byte) error) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Accumulator for the full response content (including tool logs)
	var fullResponseBuilder strings.Builder

	// Add user message
	userMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextPart(message)},
	}
	a.messages = append(a.messages, userMsg)

	toolUsed := false
	var toolResult string
	var toolName string

	if a.toolsEnabled {
		// Stage 1: Select skill if needed (only if user enables Skills)
		if enableSkills && len(a.skills) > 0 {
			selectedSkill, err := a.selectSkillForTask(ctx, message)
			if err != nil {
				log.Printf("Skill selection error: %v", err)
			} else if selectedSkill != "" {
				// Load tools for the selected skill
				skillTools, err := a.loadSkillTools(selectedSkill)
				if err != nil {
					log.Printf("Failed to load skill tools: %v", err)
				} else {
					a.selectedSkill = selectedSkill

					// Stage 2: Select specific tool from the skill
					tool, args, err := a.selectToolForTask(ctx, message, skillTools)
					if err != nil {
						log.Printf("Tool selection error: %v", err)
					} else if tool != nil {
						// Convert args to JSON string
						argsJSON, _ := json.MarshalIndent(args, "", "  ")
						argsStr := string(argsJSON)
						if argsStr == "null" {
							argsStr = "{}"
						}

						// Notify start of tool execution
						toolName = (*tool).Name()
						notifyStart := fmt.Sprintf("\n\n> 🛠️ Calling tool **%s**...\n\n", toolName)
						onChunk(ctx, []byte(notifyStart))
						fullResponseBuilder.WriteString(notifyStart)

						// Call the tool
						result, err := (*tool).Call(ctx, argsStr)

						// Notify end of tool execution
						if err != nil {
							log.Printf("Tool %s call failed: %v", toolName, err)
							notifyError := fmt.Sprintf("\n\n> ❌ Tool error: %v\n\n", err)
							onChunk(ctx, []byte(notifyError))
							fullResponseBuilder.WriteString(notifyError)
						} else {
							toolUsed = true
							toolResult = result
							log.Printf("Successfully used tool '%s' from skill '%s'", toolName, selectedSkill)

							// Format result in collapsible details
							notifyResult := fmt.Sprintf("\n\n<details>\n<summary>Tool Result: %s</summary>\n\n```\n%s\n```\n\n</details>\n\n", toolName, result)
							onChunk(ctx, []byte(notifyResult))
							fullResponseBuilder.WriteString(notifyResult)
						}
					}
				}
			}
		}

		// If no skill was selected, try MCP tools (only if user enables MCP)
		if !toolUsed && enableMCP && len(a.mcpTools) > 0 {
			tool, args, err := a.selectToolForTask(ctx, message, a.mcpTools)
			if err != nil {
				log.Printf("MCP tool selection error: %v", err)
			} else if tool != nil {
				// Convert args to JSON string
				argsJSON, _ := json.MarshalIndent(args, "", "  ")
				argsStr := string(argsJSON)
				if argsStr == "null" {
					argsStr = "{}"
				}

				// Notify start of tool execution
				toolName = (*tool).Name()
				notifyStart := fmt.Sprintf("\n\n> 🛠️ Calling tool **%s**...\n\n", toolName)
				onChunk(ctx, []byte(notifyStart))
				fullResponseBuilder.WriteString(notifyStart)

				// Call the tool
				result, err := (*tool).Call(ctx, argsStr)

				// Notify end of tool execution
				if err != nil {
					log.Printf("MCP tool %s call failed: %v", toolName, err)
					notifyError := fmt.Sprintf("\n\n> ❌ Tool error: %v\n\n", err)
					onChunk(ctx, []byte(notifyError))
					fullResponseBuilder.WriteString(notifyError)
				} else {
					toolUsed = true
					toolResult = result
					log.Printf("Successfully used MCP tool '%s'", toolName)

					// Format result in collapsible details
					notifyResult := fmt.Sprintf("\n\n<details>\n<summary>Tool Result: %s</summary>\n\n```\n%s\n```\n\n</details>\n\n", toolName, result)
					onChunk(ctx, []byte(notifyResult))
					fullResponseBuilder.WriteString(notifyResult)
				}
			}
		}
	}

	// Add tool result to conversation if a tool was used
	if toolUsed && toolResult != "" {
		toolMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart(fmt.Sprintf("I used the '%s' tool to help with your request. Here's the result:\n\n%s", toolName, toolResult)),
			},
		}
		a.messages = append(a.messages, toolMsg)
	}

	// Call LLM with full history and streaming
	response, err := a.llm.GenerateContent(ctx, a.messages, llms.WithStreamingFunc(onChunk))
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract response text
	var responseText string
	if response != nil && len(response.Choices) > 0 {
		responseText = response.Choices[0].Content
	}

	// Append LLM response to full response
	fullResponseBuilder.WriteString(responseText)
	fullResponse := fullResponseBuilder.String()

	// Add assistant response to history
	assistantMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart(fullResponse)},
	}
	a.messages = append(a.messages, assistantMsg)

	return fullResponse, nil
}

// getClientID generates a unique client ID based on IP and User-Agent
func getClientID(r *http.Request) string {
	// Get client IP
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = strings.Split(r.RemoteAddr, ":")[0]
	}

	// Get User-Agent
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "unknown"
	}

	// Create unique hash from IP + User-Agent
	h := md5.New()
	h.Write([]byte(clientIP + userAgent + "chat-salt"))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// ChatServer manages HTTP endpoints and chat agents
type ChatServer struct {
	maxHistory      int
	sessionDir      string
	agents          map[string]ChatAgent
	llm             llms.Model
	agentMu         sync.RWMutex
	port            string
	config          configpkg.Config
	sessionManagers map[string]*sessionpkg.SessionManager // clientID -> SessionManager
	smMu            sync.RWMutex
	requestSem      chan struct{} // Semaphore for controlling concurrent requests
	maxConcurrent   int           // Maximum number of concurrent requests

	// New components for enterprise features
	lifecycleManager *agentpkg.AgentLifecycleManager
	metricsCollector *monitoringpkg.MetricsCollector
	configManager    *configpkg.Manager
	healthChecker    *monitoringpkg.HealthChecker

	// Authentication components
	authService   *auth.AuthService
	jwtAuth       *middleware.AuthMiddleware
	authAPI       *api.AuthAPI
	staticHandler *api.StaticHandler
}

// NewChatServer creates a new chat server
func NewChatServer(sessionDir string, maxHistory int, port string, configPath string) (*ChatServer, error) {
	// Initialize configuration manager
	configManager := configpkg.NewManager(configpkg.Development)
	if _, err := os.Stat(configPath); err == nil {
		if err := configManager.Load(configPath); err != nil {
			log.Printf("Warning: Failed to load config from file: %v", err)
		}
	} else {
		log.Printf("Warning: Config file not found at %s", configPath)
	}
	config := configManager.Get()

	// Check API key and fallback to environment variable if not set
	if config.LLM.APIKey == "" {
		config.LLM.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	if config.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM API key not set in configuration or environment (OPENAI_API_KEY)")
	}

	// Check model and fallback to environment variable if not set
	if config.LLM.Model == "" {
		config.LLM.Model = os.Getenv("OPENAI_MODEL")
	}

	// Check BaseURL and fallback to environment variable if not set
	if config.LLM.BaseURL == "" {
		config.LLM.BaseURL = os.Getenv("OPENAI_API_BASE")
	}

	// Create OpenAI LLM (works with OpenAI-compatible APIs like Baidu)
	var llm llms.Model
	var err error

	if config.LLM.BaseURL != "" {
		llm, err = openai.New(
			openai.WithModel(config.LLM.Model),
			openai.WithToken(config.LLM.APIKey),
			openai.WithBaseURL(config.LLM.BaseURL),
		)
	} else {
		llm, err = openai.New(
			openai.WithModel(config.LLM.Model),
			openai.WithToken(config.LLM.APIKey),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	// Initialize monitoring components
	metricsCollector := monitoringpkg.NewMetricsCollector()
	healthChecker := monitoringpkg.NewHealthChecker()

	// Start metrics server if monitoring is enabled
	if config.Monitoring.Enabled {
		go func() {
			metricsServer := monitoringpkg.NewMetricsServer(metricsCollector, config.Monitoring.MetricsPort)
			log.Printf("🔧 Starting metrics server on port %d", config.Monitoring.MetricsPort)
			if err := metricsServer.Start(); err != nil {
				log.Printf("Failed to start metrics server: %v", err)
			}
		}()
	}

	// Initialize agent lifecycle manager
	lifecycleConfig := agentpkg.DefaultAgentLifecycleConfig()
	lifecycleConfig.MaxIdleTime = config.Agent.MaxIdleTime
	lifecycleConfig.HealthCheckInterval = config.Agent.HealthCheckInterval
	lifecycleConfig.MaxRetries = config.Agent.MaxRetries
	lifecycleConfig.RetryDelay = config.Agent.RetryDelay
	lifecycleManager := agentpkg.NewAgentLifecycleManager(lifecycleConfig)

	// Set lifecycle event handler
	lifecycleManager.SetEventHandler(func(event agentpkg.LifecycleEvent) {
		log.Printf("Agent Lifecycle Event: %s - %s - %s", event.EventType, event.State, event.Message)
		if event.Error != nil {
			log.Printf("Agent Error: %v", event.Error)
		}
	})

	// Register health checks
	healthChecker.RegisterCheck("lifecycle_manager", func(ctx context.Context) error {
		state := lifecycleManager.GetState()
		if state == agentpkg.StateError {
			return fmt.Errorf("agent is in error state")
		}
		return nil
	})

	healthChecker.RegisterCheck("llm_connection", func(ctx context.Context) error {
		// Simple check - in a real implementation, you might test the LLM connection
		if llm == nil {
			return fmt.Errorf("LLM is not initialized")
		}
		return nil
	})

	// Initialize authentication components
	jwtAuth := middleware.NewAuthMiddleware(
		config.Security.JWTSecret,
		config.Security.SessionTimeout,
		config.Security.SessionTimeout*7, // 7x longer for refresh tokens
	)

	authService := auth.NewAuthService(
		config.Security.JWTSecret,
		config.Security.SessionTimeout,
		config.Security.SessionTimeout*7,
	)

	// Create demo users for testing
	if err := authService.CreateDemoUsers(); err != nil {
		log.Printf("Warning: Failed to create demo users: %v", err)
	}

	authAPI := api.NewAuthAPI(authService, jwtAuth)
	staticHandler := api.NewStaticHandler(authAPI)

	// Set default max concurrent requests from configuration
	maxConcurrent := config.Agent.MaxConcurrent

	server := &ChatServer{
		authService:      authService,
		jwtAuth:          jwtAuth,
		authAPI:          authAPI,
		staticHandler:    staticHandler,
		maxHistory:       maxHistory,
		sessionDir:       sessionDir,
		agents:           make(map[string]ChatAgent),
		llm:              llm,
		port:             port,
		config:           *config,
		sessionManagers:  make(map[string]*sessionpkg.SessionManager),
		requestSem:       make(chan struct{}, maxConcurrent),
		maxConcurrent:    maxConcurrent,
		lifecycleManager: lifecycleManager,
		metricsCollector: metricsCollector,
		configManager:    configManager,
		healthChecker:    healthChecker,
	}

	// Initialize lifecycle manager
	if err := lifecycleManager.SetState(agentpkg.StateInitializing, "Server starting", nil); err != nil {
		log.Printf("Warning: Failed to set initial lifecycle state: %v", err)
	}

	return server, nil
}

// getSessionManager gets or creates a SessionManager for a specific client
func (cs *ChatServer) GetSessionManager(clientID string) *sessionpkg.SessionManager {
	cs.smMu.Lock()
	defer cs.smMu.Unlock()

	sm, exists := cs.sessionManagers[clientID]
	if !exists {
		clientSessionDir := fmt.Sprintf("%s/clients/%s", cs.sessionDir, clientID)
		store := sessionpkg.NewFileSessionStore(clientSessionDir)
		sm = sessionpkg.NewSessionManager(store, cs.maxHistory)
		cs.sessionManagers[clientID] = sm
	}
	return sm
}

// getOrCreateAgent gets an existing agent or creates a new one for a session
func (cs *ChatServer) GetOrCreateAgent(sessionID string) (ChatAgent, error) {
	cs.agentMu.RLock()
	agent, exists := cs.agents[sessionID]
	cs.agentMu.RUnlock()

	if exists {
		return agent, nil
	}

	// Create new agent
	cs.agentMu.Lock()
	defer cs.agentMu.Unlock()

	// Double-check after acquiring write lock
	if agent, exists := cs.agents[sessionID]; exists {
		return agent, nil
	}

	// Try to use the warmup agent configuration but create a new instance
	if warmupAgent, exists := cs.agents["__warmup__"]; exists {
		log.Printf("Using pre-warmed agent configuration for session %s", sessionID)
		// Don't reuse the warmup agent instance to avoid state sharing issues
		// Instead, create a new agent with the same configuration
		// Clean up the warmup agent asynchronously
		go func() {
			if warmupAgentCloser, ok := warmupAgent.(interface{ Close() error }); ok {
				warmupAgentCloser.Close()
			}
		}()
		delete(cs.agents, "__warmup__")
	}

	// Create a new agent instance for this session
	agent = NewSimpleChatAgent(cs.llm, cs.config)
	cs.agents[sessionID] = agent

	// Initialize tools asynchronously to avoid blocking
	agent.(*SimpleChatAgent).InitializeToolsAsync()

	return agent, nil
}

// GetWarmupAgent returns the warmup agent for reuse
func (cs *ChatServer) GetWarmupAgent() *SimpleChatAgent {
	cs.agentMu.Lock()
	defer cs.agentMu.Unlock()

	if warmupAgent, exists := cs.agents["__warmup__"]; exists {
		return warmupAgent.(*SimpleChatAgent)
	}
	return nil
}

// SetWarmupAgent stores a warmup agent for reuse
func (cs *ChatServer) SetWarmupAgent(agent *SimpleChatAgent) {
	cs.agentMu.Lock()
	defer cs.agentMu.Unlock()
	cs.agents["__warmup__"] = agent
}

// GetLLM returns the LLM instance
func (cs *ChatServer) GetLLM() llms.Model {
	return cs.llm
}

// GetConfig returns the server config
func (cs *ChatServer) GetConfig() *configpkg.Config {
	return &cs.config
}

// GetLifecycleManager returns the agent lifecycle manager
func (cs *ChatServer) GetLifecycleManager() *agentpkg.AgentLifecycleManager {
	return cs.lifecycleManager
}

// GetMetricsCollector returns the metrics collector
func (cs *ChatServer) GetMetricsCollector() *monitoringpkg.MetricsCollector {
	return cs.metricsCollector
}

// GetHealthChecker returns the health checker
func (cs *ChatServer) GetHealthChecker() *monitoringpkg.HealthChecker {
	return cs.healthChecker
}

// GetConfigManager returns the config manager
func (cs *ChatServer) GetConfigManager() *configpkg.Manager {
	return cs.configManager
}

// acquireRequest acquires a request slot or returns an error if limit exceeded
func (cs *ChatServer) acquireRequest() error {
	select {
	case cs.requestSem <- struct{}{}:
		return nil
	default:
		return fmt.Errorf("server is busy: maximum concurrent requests (%d) exceeded", cs.maxConcurrent)
	}
}

// releaseRequest releases a request slot
func (cs *ChatServer) releaseRequest() {
	select {
	case <-cs.requestSem:
	default:
		// This should not happen, but handle gracefully
		log.Printf("Warning: attempt to release request when semaphore is empty")
	}
}

// HandleIndex serves the main HTML page
func (cs *ChatServer) HandleIndex(w http.ResponseWriter, r *http.Request, staticFS fs.FS) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Read index.html from embedded filesystem
	data, err := fs.ReadFile(staticFS, "static/index.html")
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		log.Printf("Failed to read index.html: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// HandleNewSession creates a new chat session
func (cs *ChatServer) HandleNewSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.GetSessionManager(clientID)
	session := sm.CreateSession()

	// Set client ID cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "client_id",
		Value:    clientID,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": session.ID,
		"client_id":  clientID,
	})
}

// HandleListSessions returns all active sessions for the client
func (cs *ChatServer) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.GetSessionManager(clientID)
	sessions := sm.ListSessions()

	type SessionInfo struct {
		ID           string    `json:"id"`
		Title        string    `json:"title"`
		MessageCount int       `json:"message_count"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
	}

	sessionInfos := make([]SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		// Get the first user message as title
		title := "新会话"
		for _, msg := range session.Messages {
			if msg.Role == "user" {
				// Convert string to rune slice to properly handle UTF-8 characters
				runes := []rune(msg.Content)
				if len(runes) > 20 {
					title = string(runes[:20]) + "..."
				} else {
					title = msg.Content
				}
				break
			}
		}

		sessionInfos = append(sessionInfos, SessionInfo{
			ID:           session.ID,
			Title:        title,
			MessageCount: len(session.Messages),
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessionInfos)
}

// HandleDeleteSession deletes a session
func (cs *ChatServer) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.GetSessionManager(clientID)

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// Close and delete agent
	cs.agentMu.Lock()
	if agent, exists := cs.agents[sessionID]; exists {
		// Close agent if it implements Close method
		log.Printf("Closing agent for deleted session %s", sessionID)
		if simpleAgent, ok := agent.(*SimpleChatAgent); ok {
			// Use a goroutine with timeout to prevent blocking
			done := make(chan error, 1)
			go func() {
				done <- simpleAgent.Close()
			}()

			// Wait for close with timeout
			select {
			case err := <-done:
				if err != nil {
					log.Printf("Error closing agent for session %s: %v", sessionID, err)
				}
			case <-time.After(10 * time.Second):
				log.Printf("Warning: Agent close for session %s timed out", sessionID)
			}
		}
		delete(cs.agents, sessionID)
		log.Printf("Agent for session %s deleted", sessionID)
	}
	cs.agentMu.Unlock()

	// Delete session
	err := sm.DeleteSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGetHistory retrieves chat history for a session
func (cs *ChatServer) HandleGetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.GetSessionManager(clientID)

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	sessionID = strings.TrimSuffix(sessionID, "/history")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	messages, err := sm.GetMessages(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// HandleChat handles chat message requests
func (cs *ChatServer) HandleChat(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		cs.metricsCollector.RecordHTTPRequest(r.Method, r.URL.Path, "405", 0, 0, 0)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Acquire request slot for concurrency control
	if err := cs.acquireRequest(); err != nil {
		cs.metricsCollector.RecordHTTPRequest(r.Method, r.URL.Path, "429", 0, 0, 0)
		log.Printf("Request rejected: %v", err)
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}
	defer cs.releaseRequest()

	var req struct {
		SessionID    string `json:"session_id"`
		Message      string `json:"message"`
		UserSettings struct {
			EnableSkills bool `json:"enable_skills"`
			EnableMCP    bool `json:"enable_mcp"`
		} `json:"user_settings"`
		Stream bool `json:"stream"` // New field for streaming request
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" || req.Message == "" {
		http.Error(w, "session_id and message are required", http.StatusBadRequest)
		return
	}

	clientID := getClientID(r)
	sm := cs.GetSessionManager(clientID)

	log.Printf("Chat request for session %s: %s (stream: %v)", req.SessionID, req.Message, req.Stream)

	// Verify session exists
	_, err := sm.GetSession(req.SessionID)
	if err != nil {
		log.Printf("Session not found: %s", req.SessionID)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get or create agent for this session
	agent, err := cs.GetOrCreateAgent(req.SessionID)
	if err != nil {
		log.Printf("Failed to create agent: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Add user message to history
	_, _ = sm.AddMessage(req.SessionID, "user", req.Message)

	// Use user settings directly
	enableSkills := req.UserSettings.EnableSkills
	enableMCP := req.UserSettings.EnableMCP

	log.Printf("Tool settings for session %s - Skills: %v, MCP: %v",
		req.SessionID, enableSkills, enableMCP)

	// Record metrics
	duration := time.Since(startTime)
	requestSize := int64(r.ContentLength)
	cs.metricsCollector.RecordHTTPRequest(r.Method, r.URL.Path, "200", duration, requestSize, 0)

	if req.Stream {
		// Handle streaming response
		cs.HandleChatStream(w, r, agent, req.SessionID, req.Message, enableSkills, enableMCP)
	} else {
		// Handle non-streaming response (original behavior)
		cs.HandleChatNonStream(w, r, agent, req.SessionID, req.Message, enableSkills, enableMCP)
	}

	// Record agent session event
	cs.metricsCollector.RecordAgentSession("chat_request")
}

// HandleChatNonStream handles non-streaming chat responses (original behavior)
func (cs *ChatServer) HandleChatNonStream(w http.ResponseWriter, r *http.Request, agent ChatAgent, sessionID, message string, enableSkills, enableMCP bool) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	response, err := agent.Chat(ctx, message, enableSkills, enableMCP)
	if err != nil {
		log.Printf("Chat error for session %s: %v", sessionID, err)
		cs.metricsCollector.RecordAgentError(sessionID, "chat_error")
		http.Error(w, fmt.Sprintf("Chat failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Chat response for session %s: %s", sessionID, response)

	// Record agent metrics
	cs.metricsCollector.RecordAgentMessage(sessionID, "assistant")
	cs.metricsCollector.RecordAgentTokenUsage(sessionID, "response", int64(len(response)))

	// Add assistant response to history
	clientID := getClientID(r)
	sm := cs.GetSessionManager(clientID)
	msgID, _ := sm.AddMessage(sessionID, "assistant", response)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response":   response,
		"message_id": msgID,
	})
}

// HandleChatStream handles streaming chat responses using SSE
func (cs *ChatServer) HandleChatStream(w http.ResponseWriter, r *http.Request, agent ChatAgent, sessionID, message string, enableSkills, enableMCP bool) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Get a flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("Streaming not supported")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	clientID := getClientID(r)
	sm := cs.GetSessionManager(clientID)

	// Send initial event
	fmt.Fprintf(w, "event: start\ndata: {\"type\": \"start\"}\n\n")
	flusher.Flush()

	// Define streaming callback
	streamFunc := func(ctx context.Context, chunk []byte) error {
		data := map[string]any{
			"type":  "chunk",
			"chunk": string(chunk),
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", jsonData)
		flusher.Flush()
		return nil
	}

	// Get the full response from agent while streaming
	response, err := agent.ChatStream(ctx, message, enableSkills, enableMCP, streamFunc)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"type\": \"error\", \"error\": %q}\n\n", err.Error())
		flusher.Flush()
		return
	}

	// Save the complete response to history
	msgID, _ := sm.AddMessage(sessionID, "assistant", response)

	// Send end event
	endData := map[string]any{
		"type":       "end",
		"message":    response,
		"message_id": msgID,
	}
	jsonEndData, _ := json.Marshal(endData)
	fmt.Fprintf(w, "event: end\ndata: %s\n\n", jsonEndData)
	flusher.Flush()
}

// HandleGetClientID returns the client ID for the current user
func (cs *ChatServer) HandleGetClientID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)

	// Set client ID cookie if not already set
	_, err := r.Cookie("client_id")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "client_id",
			Value:    clientID,
			Path:     "/",
			MaxAge:   86400 * 30, // 30 days
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"client_id": clientID,
	})
}

// HandleMCPTools returns the list of available MCP tools
func (cs *ChatServer) HandleMCPTools(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// Get or create agent for this session
	agent, err := cs.GetOrCreateAgent(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Cast to SimpleChatAgent to access MCP methods
	simpleAgent, ok := agent.(*SimpleChatAgent)
	if !ok {
		http.Error(w, "Agent does not support MCP", http.StatusInternalServerError)
		return
	}

	tools := simpleAgent.GetAvailableTools()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"tools":   tools,
		"enabled": simpleAgent.toolsEnabled,
	})
}

// HandleToolsHierarchical returns tools in a hierarchical structure
func (cs *ChatServer) HandleToolsHierarchical(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// Get or create agent for this session
	agent, err := cs.GetOrCreateAgent(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Cast to SimpleChatAgent
	simpleAgent, ok := agent.(*SimpleChatAgent)
	if !ok {
		http.Error(w, "Agent does not support tools", http.StatusInternalServerError)
		return
	}

	// Prepare hierarchical data
	var result struct {
		Skills       []map[string]any `json:"skills"`
		MCPTools     []map[string]any `json:"mcp_tools"`
		Enabled      bool             `json:"enabled"`
		ToolsLoading bool             `json:"tools_loading"`
		ToolsLoaded  bool             `json:"tools_loaded"`
	}

	// Lock for reading skills and MCP tools
	simpleAgent.mu.RLock()
	result.Enabled = simpleAgent.toolsEnabled
	result.ToolsLoading = simpleAgent.toolsLoading
	result.ToolsLoaded = simpleAgent.toolsLoaded
	skills := make([]SkillInfo, len(simpleAgent.skills))
	copy(skills, simpleAgent.skills)
	mcpTools := make([]tools.Tool, len(simpleAgent.mcpTools))
	copy(mcpTools, simpleAgent.mcpTools)
	simpleAgent.mu.RUnlock()

	// Add skills with their tools
	for _, skill := range skills {
		skillData := map[string]any{
			"name":        skill.Name,
			"description": skill.Description,
			"tools":       []map[string]any{},
		}

		// Get tools for this skill if already loaded
		if skill.Loaded && len(skill.Tools) > 0 {
			for _, tool := range skill.Tools {
				skillData["tools"] = append(skillData["tools"].([]map[string]any), map[string]any{
					"name":        tool.Name(),
					"description": tool.Description(),
				})
			}
		} else {
			// Load tools on demand
			if tools, err := simpleAgent.loadSkillTools(skill.Name); err == nil {
				for _, tool := range tools {
					skillData["tools"] = append(skillData["tools"].([]map[string]any), map[string]any{
						"name":        tool.Name(),
						"description": tool.Description(),
					})
				}
			}
		}

		result.Skills = append(result.Skills, skillData)
	}

	// Add MCP tools (group them by category if possible, or list them individually)
	mcpGroups := make(map[string][]map[string]any)
	for _, tool := range mcpTools {
		toolName := tool.Name()
		desc := tool.Description()

		// Try to extract category from tool name (e.g., "puppeteer__puppeteer_navigate" -> "Puppeteer")
		parts := strings.Split(toolName, "__")
		var category string
		if len(parts) >= 2 {
			// Convert first letter to uppercase
			category = strings.ToUpper(parts[0][:1]) + strings.ToLower(parts[0][1:])
		} else {
			category = "Other"
		}

		mcpGroups[category] = append(mcpGroups[category], map[string]any{
			"name":        toolName,
			"description": desc,
		})
	}

	// Convert groups to array
	for category, tools := range mcpGroups {
		result.MCPTools = append(result.MCPTools, map[string]any{
			"category":    category,
			"description": fmt.Sprintf("%s tools", category),
			"tools":       tools,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleFeedback handles message feedback (like/dislike)
func (cs *ChatServer) HandleFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		MessageID string `json:"message_id"`
		Feedback  string `json:"feedback"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	clientID := getClientID(r)
	sm := cs.GetSessionManager(clientID)

	err := sm.UpdateMessageFeedback(req.SessionID, req.MessageID, req.Feedback)
	if err != nil {
		log.Printf("Failed to update feedback: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleConfig returns the chat configuration
func (cs *ChatServer) HandleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"chatTitle":      "聊天智能体",
		"appLogo":        "/static/images/logo.png",
		"enableFeedback": cs.config.Features.FeedbackEnabled,
		"environment":    "development", // TODO: Get from config manager
		"llmModel":       cs.config.LLM.Model,
		"version":        "1.0.0",
	})
}

// Close gracefully shuts down the server and cleans up all resources
func (cs *ChatServer) Close() error {
	log.Printf("Shutting down chat server...")

	cs.agentMu.Lock()
	defer cs.agentMu.Unlock()

	// Close all agents with error collection
	var closeErrors []error
	for sessionID, agent := range cs.agents {
		log.Printf("Closing agent for session %s", sessionID)
		if simpleAgent, ok := agent.(*SimpleChatAgent); ok {
			if err := simpleAgent.Close(); err != nil {
				log.Printf("Error closing agent for session %s: %v", sessionID, err)
				closeErrors = append(closeErrors, fmt.Errorf("session %s: %w", sessionID, err))
			}
		}
	}

	// Clear agents map
	cs.agents = make(map[string]ChatAgent)

	if len(closeErrors) > 0 {
		log.Printf("Chat server shutdown completed with %d errors", len(closeErrors))
		// Return first error but log all
		return closeErrors[0]
	}

	log.Printf("Chat server shutdown complete")
	return nil
}

// Start starts the HTTP server
func (cs *ChatServer) Start(staticFS fs.FS) error {
	// Create a new ServeMux for better route handling
	mux := http.NewServeMux()

	// Authentication routes (public)
	mux.HandleFunc("/login", cs.authAPI.HandleLoginPage)
	mux.HandleFunc("/register", cs.authAPI.HandleRegisterPage)
	mux.HandleFunc("/api/auth/login", cs.authAPI.HandleLogin)
	mux.HandleFunc("/api/auth/register", cs.authAPI.HandleRegister)
	mux.HandleFunc("/api/auth/refresh", cs.authAPI.HandleRefresh)
	mux.HandleFunc("/api/auth/logout", cs.authAPI.HandleLogout)

	// Public endpoints
	mux.HandleFunc("/health", cs.HandleHealth)
	mux.HandleFunc("/ready", cs.HandleReady)
	mux.HandleFunc("/info", cs.HandleInfo)
	mux.HandleFunc("/api/config", cs.HandleConfig)

	// Main app route - authenticate first, then serve original index.html
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Check if user is authenticated
			token := r.Header.Get("Authorization")
			if token == "" {
				// Check for token in cookie (for browser access)
				cookie, err := r.Cookie("access_token")
				if err == nil {
					token = "Bearer " + cookie.Value
				}
			}

			if token != "" {
				// Validate token
				if strings.HasPrefix(token, "Bearer ") {
					tokenStr := strings.TrimPrefix(token, "Bearer ")
					if _, err := cs.jwtAuth.ValidateToken(tokenStr); err == nil {
						// User is authenticated, serve original index.html
						cs.HandleIndex(w, r, staticFS)
						return
					}
				}
			}

			// User is not authenticated, redirect to login page
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		cs.HandleIndex(w, r, staticFS)
	})

	// Protected routes (require authentication)
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/api/client-id", cs.HandleGetClientID)
	protectedMux.HandleFunc("/api/auth/me", cs.authAPI.HandleGetCurrentUser)
	protectedMux.HandleFunc("/api/sessions/new", cs.HandleNewSession)
	protectedMux.HandleFunc("/api/sessions", cs.HandleListSessions)
	protectedMux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/history") {
			cs.HandleGetHistory(w, r)
		} else if r.Method == http.MethodDelete {
			cs.HandleDeleteSession(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	protectedMux.HandleFunc("/api/chat", cs.HandleChat)
	protectedMux.HandleFunc("/api/feedback", cs.HandleFeedback)
	protectedMux.HandleFunc("/api/mcp/tools", cs.HandleMCPTools)
	protectedMux.HandleFunc("/api/tools/hierarchical", cs.HandleToolsHierarchical)
	protectedMux.HandleFunc("/metrics", cs.HandleMetrics)

	// Apply authentication middleware to protected routes
	mux.Handle("/api/", cs.jwtAuth.Middleware(protectedMux))

	// Serve static files from embedded filesystem
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem: %w", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))))

	addr := ":" + cs.port
	log.Printf("🌐 HTTP server listening on http://localhost%s", addr)
	log.Printf("🔐 Authentication enabled - visit /login to sign in")
	return http.ListenAndServe(addr, mux)
}

// getSkillsOverview returns a formatted string of available skills (name and description only)
func (a *SimpleChatAgent) getSkillsOverview() string {
	if len(a.skills) == 0 {
		return ""
	}

	var info strings.Builder
	info.WriteString("Available Skills:\n\n")

	for _, skill := range a.skills {
		info.WriteString(fmt.Sprintf("- %s: %s\n", skill.Name, skill.Description))
	}

	return info.String()
}

// loadSkillTools loads and caches tools for a specific skill
func (a *SimpleChatAgent) loadSkillTools(skillName string) ([]tools.Tool, error) {
	// Find the skill
	for i := range a.skills {
		if strings.EqualFold(a.skills[i].Name, skillName) {
			if !a.skills[i].Loaded {
				// Convert skill to tools
				skillTools, err := adaptergoskills.SkillsToTools(a.skills[i].Package)
				if err != nil {
					return nil, fmt.Errorf("failed to convert skill '%s' to tools: %w", skillName, err)
				}
				a.skills[i].Tools = skillTools
				a.skills[i].Loaded = true
				log.Printf("Loaded %d tools from skill '%s'", len(skillTools), skillName)
			}
			return a.skills[i].Tools, nil
		}
	}
	return nil, fmt.Errorf("skill '%s' not found", skillName)
}

// selectSkillForTask uses LLM to determine which skill (if any) should be used for the task
func (a *SimpleChatAgent) selectSkillForTask(ctx context.Context, message string) (string, error) {
	if len(a.skills) == 0 {
		return "", nil // No skills available
	}

	skillsOverview := a.getSkillsOverview()

	skillPrompt := fmt.Sprintf(`Based on the user's message, determine if any of the available skills should be used to help with this task.

%s

User message: %s

Respond with a JSON object:
- If no skill is needed: {"use_skill": false, "reason": "reason why no skill is needed"}
- If a skill is needed: {"use_skill": true, "skill_name": "exact skill name", "reason": "why this skill is appropriate"}

IMPORTANT:
- Return ONLY valid JSON
- Do NOT use markdown code fences
- Do NOT use `+"```json"+` wrapper
- Choose the skill that best matches the user's needs`, skillsOverview, message)

	// Create LLM call for skill selection
	skillMsg := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant that selects appropriate skills for tasks. Respond only with valid JSON.")}},
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextPart(skillPrompt)}},
	}

	response, err := a.llm.GenerateContent(ctx, skillMsg)
	if err != nil {
		return "", fmt.Errorf("LLM call failed for skill selection: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	decision := response.Choices[0].Content
	log.Printf("Skill selection decision: %s", decision)

	// Clean up the decision
	cleanDecision := strings.TrimSpace(decision)
	if strings.HasPrefix(cleanDecision, "```json") {
		cleanDecision = strings.TrimPrefix(cleanDecision, "```json")
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	} else if strings.HasPrefix(cleanDecision, "```") {
		cleanDecision = strings.TrimPrefix(cleanDecision, "```")
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	}

	// Parse the decision
	var skillDecision struct {
		UseSkill  bool   `json:"use_skill"`
		SkillName string `json:"skill_name"`
		Reason    string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(cleanDecision), &skillDecision); err != nil {
		return "", fmt.Errorf("failed to parse skill decision: %w", err)
	}

	if skillDecision.UseSkill {
		log.Printf("Selected skill '%s' because: %s", skillDecision.SkillName, skillDecision.Reason)
		return skillDecision.SkillName, nil
	}

	log.Printf("No skill selected: %s", skillDecision.Reason)
	return "", nil
}

// selectToolForTask uses LLM to determine which tool should be used
func (a *SimpleChatAgent) selectToolForTask(ctx context.Context, message string, availableTools []tools.Tool) (*tools.Tool, map[string]any, error) {
	if len(availableTools) == 0 {
		return nil, nil, nil // No tools available
	}

	// Build tools info
	var toolsInfo strings.Builder
	for _, tool := range availableTools {
		toolsInfo.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	toolPrompt := fmt.Sprintf(`Based on the user's message, determine which tool should be used.

Available tools:
%s

User message: %s

Respond with a JSON object:
- If no tool is needed: {"use_tool": false, "reason": "reason why no tool is needed"}
- If a tool is needed: {"use_tool": true, "tool_name": "exact tool name", "args": {parameter: "value"}, "reason": "why this tool is appropriate"}

IMPORTANT:
- Return ONLY valid JSON
- Do NOT use markdown code fences
- Do NOT use `+"```json"+` wrapper
- Select the tool that can best accomplish the user's request`, toolsInfo.String(), message)

	// Create LLM call for tool selection
	toolMsg := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant that selects appropriate tools for tasks. Respond only with valid JSON.")}},
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextPart(toolPrompt)}},
	}

	response, err := a.llm.GenerateContent(ctx, toolMsg)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM call failed for tool selection: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, nil, fmt.Errorf("no response from LLM")
	}

	decision := response.Choices[0].Content
	log.Printf("Tool selection decision: %s", decision)

	// Clean up the decision
	cleanDecision := strings.TrimSpace(decision)
	if strings.HasPrefix(cleanDecision, "```json") {
		cleanDecision = strings.TrimPrefix(cleanDecision, "```json")
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	} else if strings.HasPrefix(cleanDecision, "```") {
		cleanDecision = strings.TrimPrefix(cleanDecision, "```")
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	}

	// Parse the decision
	var toolDecision struct {
		UseTool  bool           `json:"use_tool"`
		ToolName string         `json:"tool_name"`
		Args     map[string]any `json:"args"`
		Reason   string         `json:"reason"`
	}

	if err := json.Unmarshal([]byte(cleanDecision), &toolDecision); err != nil {
		return nil, nil, fmt.Errorf("failed to parse tool decision: %w", err)
	}

	if toolDecision.UseTool {
		// Find the selected tool
		for _, tool := range availableTools {
			if strings.EqualFold(tool.Name(), toolDecision.ToolName) {
				log.Printf("Selected tool '%s' because: %s", toolDecision.ToolName, toolDecision.Reason)
				return &tool, toolDecision.Args, nil
			}
		}
		return nil, nil, fmt.Errorf("tool '%s' not found in available tools", toolDecision.ToolName)
	}

	log.Printf("No tool selected: %s", toolDecision.Reason)
	return nil, nil, nil
}

// HandleHealth handles health check requests
func (s *ChatServer) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Perform health checks using the health checker
	if s.healthChecker != nil {
		results := s.healthChecker.CheckHealth(ctx)

		// Check if any check failed
		allHealthy := true
		for _, status := range results {
			if status.Status == "unhealthy" {
				allHealthy = false
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if allHealthy {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"timestamp": time.Now().UTC(),
				"checks":    results,
			})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "unhealthy",
				"timestamp": time.Now().UTC(),
				"checks":    results,
			})
		}
	} else {
		// Fallback health check
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
		})
	}
}

// HandleMetrics handles metrics requests
func (s *ChatServer) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.metricsCollector != nil {
		s.metricsCollector.UpdateSystemMetrics()

		// Redirect to the actual metrics server
		if s.config.Monitoring.Enabled {
			http.Redirect(w, r, fmt.Sprintf("http://localhost:%d/metrics", s.config.Monitoring.MetricsPort), http.StatusTemporaryRedirect)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "Metrics collection not enabled",
	})
}

// HandleReady handles readiness probe requests
func (s *ChatServer) HandleReady(w http.ResponseWriter, r *http.Request) {
	// Check if the server is ready to handle requests
	ctx := r.Context()

	// Perform basic readiness checks
	if s.healthChecker != nil {
		results := s.healthChecker.CheckHealth(ctx)

		// Consider the service ready if at least one health check passes
		ready := false
		for _, status := range results {
			if status.Status == "healthy" {
				ready = true
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if ready {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "ready",
				"timestamp": time.Now().UTC(),
			})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "not ready",
				"timestamp": time.Now().UTC(),
			})
		}
	} else {
		// Default to ready if no health checker is configured
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ready",
			"timestamp": time.Now().UTC(),
		})
	}
}

// HandleInfo handles server info requests
func (s *ChatServer) HandleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service":     "LangChat Agent",
		"version":     "1.0.0",
		"environment": "development", // TODO: Get from config manager
		"timestamp":   time.Now().UTC(),
		"features": map[string]interface{}{
			"agent_management": s.lifecycleManager != nil,
			"monitoring":       s.metricsCollector != nil,
			"health_checks":    s.healthChecker != nil,
			"tools":            false, // TODO: Implement tools field
			"skills":           false, // TODO: Implement skills field
		},
	}

	// Add agent statistics if lifecycle manager is available
	if s.lifecycleManager != nil {
		agentMetrics := s.lifecycleManager.GetMetrics()
		info["agent_stats"] = agentMetrics
	}

	// Add configuration summary
	info["config"] = map[string]interface{}{
		"server": map[string]interface{}{
			"host": s.config.Server.Host,
			"port": s.config.Server.Port,
		},
		"agent": map[string]interface{}{
			"max_concurrent": s.config.Agent.MaxConcurrent,
			"max_idle_time":  s.config.Agent.MaxIdleTime,
		},
		"monitoring": map[string]interface{}{
			"enabled":      s.config.Monitoring.Enabled,
			"metrics_port": s.config.Monitoring.MetricsPort,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)
}
