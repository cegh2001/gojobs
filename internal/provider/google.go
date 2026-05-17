package provider

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GoogleProvider implements the Provider interface for Google's Gemini models.
type GoogleProvider struct {
	client *genai.Client
}

// NewGoogleProvider creates a new GoogleProvider with the given API key.
func NewGoogleProvider(ctx context.Context, apiKey string) (*GoogleProvider, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create Google Gen AI client: %w", err)
	}

	return &GoogleProvider{client: client}, nil
}

// Name returns "google".
func (g *GoogleProvider) Name() string {
	return "google"
}

// SupportedModels returns the list of Gemma models this provider supports.
func (g *GoogleProvider) SupportedModels() []string {
	return []string{"gemma-4-31b-it", "gemma-4-26b-a4b-it"}
}

// SendMessageStream sends messages using a chat session with streaming.
// System messages (RoleSystem) are extracted and used as SystemInstruction.
// History messages (all except the last) are used to create the chat.
// The last message is sent as the current user input.
func (g *GoogleProvider) SendMessageStream(ctx context.Context, model string, messages []Message) (<-chan StreamToken, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	// Extract system message(s) for SystemInstruction
	var systemParts []*genai.Part
	var chatMessages []Message
	for _, msg := range messages {
		if msg.Role == RoleSystem {
			systemParts = append(systemParts, &genai.Part{Text: msg.Content})
		} else {
			chatMessages = append(chatMessages, msg)
		}
	}

	if len(chatMessages) == 0 {
		return nil, fmt.Errorf("no non-system messages provided")
	}

	// Build history from all chat messages except the last one
	var history []*genai.Content
	for i := 0; i < len(chatMessages)-1; i++ {
		history = append(history, toGenaiContent(chatMessages[i]))
	}

	// Build config with system instruction if present
	var config *genai.GenerateContentConfig
	if len(systemParts) > 0 {
		config = &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Role:  string(genai.RoleUser),
				Parts: systemParts,
			},
		}
	}

	chat, err := g.client.Chats.Create(ctx, model, config, history)
	if err != nil {
		return nil, fmt.Errorf("create chat session: %w", err)
	}

	lastMsg := chatMessages[len(chatMessages)-1]

	ch := make(chan StreamToken, 32)
	go func() {
		defer close(ch)

		stream := chat.SendMessageStream(ctx, genai.Part{Text: lastMsg.Content})
		for resp, err := range stream {
			if err != nil {
				ch <- StreamToken{Err: fmt.Errorf("stream error: %w", err)}
				return
			}

			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
					if part.Text != "" {
						select {
						case ch <- StreamToken{Token: part.Text}:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return ch, nil
}

// SendMessage sends messages and returns the complete response text.
func (g *GoogleProvider) SendMessage(ctx context.Context, model string, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages provided")
	}

	// Convert all messages to genai Content for a single-turn request
	var contents []*genai.Content
	for _, msg := range messages {
		contents = append(contents, toGenaiContent(msg))
	}

	resp, err := g.client.Models.GenerateContent(ctx, model, contents, nil)
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}

	return resp.Text(), nil
}

// toGenaiContent converts a provider.Message to a genai.Content.
func toGenaiContent(msg Message) *genai.Content {
	role := string(genai.RoleUser)

	switch msg.Role {
	case RoleSystem:
		role = string(genai.RoleUser) // genai Content role does not support "system"; map to "user"
	case RoleAssistant:
		role = string(genai.RoleModel)
	default:
		role = string(genai.RoleUser)
	}

	return &genai.Content{
		Role:  role,
		Parts: []*genai.Part{{Text: msg.Content}},
	}
}
