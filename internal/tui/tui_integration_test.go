//go:build integration

// Package tui provides integration tests for the BubbleTea TUI.
//
// KNOWN ISSUE: Running this test alongside unit tests (without -run filter)
// causes "fatal error: found bad pointer in Go heap" due to a sync.Pool
// bug in charmbracelet/x/ansi used by teatest. This is an upstream issue,
// not a gojobs bug.
//
// Workaround: Run integration tests separately:
//
//	go test -tags=integration -run TestTUIChatFlow ./internal/tui/...
//
// Do NOT run: go test -tags=integration ./...  (may crash)
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

	// Setup: create TUI model with pre-typed input
	m := NewModel(store, router)
	m.width = 80
	m.height = 40

	// Pre-populate input by simulating typing
	message := "Hello, this is a test message"
	for _, r := range message {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	// Verify input has text before launching teatest
	if m.chatInput == "" {
		t.Fatal("chatInput is empty after pre-typing — setup failed")
	}

	// Launch TUI with teatest
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Give the TUI time to render and load sessions
	time.Sleep(500 * time.Millisecond)

	// Send Enter to trigger message send
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(500 * time.Millisecond)

	// Wait for the goroutine to finish streaming
	time.Sleep(500 * time.Millisecond)

	// Quit the program
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	time.Sleep(200 * time.Millisecond)

	// Wait for program to finish and get final model
	final := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second))
	finalModel := final.(Model)

	// Verify model state after the full flow
	if finalModel.chatSession == nil {
		t.Fatal("expected a chat session after sending message")
	}

	// Session should have at least 2 messages: user message + AI response
	if len(finalModel.chatSession.Messages) < 2 {
		t.Fatalf("expected at least 2 messages in session (user + AI), got %d", len(finalModel.chatSession.Messages))
	}

	// First message should be from user
	userMsg := finalModel.chatSession.Messages[0]
	if string(userMsg.Role) != "user" {
		t.Errorf("first message role = %q, want user", userMsg.Role)
	}

	// Second message should be from assistant
	aiMsg := finalModel.chatSession.Messages[1]
	if string(aiMsg.Role) != "assistant" {
		t.Errorf("second message role = %q, want assistant", aiMsg.Role)
	}
	if aiMsg.Content != "Mock response" {
		t.Errorf("second message content = %q, want %q", aiMsg.Content, "Mock response")
	}

	// Loading should be false (stream completed)
	if finalModel.chatLoading {
		t.Error("chatLoading should be false after StreamDoneMsg")
	}

	// Verify final output is non-empty (rendered something)
	finalOut := tm.FinalOutput(t, teatest.WithFinalTimeout(3*time.Second))
	output := readAllStr(finalOut)
	if len(output) < 100 {
		t.Errorf("final output too short (%d bytes), expected rendered TUI content", len(output))
	}
	if output == "" {
		t.Error("final output should not be empty")
	}
}

// readAllStr reads all content from a reader into a string.
func readAllStr(r io.Reader) string {
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
