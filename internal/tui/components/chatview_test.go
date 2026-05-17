package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/session"
)

func TestNewChatView(t *testing.T) {
	cv := NewChatView()
	if cv.View() == "" {
		t.Error("NewChatView().View() returned empty string")
	}
}

func TestAppendToken(t *testing.T) {
	cv := NewChatView()
	cv.AppendToken("Hello")
	cv.AppendToken(" ")
	cv.AppendToken("World")

	result := cv.View()
	if !strings.Contains(result, "Hello World") {
		t.Errorf("View() after AppendToken does not contain expected content: %s", result)
	}
}

func TestAppendMessage(t *testing.T) {
	cv := NewChatView()
	cv.AppendMessage("user", "Hello AI")
	cv.AppendMessage("assistant", "Hello Human")

	result := cv.View()
	if !strings.Contains(result, "Hello AI") {
		t.Errorf("View() does not contain user message: %s", result)
	}
	if !strings.Contains(result, "Hello Human") {
		t.Errorf("View() does not contain assistant message: %s", result)
	}
}

func TestClearChatView(t *testing.T) {
	cv := NewChatView()
	cv.AppendToken("some content")

	cv.Clear()

	// After clear, the viewport should not contain old content
	result := cv.View()
	if strings.Contains(result, "some content") {
		t.Error("View() after Clear() still contains old content")
	}
}

func TestLoadHistory(t *testing.T) {
	messages := []session.Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
	}

	cv := NewChatView()
	cv.LoadHistory(messages)

	result := cv.View()
	if !strings.Contains(result, "Message 1") {
		t.Errorf("View() does not contain history message 'Message 1': %s", result)
	}
	if !strings.Contains(result, "Response 1") {
		t.Errorf("View() does not contain history message 'Response 1': %s", result)
	}
}

func TestSetSizeChatView(t *testing.T) {
	cv := NewChatView()
	cv.SetSize(80, 24)
	// Size is set internally; verify View() still returns non-empty
	if cv.View() == "" {
		t.Error("View() is empty after SetSize")
	}
}

func TestChatViewUpdate(t *testing.T) {
	cv := NewChatView()
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := cv.Update(msg)

	if cmd != nil {
		t.Log("Update returned a command (expected for viewport resize)")
	}
	if updated.View() == "" {
		t.Error("View() after Update returned empty string")
	}
}
