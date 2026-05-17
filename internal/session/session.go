package session

import (
	"fmt"
	"strings"
	"time"
)

// Role represents the role of a message in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a single message in a conversation.
type Message struct {
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Session represents a conversation session.
type Session struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	JobURL      string    `json:"job_url,omitempty"`
	ProfilePath string    `json:"profile_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewSession creates a new session with a unique ID and the given model.
func NewSession(model string) *Session {
	now := time.Now()
	id := fmt.Sprintf("%x", now.UnixNano())

	return &Session{
		ID:        id,
		Model:     model,
		Messages:  make([]Message, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage appends a message to the session history and updates UpdatedAt.
func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, Message{
		Role:      Role(role),
		Content:   content,
		Timestamp: time.Now(),
	})

	if s.Name == "" {
		s.Name = generateName(content)
	}

	s.UpdatedAt = time.Now()
}

// generateName creates a session name from the first message content.
func generateName(firstMessage string) string {
	trimmed := strings.TrimSpace(firstMessage)
	if trimmed == "" {
		return "New chat"
	}

	runes := []rune(trimmed)
	if len(runes) > 40 {
		return string(runes[:40])
	}

	return trimmed
}
