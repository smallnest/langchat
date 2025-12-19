package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	"github.com/fsnotify/fsnotify"
)

// Environment represents the deployment environment
type Environment string

const (
	Development Environment = "development"
	Testing     Environment = "testing"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// Config represents the complete application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server" yaml:"server"`

	// Agent configuration
	Agent AgentConfig `json:"agent" yaml:"agent"`

	// LLM configuration
	LLM LLMConfig `json:"llm" yaml:"llm"`

	// Database configuration
	Database DatabaseConfig `json:"database" yaml:"database"`

	// Security configuration
	Security SecurityConfig `json:"security" yaml:"security"`

	// Monitoring configuration
	Monitoring MonitoringConfig `json:"monitoring" yaml:"monitoring"`

	// Logging configuration
	Logging LoggingConfig `json:"logging" yaml:"logging"`

	// Cache configuration
	Cache CacheConfig `json:"cache" yaml:"cache"`

	// Features configuration
	Features FeaturesConfig `json:"features" yaml:"features"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host" env:"SERVER_HOST" default:"localhost"`
	Port         int           `json:"port" yaml:"port" env:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" env:"SERVER_READ_TIMEOUT" default:"30s"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT" default:"30s"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout" env:"SERVER_IDLE_TIMEOUT" default:"120s"`
	MaxConns     int           `json:"max_conns" yaml:"max_conns" env:"SERVER_MAX_CONNS" default:"1000"`
}

// AgentConfig holds agent-related configuration
type AgentConfig struct {
	MaxConcurrent    int           `json:"max_concurrent" yaml:"max_concurrent" env:"AGENT_MAX_CONCURRENT" default:"50"`
	MaxIdleTime      time.Duration `json:"max_idle_time" yaml:"max_idle_time" env:"AGENT_MAX_IDLE_TIME" default:"30m"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval" env:"AGENT_HEALTH_CHECK_INTERVAL" default:"30s"`
	MaxRetries       int           `json:"max_retries" yaml:"max_retries" env:"AGENT_MAX_RETRIES" default:"3"`
	RetryDelay       time.Duration `json:"retry_delay" yaml:"retry_delay" env:"AGENT_RETRY_DELAY" default:"5s"`
	SessionTimeout   time.Duration `json:"session_timeout" yaml:"session_timeout" env:"AGENT_SESSION_TIMEOUT" default:"60m"`
	MaxHistory       int           `json:"max_history" yaml:"max_history" env:"AGENT_MAX_HISTORY" default:"100"`
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider     string `json:"provider" yaml:"provider" env:"LLM_PROVIDER" default:"openai"`
	Model        string `json:"model" yaml:"model" env:"LLM_MODEL" default:"gpt-4"`
	APIKey       string `json:"api_key" yaml:"api_key" env:"LLM_API_KEY"`
	BaseURL      string `json:"base_url" yaml:"base_url" env:"LLM_BASE_URL"`
	Temperature  float64 `json:"temperature" yaml:"temperature" env:"LLM_TEMPERATURE" default:"0.7"`
	MaxTokens    int    `json:"max_tokens" yaml:"max_tokens" env:"LLM_MAX_TOKENS" default:"4096"`
	Timeout      time.Duration `json:"timeout" yaml:"timeout" env:"LLM_TIMEOUT" default:"60s"`
	RetryAttempts int   `json:"retry_attempts" yaml:"retry_attempts" env:"LLM_RETRY_ATTEMPTS" default:"3"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type     string `json:"type" yaml:"type" env:"DB_TYPE" default:"sqlite"`
	Host     string `json:"host" yaml:"host" env:"DB_HOST" default:"localhost"`
	Port     int    `json:"port" yaml:"port" env:"DB_PORT" default:"5432"`
	Name     string `json:"name" yaml:"name" env:"DB_NAME" default:"chatbot"`
	User     string `json:"user" yaml:"user" env:"DB_USER" default:""`
	Password string `json:"password" yaml:"password" env:"DB_PASSWORD" default:""`
	SSLMode  string `json:"ssl_mode" yaml:"ssl_mode" env:"DB_SSL_MODE" default:"disable"`
	FilePath string `json:"file_path" yaml:"file_path" env:"DB_FILE_PATH" default:"./data/chat.db"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	JWTSecret         string        `json:"jwt_secret" yaml:"jwt_secret" env:"JWT_SECRET" default:"your-secret-key"`
	SessionTimeout    time.Duration `json:"session_timeout" yaml:"session_timeout" env:"SESSION_TIMEOUT" default:"24h"`
	RateLimitEnabled  bool          `json:"rate_limit_enabled" yaml:"rate_limit_enabled" env:"RATE_LIMIT_ENABLED" default:"true"`
	RateLimitRPS      int           `json:"rate_limit_rps" yaml:"rate_limit_rps" env:"RATE_LIMIT_RPS" default:"10"`
	CorsEnabled       bool          `json:"cors_enabled" yaml:"cors_enabled" env:"CORS_ENABLED" default:"true"`
	AllowedOrigins    []string      `json:"allowed_origins" yaml:"allowed_origins" env:"ALLOWED_ORIGINS"`
	EncryptionEnabled bool          `json:"encryption_enabled" yaml:"encryption_enabled" env:"ENCRYPTION_ENABLED" default:"false"`
	EncryptionKey     string        `json:"encryption_key" yaml:"encryption_key" env:"ENCRYPTION_KEY"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	Enabled          bool          `json:"enabled" yaml:"enabled" env:"MONITORING_ENABLED" default:"true"`
	MetricsPort      int           `json:"metrics_port" yaml:"metrics_port" env:"METRICS_PORT" default:"9090"`
	TracingEnabled   bool          `json:"tracing_enabled" yaml:"tracing_enabled" env:"TRACING_ENABLED" default:"false"`
	JaegerEndpoint   string        `json:"jaeger_endpoint" yaml:"jaeger_endpoint" env:"JAEGER_ENDPOINT"`
	HealthCheckEnabled bool        `json:"health_check_enabled" yaml:"health_check_enabled" env:"HEALTH_CHECK_ENABLED" default:"true"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval" env:"HEALTH_CHECK_INTERVAL" default:"30s"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `json:"level" yaml:"level" env:"LOG_LEVEL" default:"info"`
	Format     string `json:"format" yaml:"format" env:"LOG_FORMAT" default:"json"`
	Output     string `json:"output" yaml:"output" env:"LOG_OUTPUT" default:"stdout"`
	File       string `json:"file" yaml:"file" env:"LOG_FILE" default:"./logs/app.log"`
	MaxSize    int    `json:"max_size" yaml:"max_size" env:"LOG_MAX_SIZE" default:"100"`
	MaxBackups int    `json:"max_backups" yaml:"max_backups" env:"LOG_MAX_BACKUPS" default:"3"`
	MaxAge     int    `json:"max_age" yaml:"max_age" env:"LOG_MAX_AGE" default:"28"`
	Compress   bool   `json:"compress" yaml:"compress" env:"LOG_COMPRESS" default:"true"`
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Type     string        `json:"type" yaml:"type" env:"CACHE_TYPE" default:"memory"`
	TTL      time.Duration `json:"ttl" yaml:"ttl" env:"CACHE_TTL" default:"1h"`
	MaxSize  int           `json:"max_size" yaml:"max_size" env:"CACHE_MAX_SIZE" default:"1000"`
	RedisURL string        `json:"redis_url" yaml:"redis_url" env:"REDIS_URL"`
}

// FeaturesConfig holds feature flags
type FeaturesConfig struct {
	ArtifactsEnabled    bool `json:"artifacts_enabled" yaml:"artifacts_enabled" env:"FEATURES_ARTIFACTS" default:"true"`
	ToolsEnabled        bool `json:"tools_enabled" yaml:"tools_enabled" env:"FEATURES_TOOLS" default:"true"`
	MCPEnabled          bool `json:"mcp_enabled" yaml:"mcp_enabled" env:"FEATURES_MCP" default:"true"`
	WebSocketEnabled    bool `json:"websocket_enabled" yaml:"websocket_enabled" env:"FEATURES_WEBSOCKET" default:"true"`
	FileUploadEnabled   bool `json:"file_upload_enabled" yaml:"file_upload_enabled" env:"FEATURES_FILE_UPLOAD" default:"false"`
	VoiceEnabled        bool `json:"voice_enabled" yaml:"voice_enabled" env:"FEATURES_VOICE" default:"false"`
	FeedbackEnabled     bool `json:"feedback_enabled" yaml:"feedback_enabled" env:"FEATURES_FEEDBACK" default:"true"`
}

// Manager manages configuration with hot reload capability
type Manager struct {
	mu           sync.RWMutex
	config       *Config
	environment  Environment
	watchers     []chan *Config
	configPath   string
	watcher      *fsnotify.Watcher
	watching     bool
}

// NewManager creates a new configuration manager
func NewManager(environment Environment) *Manager {
	return &Manager{
		config:      &Config{},
		environment: environment,
		watchers:    make([]chan *Config, 0),
	}
}

// Load loads configuration from various sources
func (m *Manager) Load(configPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load default configuration
	m.loadDefaults()

	// Load from file if provided
	if configPath != "" {
		if err := m.loadFromFile(configPath); err != nil {
			return fmt.Errorf("failed to load config from file: %w", err)
		}
		m.configPath = configPath
	}

	// Override with environment variables
	if err := m.loadFromEnv(); err != nil {
		return fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Validate configuration
	if err := m.validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Start watching for configuration changes
	if err := m.StartWatching(); err != nil {
		log.Printf("Warning: Failed to start config watching: %v", err)
	}

	return nil
}

// Get returns a copy of the current configuration
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a deep copy
	configCopy, _ := json.Marshal(m.config)
	var result Config
	json.Unmarshal(configCopy, &result)
	return &result
}

// Watch registers a watcher for configuration changes
func (m *Manager) Watch() chan *Config {
	m.mu.Lock()
	defer m.mu.Unlock()

	watcher := make(chan *Config, 1)
	m.watchers = append(m.watchers, watcher)
	return watcher
}

// Reload reloads configuration from the original sources
func (m *Manager) Reload() error {
	return m.Load(m.configPath)
}

// loadDefaults sets default values
func (m *Manager) loadDefaults() {
	m.config = &Config{
		Server: ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			MaxConns:     1000,
		},
		Agent: AgentConfig{
			MaxConcurrent:      50,
			MaxIdleTime:        30 * time.Minute,
			HealthCheckInterval: 30 * time.Second,
			MaxRetries:         3,
			RetryDelay:         5 * time.Second,
			SessionTimeout:     60 * time.Minute,
			MaxHistory:         100,
		},
		LLM: LLMConfig{
			Provider:      "openai",
			Model:         "gpt-4",
			Temperature:   0.7,
			MaxTokens:     4096,
			Timeout:       60 * time.Second,
			RetryAttempts: 3,
		},
		Database: DatabaseConfig{
			Type:     "sqlite",
			FilePath: "./data/chat.db",
		},
		Security: SecurityConfig{
			JWTSecret:         "your-secret-key",
			SessionTimeout:    24 * time.Hour,
			RateLimitEnabled:  true,
			RateLimitRPS:      10,
			CorsEnabled:       true,
			EncryptionEnabled: false,
		},
		Monitoring: MonitoringConfig{
			Enabled:             true,
			MetricsPort:         9090,
			TracingEnabled:      false,
			HealthCheckEnabled:  true,
			HealthCheckInterval: 30 * time.Second,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
		Cache: CacheConfig{
			Type:    "memory",
			TTL:     time.Hour,
			MaxSize: 1000,
		},
		Features: FeaturesConfig{
			ArtifactsEnabled:  true,
			ToolsEnabled:      true,
			MCPEnabled:        true,
			WebSocketEnabled:  true,
			FileUploadEnabled: false,
			VoiceEnabled:      false,
			FeedbackEnabled:   true,
		},
	}
}

// loadFromFile loads configuration from a file
func (m *Manager) loadFromFile(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Support both JSON and YAML formats
	ext := strings.ToLower(filepath.Ext(configPath))
	if ext == ".json" {
		return json.Unmarshal(data, m.config)
	} else if ext == ".yaml" || ext == ".yml" {
		return yaml.Unmarshal(data, m.config)
	}

	return fmt.Errorf("unsupported config file format: %s", ext)
}

// loadFromEnv loads configuration from environment variables
func (m *Manager) loadFromEnv() error {
	return m.loadEnvStruct(reflect.ValueOf(m.config).Elem())
}

// loadEnvStruct recursively loads environment variables into a struct
func (m *Manager) loadEnvStruct(v reflect.Value) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := m.loadEnvStruct(field); err != nil {
				return err
			}
			continue
		}

		// Get environment variable name
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		envValue := os.Getenv(envTag)
		if envValue == "" {
			continue
		}

		// Convert and set the value
		if err := m.setFieldValue(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a field value from a string
func (m *Manager) setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(duration))
		} else {
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intVal)
		}
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			values := strings.Split(value, ",")
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(strings.TrimSpace(v))
			}
			field.Set(slice)
		}
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// validate validates the configuration
func (m *Manager) validate() error {
	// Validate server configuration
	if m.config.Server.Port <= 0 || m.config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", m.config.Server.Port)
	}

	// Validate LLM configuration
	if m.config.LLM.APIKey == "" {
		return fmt.Errorf("LLM API key is required")
	}

	// Validate agent configuration
	if m.config.Agent.MaxConcurrent <= 0 {
		return fmt.Errorf("max concurrent must be positive")
	}

	return nil
}

// notifyWatchers notifies all watchers of configuration changes
func (m *Manager) notifyWatchers() {
	configCopy := m.Get()
	for _, watcher := range m.watchers {
		select {
		case watcher <- configCopy:
		default:
			// Watcher is not ready to receive
		}
	}
}

// StartWatching starts watching the configuration file for changes
func (m *Manager) StartWatching() error {
	if m.watching || m.configPath == "" {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	m.watcher = watcher
	m.watching = true

	// Watch the config file
	configDir := filepath.Dir(m.configPath)
	if err := m.watcher.Add(configDir); err != nil {
		m.watcher.Close()
		m.watching = false
		return fmt.Errorf("failed to watch config directory: %w", err)
	}

	// Start watching in background
	go m.watchConfigFile()

	return nil
}

// StopWatching stops watching the configuration file
func (m *Manager) StopWatching() {
	if !m.watching || m.watcher == nil {
		return
	}

	m.watching = false
	m.watcher.Close()
	m.watcher = nil
}

// watchConfigFile watches for configuration file changes
func (m *Manager) watchConfigFile() {
	for m.watching {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			// Check if the event is for our config file
			if filepath.Clean(event.Name) != filepath.Clean(m.configPath) {
				continue
			}

			// Handle file events
			if event.Op&fsnotify.Write == fsnotify.Write ||
			   event.Op&fsnotify.Create == fsnotify.Create {
				// Debounce rapid file changes
				time.Sleep(100 * time.Millisecond)

				if err := m.reloadConfig(); err != nil {
					// Log error but continue watching
					fmt.Printf("Error reloading config: %v\n", err)
				} else {
					fmt.Println("Configuration reloaded successfully")
				}
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Config watcher error: %v\n", err)
		}
	}
}

// reloadConfig reloads the configuration from file
func (m *Manager) reloadConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create new config instance
	newConfig := &Config{}

	// Load from file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse based on file extension
	ext := strings.ToLower(filepath.Ext(m.configPath))
	if ext == ".json" {
		if err := json.Unmarshal(data, newConfig); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	} else if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, newConfig); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	// Override with environment variables
	tempManager := &Manager{config: newConfig}
	if err := tempManager.loadFromEnv(); err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Validate new configuration
	if err := m.validateConfig(newConfig); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Apply new configuration
	m.config = newConfig

	// Notify watchers of changes
	m.notifyWatchers()

	return nil
}

// validateConfig validates the configuration
func (m *Manager) validateConfig(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Agent.MaxConcurrent <= 0 {
		return fmt.Errorf("invalid max concurrent agents: %d", config.Agent.MaxConcurrent)
	}

	if config.LLM.Model == "" {
		return fmt.Errorf("LLM model cannot be empty")
	}

	return nil
}