package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AgentState represents the current state of an agent
type AgentState int

const (
	StateUninitialized AgentState = iota
	StateInitializing
	StateReady
	StateRunning
	StatePaused
	StateStopping
	StateStopped
	StateError
)

func (s AgentState) String() string {
	switch s {
	case StateUninitialized:
		return "uninitialized"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// AgentLifecycleConfig holds configuration for agent lifecycle management
type AgentLifecycleConfig struct {
	MaxIdleTime     time.Duration `json:"max_idle_time"`     // How long agent can be idle before stopping
	HealthCheckInterval time.Duration `json:"health_check_interval"` // Health check interval
	MaxRetries      int           `json:"max_retries"`       // Maximum number of retries on error
	RetryDelay      time.Duration `json:"retry_delay"`       // Delay between retries
}

// DefaultAgentLifecycleConfig returns a default lifecycle configuration
func DefaultAgentLifecycleConfig() *AgentLifecycleConfig {
	return &AgentLifecycleConfig{
		MaxIdleTime:        30 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:         3,
		RetryDelay:         5 * time.Second,
	}
}

// AgentLifecycleManager manages the lifecycle of an agent
type AgentLifecycleManager struct {
	id           string
	state        AgentState
	config       *AgentLifecycleConfig
	stateMu      sync.RWMutex
	lastActivity time.Time
	health       HealthStatus
	healthMu     sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	eventChan    chan LifecycleEvent
	eventHandler LifecycleEventHandler
	metrics      *AgentMetrics
}

// LifecycleEvent represents a lifecycle event
type LifecycleEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	State     AgentState `json:"state"`
	Message   string    `json:"message"`
	Error     error     `json:"error,omitempty"`
}

// HealthStatus represents the health status of an agent
type HealthStatus struct {
	IsHealthy     bool      `json:"is_healthy"`
	LastCheck     time.Time `json:"last_check"`
	CheckDuration time.Duration `json:"check_duration"`
	Message       string    `json:"message"`
}

// AgentMetrics tracks various metrics for an agent
type AgentMetrics struct {
	MessageCount      int64         `json:"message_count"`
	TotalTokensIn     int64         `json:"total_tokens_in"`
	TotalTokensOut    int64         `json:"total_tokens_out"`
	ErrorCount        int64         `json:"error_count"`
	AverageLatency    time.Duration `json:"average_latency"`
	Uptime            time.Duration `json:"uptime"`
	StartTime         time.Time     `json:"start_time"`
}

// LifecycleEventHandler handles lifecycle events
type LifecycleEventHandler func(event LifecycleEvent)

// NewAgentLifecycleManager creates a new lifecycle manager for an agent
func NewAgentLifecycleManager(config *AgentLifecycleConfig) *AgentLifecycleManager {
	if config == nil {
		config = DefaultAgentLifecycleConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &AgentLifecycleManager{
		id:           uuid.New().String(),
		state:        StateUninitialized,
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
		eventChan:    make(chan LifecycleEvent, 100),
		metrics:      &AgentMetrics{StartTime: time.Now()},
	}

	// Start background routines
	go manager.eventProcessor()
	go manager.healthChecker()
	go manager.idleMonitor()

	return manager
}

// GetID returns the unique ID of this lifecycle manager
func (lm *AgentLifecycleManager) GetID() string {
	return lm.id
}

// GetState returns the current state of the agent
func (lm *AgentLifecycleManager) GetState() AgentState {
	lm.stateMu.RLock()
	defer lm.stateMu.RUnlock()
	return lm.state
}

// SetState changes the agent state and triggers an event
func (lm *AgentLifecycleManager) SetState(state AgentState, message string, err error) error {
	lm.stateMu.Lock()
	defer lm.stateMu.Unlock()

	oldState := lm.state
	lm.state = state
	lm.lastActivity = time.Now()

	// Validate state transition
	if !lm.isValidTransition(oldState, state) {
		return fmt.Errorf("invalid state transition from %s to %s", oldState, state)
	}

	// Emit event
	event := LifecycleEvent{
		Timestamp: time.Now(),
		EventType: "state_change",
		State:     state,
		Message:   message,
		Error:     err,
	}

	select {
	case lm.eventChan <- event:
	default:
		// Channel is full, log warning
	}

	return nil
}

// isValidTransition checks if a state transition is valid
func (lm *AgentLifecycleManager) isValidTransition(from, to AgentState) bool {
	// Define valid transitions
	validTransitions := map[AgentState][]AgentState{
		StateUninitialized: {StateInitializing, StateStopped},
		StateInitializing:   {StateReady, StateError, StateStopped},
		StateReady:         {StateRunning, StateStopping, StateError},
		StateRunning:       {StateReady, StatePaused, StateStopping, StateError},
		StatePaused:        {StateRunning, StateStopping, StateError},
		StateStopping:      {StateStopped, StateError},
		StateStopped:       {StateInitializing},
		StateError:         {StateInitializing, StateStopped},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, validState := range allowed {
		if to == validState {
			return true
		}
	}

	return false
}

// UpdateActivity updates the last activity timestamp
func (lm *AgentLifecycleManager) UpdateActivity() {
	lm.lastActivity = time.Now()
}

// GetHealthStatus returns the current health status
func (lm *AgentLifecycleManager) GetHealthStatus() HealthStatus {
	lm.healthMu.RLock()
	defer lm.healthMu.RUnlock()
	return lm.health
}

// GetMetrics returns the current agent metrics
func (lm *AgentLifecycleManager) GetMetrics() AgentMetrics {
	lm.stateMu.RLock()
	defer lm.stateMu.RUnlock()

	metrics := *lm.metrics
	metrics.Uptime = time.Since(lm.metrics.StartTime)
	return metrics
}

// IncrementMessageCount increments the message counter
func (lm *AgentLifecycleManager) IncrementMessageCount() {
	lm.stateMu.Lock()
	defer lm.stateMu.Unlock()
	lm.metrics.MessageCount++
	lm.lastActivity = time.Now()
}

// IncrementErrorCount increments the error counter
func (lm *AgentLifecycleManager) IncrementErrorCount() {
	lm.stateMu.Lock()
	defer lm.stateMu.Unlock()
	lm.metrics.ErrorCount++
}

// UpdateTokenMetrics updates token usage metrics
func (lm *AgentLifecycleManager) UpdateTokenMetrics(tokensIn, tokensOut int64) {
	lm.stateMu.Lock()
	defer lm.stateMu.Unlock()
	lm.metrics.TotalTokensIn += tokensIn
	lm.metrics.TotalTokensOut += tokensOut
}

// SetEventHandler sets the handler for lifecycle events
func (lm *AgentLifecycleManager) SetEventHandler(handler LifecycleEventHandler) {
	lm.eventHandler = handler
}

// Stop gracefully stops the lifecycle manager
func (lm *AgentLifecycleManager) Stop() {
	lm.SetState(StateStopping, "Lifecycle manager stopping", nil)

	// Cancel context to stop all background routines
	lm.cancel()

	// Close event channel
	close(lm.eventChan)

	lm.SetState(StateStopped, "Lifecycle manager stopped", nil)
}

// eventProcessor processes lifecycle events
func (lm *AgentLifecycleManager) eventProcessor() {
	for {
		select {
		case <-lm.ctx.Done():
			return
		case event, ok := <-lm.eventChan:
			if !ok {
				return
			}
			if lm.eventHandler != nil {
				lm.eventHandler(event)
			}
		}
	}
}

// healthChecker periodically checks the health of the agent
func (lm *AgentLifecycleManager) healthChecker() {
	ticker := time.NewTicker(lm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-ticker.C:
			lm.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check on the agent
func (lm *AgentLifecycleManager) performHealthCheck() {
	start := time.Now()

	// Basic health check - check if agent is responsive
	isHealthy := lm.GetState() != StateError

	health := HealthStatus{
		IsHealthy:     isHealthy,
		LastCheck:     time.Now(),
		CheckDuration: time.Since(start),
		Message:       "Health check completed",
	}

	if !isHealthy {
		health.Message = "Agent is in error state"
	}

	lm.healthMu.Lock()
	lm.health = health
	lm.healthMu.Unlock()
}

// idleMonitor monitors agent inactivity and stops idle agents
func (lm *AgentLifecycleManager) idleMonitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-ticker.C:
			lm.checkIdleTimeout()
		}
	}
}

// checkIdleTimeout checks if the agent has been idle for too long
func (lm *AgentLifecycleManager) checkIdleTimeout() {
	state := lm.GetState()
	if state == StateRunning || state == StateReady {
		idleTime := time.Since(lm.lastActivity)
		if idleTime > lm.config.MaxIdleTime {
			lm.SetState(StateStopping,
				fmt.Sprintf("Agent idle for %v, stopping", idleTime),
				nil)
		}
	}
}

// GetContext returns the context for this agent
func (lm *AgentLifecycleManager) GetContext() context.Context {
	return lm.ctx
}

// IsStopped returns true if the agent is stopped
func (lm *AgentLifecycleManager) IsStopped() bool {
	return lm.GetState() == StateStopped
}