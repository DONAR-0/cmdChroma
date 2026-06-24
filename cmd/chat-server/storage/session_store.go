// Package storage provides session storage for the chat server.
package storage

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Message is a single turn in a chat session.
type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
	Tokens  int    `json:"tokens"` // number of tokens in assistant response
}

// Session represents a single user's chat session against a collection.
type Session struct {
	ID         string    `json:"id"`
	APIKey     string    `json:"-"` // not exposed to clients
	Collection string    `json:"collection"`
	Messages   []Message `json:"messages"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SessionStore manages in-memory chat sessions.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session // key: sessionID
}

// NewSessionStore creates an empty in-memory session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// GetOrCreate returns an existing session by ID or creates a new one with the given collection.
// If sessionID is empty, a new UUID is generated.
func (s *SessionStore) GetOrCreate(apiKey, sessionID, collection string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	existing, ok := s.sessions[sessionID]
	if ok && existing.APIKey == apiKey {
		existing.UpdatedAt = time.Now()
		return existing
	}

	// Session doesn't exist or mismatched API key — create fresh
	// (API key mismatch: don't leak session existence, return new)
	sess := &Session{
		ID:         sessionID,
		APIKey:     apiKey,
		Collection: collection,
		Messages:   nil,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	s.sessions[sessionID] = sess

	return sess
}

// Get retrieves a session by ID and API key. Returns nil if not found or key mismatch.
func (s *SessionStore) Get(apiKey, sessionID string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[sessionID]
	if !ok || sess.APIKey != apiKey {
		return nil
	}

	return sess
}

// AppendMessage adds a message to the session's history.
func (s *SessionStore) AppendMessage(sessionID, apiKey, role, content string, tokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok || sess.APIKey != apiKey {
		return nil // Treat as no-op rather than error
	}

	sess.Messages = append(sess.Messages, Message{
		Role:    role,
		Content: content,
		Tokens:  tokens,
	})
	sess.UpdatedAt = time.Now()

	return nil
}

// Clear removes all messages from a session but preserves the session entry.
func (s *SessionStore) Clear(apiKey, sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok || sess.APIKey != apiKey {
		return false
	}

	sess.Messages = nil
	sess.UpdatedAt = time.Now()

	return true
}

// Delete removes a session entirely.
func (s *SessionStore) Delete(apiKey, sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok || sess.APIKey != apiKey {
		return false
	}

	delete(s.sessions, sessionID)

	return true
}

// List returns all sessions for an API key (without API key in the response).
func (s *SessionStore) List(apiKey string) []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Session, 0)

	for _, sess := range s.sessions {
		if sess.APIKey == apiKey {
			out = append(out, sess)
		}
	}

	return out
}
