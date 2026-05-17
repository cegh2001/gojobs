//go:build integration

package tui

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"gojobs/internal/provider"
	"gojobs/internal/session"
)

// mockIntegrationProvider implements provider.Provider for integration tests.
// It streams mock tokens without making real API calls.
type mockIntegrationProvider struct {
	name            string
	supportedModels []string
}

func (m *mockIntegrationProvider) Name() string              { return m.name }
func (m *mockIntegrationProvider) SupportedModels() []string { return m.supportedModels }

func (m *mockIntegrationProvider) SendMessageStream(_ context.Context, _ string, _ []provider.Message) (<-chan provider.StreamToken, error) {
	ch := make(chan provider.StreamToken, 3)
	go func() {
		defer close(ch)
		ch <- provider.StreamToken{Token: "Mock "}
		ch <- provider.StreamToken{Token: "response"}
	}()
	return ch, nil
}

func (m *mockIntegrationProvider) SendMessage(_ context.Context, _ string, _ []provider.Message) (string, error) {
	return "Mock response", nil
}

var _ provider.Provider = (*mockIntegrationProvider)(nil)

func TestTUIChatFlow(t *testing.T) {
	// Setup: create mock provider
	mockProv := &mockIntegrationProvider{
		name:            "google",
		supportedModels: []string{"gemma-4-31b-it", "gemma-4-26b-a4b-it"},
	}
	router := provider.NewRouter()
	router.Register(mockProv)

	// Setup: create session store in temp dir
	tempDir := t.TempDir()
	store := session.NewStore(tempDir, 10)

	// Setup: create TUI model
	m := NewModel(store, router)

	// Launch TUI with teatest
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Give the TUI time to render and load sessions
	time.Sleep(500 * time.Millisecond)

	// Simulate typing a message
	tm.Type("Hello, this is a test message")
	time.Sleep(300 * time.Millisecond)

	// Send the message directly (bypass key handling for reliability)
	tm.Send(SendMessageMsg{})
	time.Sleep(500 * time.Millisecond)

	// Send mock AI response via StreamTokenMsg
	tm.Send(StreamTokenMsg{Token: "Mock "})
	tm.Send(StreamTokenMsg{Token: "response"})
	time.Sleep(100 * time.Millisecond)

	// Send stream completion
	tm.Send(StreamDoneMsg{Content: "Mock response"})
	time.Sleep(300 * time.Millisecond)

	// Quit the program
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	time.Sleep(200 * time.Millisecond)

	// Wait for program to finish and get final model
	final := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second))
	finalModel := final.(Model)

	// Verify model state after the full flow
	if finalModel.currentSession == nil {
		t.Fatal("expected a current session after sending message")
	}

	// Session should have at least 2 messages: user message + AI response
	if len(finalModel.currentSession.Messages) < 2 {
		t.Fatalf("expected at least 2 messages in session (user + AI), got %d", len(finalModel.currentSession.Messages))
	}

	// First message should be from user
	userMsg := finalModel.currentSession.Messages[0]
	if string(userMsg.Role) != "user" {
		t.Errorf("first message role = %q, want user", userMsg.Role)
	}

	// Second message should be from assistant
	aiMsg := finalModel.currentSession.Messages[1]
	if string(aiMsg.Role) != "assistant" {
		t.Errorf("second message role = %q, want assistant", aiMsg.Role)
	}
	if aiMsg.Content != "Mock response" {
		t.Errorf("second message content = %q, want %q", aiMsg.Content, "Mock response")
	}

	// Loading should be false (stream completed)
	if finalModel.loading {
		t.Error("loading should be false after StreamDoneMsg")
	}

	// Verify final output is non-empty (rendered something)
	finalOut := tm.FinalOutput(t, teatest.WithFinalTimeout(3*time.Second))
	output := readAll(finalOut)
	if len(output) < 100 {
		t.Errorf("final output too short (%d bytes), expected rendered TUI content", len(output))
	}
	if output == "" {
		t.Error("final output should not be empty")
	}
}

// readAll reads all content from a reader.
func readAll(r io.Reader) string {
	var buf strings.Builder
	data := make([]byte, 4096)
	for {
		n, err := r.Read(data)
		if n > 0 {
			buf.Write(data[:n])
		}
		if err != nil {
			break
		}
	}
	return buf.String()
}
