package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a conversation session
type Session struct {
	ID           string
	UserID       string
	CreatedAt    time.Time
	LastActivity time.Time
	Messages     []Message
	mu           sync.RWMutex
}

// AddMessage adds a message to the session
func (s *Session) AddMessage(role, content string, data interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Messages = append(s.Messages, Message{
		Role:    role,
		Content: content,
	})
	s.LastActivity = time.Now()
}

// GetMessages returns all messages in the session
func (s *Session) GetMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messages := make([]Message, len(s.Messages))
	copy(messages, s.Messages)
	return messages
}

// SessionManager manages AI agent sessions
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	maxAge   time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(maxAge time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		maxAge:   maxAge,
	}

	// Start cleanup goroutine
	go sm.cleanupExpired()

	return sm
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession(ctx context.Context, userID string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Messages:     []Message{},
	}

	sm.sessions[session.ID] = session
	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, sessionID)
	return nil
}

// ListUserSessions returns all sessions for a user
func (sm *SessionManager) ListUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []*Session
	for _, session := range sm.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// cleanupExpired removes expired sessions
func (sm *SessionManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, session := range sm.sessions {
			if now.Sub(session.LastActivity) > sm.maxAge {
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}
