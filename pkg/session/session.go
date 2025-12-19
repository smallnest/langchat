package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Message represents a single chat message
type Message struct {
	ID        string    `json:"id"`        // unique message id
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`   // message content
	Timestamp time.Time `json:"timestamp"` // when the message was sent
	Feedback  string    `json:"feedback"`  // "like", "dislike", or empty
}

// Session represents a chat session with history
type Session struct {
	ID        string    `json:"id"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	mu        sync.RWMutex
}

// SessionStore defines the interface for session persistence
type SessionStore interface {
	Save(session *Session) error
	Load(id string) (*Session, error)
	Delete(id string) error
	List() ([]*Session, error)
}

// FileSessionStore implements SessionStore using local files
type FileSessionStore struct {
	sessionDir string
}

// NewFileSessionStore creates a new FileSessionStore
func NewFileSessionStore(sessionDir string) *FileSessionStore {
	os.MkdirAll(sessionDir, 0755)
	return &FileSessionStore{sessionDir: sessionDir}
}

func (s *FileSessionStore) Save(session *Session) error {
	// Only save sessions that have messages
	if len(session.Messages) == 0 {
		// If the session has no messages, don't save it to disk
		// If it exists on disk from before, delete it
		filePath := filepath.Join(s.sessionDir, fmt.Sprintf("%s.json", session.ID))
		if _, err := os.Stat(filePath); err == nil {
			// File exists, delete it
			os.Remove(filePath)
		}
		return nil
	}

	filePath := filepath.Join(s.sessionDir, fmt.Sprintf("%s.json", session.ID))

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

func (s *FileSessionStore) Load(id string) (*Session, error) {
	filePath := filepath.Join(s.sessionDir, id+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("session file not found: %s", id)
	}

	var session Session
	err = json.Unmarshal(data, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %v", err)
	}

	return &session, nil
}

func (s *FileSessionStore) Delete(id string) error {
	filePath := filepath.Join(s.sessionDir, fmt.Sprintf("%s.json", id))
	return os.Remove(filePath)
}

func (s *FileSessionStore) List() ([]*Session, error) {
	files, err := os.ReadDir(s.sessionDir)
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		id := strings.TrimSuffix(file.Name(), ".json")
		session, err := s.Load(id)
		if err != nil {
			continue
		}

		// Only include sessions that have messages
		if len(session.Messages) > 0 {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// SessionManager manages multiple chat sessions with an in-memory cache
type SessionManager struct {
	sessions   map[string]*Session
	store      SessionStore
	maxHistory int
	mu         sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager(store SessionStore, maxHistory int) *SessionManager {
	sm := &SessionManager{
		sessions:   make(map[string]*Session),
		store:      store,
		maxHistory: maxHistory,
	}

	// Load all sessions at startup
	sm.loadSessions()

	return sm
}

// GetMaxHistory returns the maximum history length
func (sm *SessionManager) GetMaxHistory() int {
	return sm.maxHistory
}

// CreateSession creates a new chat session
func (sm *SessionManager) CreateSession() *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:        uuid.New().String(),
		Messages:  make([]Message, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	sm.sessions[session.ID] = session
	return session
}

// GetSession retrieves a session by ID (lazy loads from store if not in memory)
func (sm *SessionManager) GetSession(id string) (*Session, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[id]
	sm.mu.RUnlock()

	if exists {
		return session, nil
	}

	// Try to load from store
	session, err := sm.store.Load(id)
	if err != nil {
		return nil, err
	}

	// Store in memory for future access
	sm.mu.Lock()
	sm.sessions[id] = session
	sm.mu.Unlock()

	return session, nil
}

// ListSessions returns all active sessions
func (sm *SessionManager) ListSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, id)
	return sm.store.Delete(id)
}

// AddMessage adds a message to a session
func (sm *SessionManager) AddMessage(sessionID, role, content string) (string, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	msgID := uuid.New().String()
	message := Message{
		ID:        msgID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, message)
	session.UpdatedAt = time.Now()

	if sm.maxHistory > 0 && len(session.Messages) > sm.maxHistory {
		session.Messages = session.Messages[len(session.Messages)-sm.maxHistory:]
	}

	// Save to store
	sm.store.Save(session)

	return msgID, nil
}

// UpdateMessageFeedback updates the feedback for a specific message
func (sm *SessionManager) UpdateMessageFeedback(sessionID, messageID, feedback string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	found := false
	for i := range session.Messages {
		if session.Messages[i].ID == messageID {
			session.Messages[i].Feedback = feedback
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("message not found: %s", messageID)
	}

	session.UpdatedAt = time.Now()
	return sm.store.Save(session)
}

// GetMessages retrieves all messages from a session
func (sm *SessionManager) GetMessages(sessionID string) ([]Message, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	messages := make([]Message, len(session.Messages))
	copy(messages, session.Messages)

	return messages, nil
}

func (sm *SessionManager) loadSessions() {
	sessions, err := sm.store.List()
	if err != nil {
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, s := range sessions {
		sm.sessions[s.ID] = s
	}
}

// ClearHistory clears all messages in a session
func (sm *SessionManager) ClearHistory(sessionID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Messages = make([]Message, 0)
	session.UpdatedAt = time.Now()

	return sm.store.Save(session)
}