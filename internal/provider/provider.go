package provider

import "context"

// Role represents the role of a message in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// StreamToken represents a single token (or error) from a streaming response.
type StreamToken struct {
	Token string
	Err   error
}

// Provider defines the interface for AI model providers.
type Provider interface {
	// Name returns the provider identifier (e.g., "google", "deepseek").
	Name() string

	// SupportedModels returns the list of model names this provider supports.
	SupportedModels() []string

	// SendMessageStream sends messages and returns a channel that yields
	// tokens as they are generated. The channel is closed when streaming completes.
	// Returns an error immediately if the request cannot be initiated.
	SendMessageStream(ctx context.Context, model string, messages []Message) (<-chan StreamToken, error)

	// SendMessage sends messages and returns the complete response text.
	SendMessage(ctx context.Context, model string, messages []Message) (string, error)
}
