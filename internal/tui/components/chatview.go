package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/session"
)

// ChatViewModel wraps a viewport.Model for chat message display.
type ChatViewModel struct {
	viewport viewport.Model
	content  strings.Builder
}

// NewChatView creates a new ChatViewModel with default dimensions.
func NewChatView() ChatViewModel {
	vp := viewport.New(80, 20)
	return ChatViewModel{viewport: vp}
}

// AppendToken appends a streaming token to the viewport content.
func (cv *ChatViewModel) AppendToken(token string) {
	cv.content.WriteString(token)
	cv.viewport.SetContent(cv.content.String())
	cv.viewport.GotoBottom()
}

// AppendMessage formats and appends a full message to the viewport.
func (cv *ChatViewModel) AppendMessage(role, content string) {
	current := cv.content.String()
	if current != "" {
		current += "\n\n"
	}

	var line string
	if role == "user" {
		line = fmt.Sprintf("┌─ You ──────────────────────────────\n│ %s\n└────────────────────────────────────", content)
	} else {
		line = fmt.Sprintf("┌─ AI ─────────────────────────────\n│ %s\n└────────────────────────────────────", content)
	}

	cv.content.Reset()
	cv.content.WriteString(current + line)
	cv.viewport.SetContent(cv.content.String())
	cv.viewport.GotoBottom()
}

// Clear resets the viewport content.
func (cv *ChatViewModel) Clear() {
	cv.content.Reset()
	cv.viewport.SetContent("")
}

// LoadHistory formats and loads all session messages into the viewport.
func (cv *ChatViewModel) LoadHistory(messages []session.Message) {
	var lines []string
	for _, msg := range messages {
		var line string
		if msg.Role == "user" {
			line = fmt.Sprintf("┌─ You ──────────────────────────────\n│ %s\n└────────────────────────────────────", msg.Content)
		} else {
			line = fmt.Sprintf("┌─ AI ─────────────────────────────\n│ %s\n└────────────────────────────────────", msg.Content)
		}
		lines = append(lines, line)
	}

	cv.content.Reset()
	cv.content.WriteString(strings.Join(lines, "\n\n"))
	cv.viewport.SetContent(cv.content.String())
	cv.viewport.GotoBottom()
}

// Update delegates to the underlying viewport.Model.
func (cv ChatViewModel) Update(msg tea.Msg) (ChatViewModel, tea.Cmd) {
	var cmd tea.Cmd
	cv.viewport, cmd = cv.viewport.Update(msg)
	return cv, cmd
}

// View renders the viewport.
func (cv ChatViewModel) View() string {
	return cv.viewport.View()
}

// SetSize updates the viewport dimensions.
func (cv *ChatViewModel) SetSize(w, h int) {
	cv.viewport.Width = w
	cv.viewport.Height = h
}
