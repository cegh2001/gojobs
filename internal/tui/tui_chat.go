package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/provider"
	"gojobs/internal/session"
)

// handleChatTextInput accumulates text input from the user.
func (m Model) handleChatTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.chatLoading {
		return m, nil
	}

	// Handle paste: normalize and append
	if msg.Paste {
		m.chatInput += normalizePastedInput(string(msg.Runes))
		return m, nil
	}

	// Handle backspace
	if msg.Type == tea.KeyBackspace {
		if len(m.chatInput) > 0 {
			runes := []rune(m.chatInput)
			m.chatInput = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	// Handle regular key runes
	for _, r := range msg.Runes {
		m.chatInput += string(r)
	}
	return m, nil
}

// handleChatSend processes the current input and sends it to the AI provider.
func (m Model) handleChatSend() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.chatInput)
	if input == "" || m.chatLoading {
		return m, nil
	}

	// Create session if needed
	if m.chatSession == nil {
		m.chatSession = session.NewSession(m.currentModel)
		m.chatMessages = nil
	}

	// Add user message
	m.chatSession.AddMessage("user", input)
	m.chatMessages = m.chatSession.Messages
	m.chatInput = ""
	m.chatLoading = true
	m.chatScroll = 0
	m.err = nil

	// Persist
	_ = m.sessionStore.Save(m.chatSession)

	// Resolve provider and build message list
	prov, err := m.providerRouter.Resolve(m.currentModel)
	if err != nil {
		// Provider resolution failed — show error inline
		errContent := fmt.Sprintf("Error: model %q — %v", m.currentModel, err)
		m.chatSession.AddMessage("assistant", errContent)
		m.chatMessages = m.chatSession.Messages
		m.chatLoading = false
		m.err = err
		_ = m.sessionStore.Save(m.chatSession)
		return m, nil
	}

	// Build provider messages from session history
	var providerMsgs []provider.Message
	for _, msg := range m.chatMessages {
		providerMsgs = append(providerMsgs, provider.Message{
			Role:    provider.Role(msg.Role),
			Content: msg.Content,
		})
	}

	return m, m.sendChatCmd(prov, providerMsgs)
}

// sendChatCmd creates a goroutine that reads stream tokens, accumulates them,
// and sends a single chatResponseMsg when streaming completes.
func (m Model) sendChatCmd(prov provider.Provider, messages []provider.Message) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		ch, err := prov.SendMessageStream(ctx, m.currentModel, messages)
		if err != nil {
			return chatResponseMsg{err: fmt.Errorf("send message: %w", err)}
		}

		var content string
		for token := range ch {
			if token.Err != nil {
				return chatResponseMsg{err: fmt.Errorf("stream error: %w", token.Err)}
			}
			content += token.Token
		}

		return chatResponseMsg{content: content}
	}
}

// handleChatResponse appends the AI response to the chat history.
func (m Model) handleChatResponse(msg chatResponseMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		errContent := fmt.Sprintf("Error: %v", msg.err)
		m.chatSession.AddMessage("assistant", errContent)
		m.chatMessages = m.chatSession.Messages
		m.chatLoading = false
		m.err = msg.err
		m.chatScroll = 0
		_ = m.sessionStore.Save(m.chatSession)
		return m, nil
	}

	if msg.content == "" {
		msg.content = "(no response)"
	}

	m.chatSession.AddMessage("assistant", msg.content)
	m.chatMessages = m.chatSession.Messages
	m.chatLoading = false
	m.chatScroll = 0
	m.err = nil
	_ = m.sessionStore.Save(m.chatSession)

	return m, nil
}

// handleChatNew creates a new chat session.
func (m Model) handleChatNew() (tea.Model, tea.Cmd) {
	// Save current session first
	if m.chatSession != nil {
		_ = m.sessionStore.Save(m.chatSession)
	}

	m.chatSession = session.NewSession(m.currentModel)
	m.chatMessages = nil
	m.chatInput = ""
	m.chatLoading = false
	m.chatScroll = 0
	m.err = nil

	// Add to sessions list
	m.sessions = append([]session.Session{*m.chatSession}, m.sessions...)

	return m, nil
}

// normalizePastedInput replaces newline characters with spaces.
func normalizePastedInput(text string) string {
	return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(text)
}
