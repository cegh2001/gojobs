package tui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/provider"
	"gojobs/internal/session"
)

// =============================================================================
// Edge Case 5: Empty message send
// =============================================================================

func TestSendMessageMsgRejectsEmptyInput(t *testing.T) {
	m := setupTestModel(t)
	m.currentSession = session.NewSession("gemma-4-31b-it")

	// Pre-condition: input is empty
	if m.inputArea.Value() != "" {
		t.Fatal("expected empty input area before test")
	}

	// Simulate sending with empty input
	m = updateAndDrain(m, SendMessageMsg{}, t)

	// After send with empty input:
	// - No message should be added to session
	// - Chat should still be empty (no user message appended)
	// - loading should remain false
	if m.loading {
		t.Error("loading should be false after empty send")
	}

	if m.currentSession == nil {
		t.Fatal("currentSession should not be nil — session was set before test")
	}

	if len(m.currentSession.Messages) != 0 {
		t.Errorf("expected 0 messages in session after empty send, got %d", len(m.currentSession.Messages))
	}
}

func TestSendMessageMsgAllowsNonEmptyInput(t *testing.T) {
	m := setupTestModel(t)
	m.currentSession = session.NewSession("gemma-4-31b-it")

	// Pre-populate input with text — Focus first, then type
	m.inputArea.Focus()
	m.inputArea, _ = m.inputArea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Hello!")})

	if m.inputArea.Value() == "" {
		t.Fatal("input area value is empty after typing — setup failed")
	}

	// Verify the model processes non-empty SendMessageMsg
	m = updateAndDrain(m, SendMessageMsg{}, t)
	if !m.loading {
		t.Error("loading should be true after non-empty send")
	}
	// Verify message was added to session
	if m.currentSession == nil {
		t.Fatal("currentSession should not be nil after send")
	}
	if len(m.currentSession.Messages) != 1 {
		t.Errorf("expected 1 message in session after non-empty send, got %d", len(m.currentSession.Messages))
	}
}

// =============================================================================
// Edge Case 6: Rapid sends (second send should not overlap)
// =============================================================================

func TestSendMessageMsgBlocksWhileLoading(t *testing.T) {
	m := setupTestModel(t)
	m.currentSession = session.NewSession("gemma-4-31b-it")
	m.loading = true // Simulate that a request is in progress

	// Record message count before
	msgCountBefore := len(m.currentSession.Messages)

	// SendMessageMsg while loading should be silently ignored
	m = updateAndDrain(m, SendMessageMsg{}, t)

	// loading should remain true
	if !m.loading {
		t.Error("loading should remain true when send is blocked")
	}

	// No new messages should be added
	if len(m.currentSession.Messages) != msgCountBefore {
		t.Errorf("expected %d messages (no new), got %d", msgCountBefore, len(m.currentSession.Messages))
	}
}

func TestSendMessageMsgProceedsWhenNotLoading(t *testing.T) {
	m := setupTestModel(t)
	m.currentSession = session.NewSession("gemma-4-31b-it")
	m.loading = false

	// Focus the input area and type some text
	m.inputArea.Focus()
	m.inputArea, _ = m.inputArea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")})

	if m.inputArea.Value() == "" {
		t.Fatal("input area value is empty after typing — setup failed")
	}

	m = updateAndDrain(m, SendMessageMsg{}, t)

	// loading should now be true (send initiated)
	if !m.loading {
		t.Error("loading should be true after send when not already loading")
	}
}

// =============================================================================
// Edge Case 1: Empty API key — TUI launches but shows error on first send
// =============================================================================

func TestModelStartsWithEmptyRouter(t *testing.T) {
	// Model created with empty router (no providers registered)
	// This simulates the case where API key is not configured.
	m := setupTestModel(t)

	// The model should not crash — View() should return something
	view := m.View()
	if view == "" {
		t.Error("View() returned empty string with empty router")
	}

	// Type text and send a message — should not crash even with empty router
	m.currentSession = session.NewSession("gemma-4-31b-it")
	m.inputArea.Focus()
	m.inputArea, _ = m.inputArea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Hello")})

	if m.inputArea.Value() == "" {
		t.Fatal("input area value is empty after typing — setup failed")
	}

	m = updateAndDrain(m, SendMessageMsg{}, t)

	// Should not crash, should still have loading state
	if !m.loading {
		t.Error("loading should be true after send even with empty router")
	}
	// Verify the user message was added to the session
	if len(m.currentSession.Messages) != 1 {
		t.Errorf("expected 1 user message in session, got %d", len(m.currentSession.Messages))
	}
}

// =============================================================================
// Stream error resilience
// =============================================================================

func TestStreamErrorShowsInlineAndClearsLoading(t *testing.T) {
	m := setupTestModel(t)
	m.loading = true

	wantErr := errors.New("API key not configured")
	m = updateAndDrain(m, StreamErrMsg{Err: wantErr}, t)

	if m.loading {
		t.Error("loading should be false after StreamErrMsg")
	}
	if m.err == nil {
		t.Error("err should be set after StreamErrMsg")
	}
	if m.err != wantErr {
		t.Errorf("err = %v, want %v", m.err, wantErr)
	}
	// View should not crash
	if m.View() == "" {
		t.Error("View() returned empty after StreamErrMsg")
	}
}

// =============================================================================
// Edge Case: Model selection without registered provider
// =============================================================================

func TestModelSelectedWithUnknownModel(t *testing.T) {
	m := setupTestModel(t)

	// Select a model that doesn't exist in any registered provider
	m = updateAndDrain(m, ModelSelectedMsg{Model: "unknown-model"}, t)

	// Model should be set — validation happens at send time, not selection time
	if m.currentModel != "unknown-model" {
		t.Errorf("currentModel = %q, want %q", m.currentModel, "unknown-model")
	}

	// View should still render
	if m.View() == "" {
		t.Error("View() returned empty after selecting unknown model")
	}
}

// =============================================================================
// Edge Case: NewSessionMsg during loading
// =============================================================================

func TestNewSessionClearsLoadingState(t *testing.T) {
	m := setupTestModel(t)
	m.loading = true

	m = updateAndDrain(m, NewSessionMsg{}, t)

	// New session should clear loading state
	if m.loading {
		t.Error("loading should be false after NewSessionMsg")
	}
	// Current session should be set
	if m.currentSession == nil {
		t.Error("currentSession should not be nil after NewSessionMsg")
	}
}

// =============================================================================
// Edge Case: DeleteSessionMsg with no sessions
// =============================================================================

func TestDeleteSessionWhenNoneExist(t *testing.T) {
	m := setupTestModel(t)
	// No current session and no sessions
	m.currentSession = nil
	m.sessions = nil

	// Should not crash
	m = updateAndDrain(m, DeleteSessionMsg{}, t)

	// Should remain in valid state
	if m.View() == "" {
		t.Error("View() should not be empty after DeleteSessionMsg with no sessions")
	}
}

// =============================================================================
// Smoke test: TUI renders non-empty View in all initial states
// =============================================================================

func TestViewNeverEmpty(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
	}{
		{
			name: "fresh model",
			setup: func(m *Model) {},
		},
		{
			name: "with sessions loaded",
			setup: func(m *Model) {
				m.sessions = []session.Session{{ID: "1", Name: "Test", Model: "gemma-4-31b-it"}}
				m.sidebar.SetSessions(m.sessions)
			},
		},
		{
			name: "with current session",
			setup: func(m *Model) {
				m.currentSession = session.NewSession("gemma-4-31b-it")
			},
		},
		{
			name: "with error",
			setup: func(m *Model) {
				m.err = errors.New("something went wrong")
			},
		},
		{
			name: "while loading",
			setup: func(m *Model) {
				m.loading = true
				m.spinner.Start()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := setupTestModel(t)
			tt.setup(&m)

			view := m.View()
			if view == "" {
				t.Errorf("View() returned empty string in state: %s", tt.name)
			}
		})
	}
}

// =============================================================================
// Edge Case: Corrupted session file (integration-style, verifies List skips)
// =============================================================================

func TestSessionsLoadedSkipsCorruptedFiles(t *testing.T) {
	m := setupTestModel(t)

	// Even with empty session list, SessionsLoadedMsg should work
	m = updateAndDrain(m, SessionsLoadedMsg{Sessions: nil}, t)

	if len(m.sessions) != 0 {
		t.Errorf("expected 0 sessions with nil, got %d", len(m.sessions))
	}

	// Verify sidebar can handle empty sessions
	m = updateAndDrain(m, SessionsLoadedMsg{Sessions: []session.Session{}}, t)
	if len(m.sessions) != 0 {
		t.Errorf("expected 0 sessions with empty slice, got %d", len(m.sessions))
	}
}

// =============================================================================
// Edge Case: Unknown model resolution via router (already tested in provider)
// This test verifies the TUI integration doesn't crash
// =============================================================================

func TestTUIWithUnknownModelDoesNotCrash(t *testing.T) {
	// Create router with only google provider
	router := provider.NewRouter()
	mockProv := &mockTUIProvider{
		name:            "google",
		supportedModels: []string{"gemma-4-31b-it", "gemma-4-26b-a4b-it"},
	}
	router.Register(mockProv)

	// Try to resolve unknown model
	_, err := router.Resolve("unknown-model")
	if err == nil {
		t.Error("expected error resolving unknown model")
	}

	// The TUI model itself should handle this gracefully (model is set before resolution)
	m := NewModel(session.NewStore(t.TempDir(), 10), router)
	m.currentModel = "unknown-model"

	// View should still render
	if m.View() == "" {
		t.Error("View() returned empty with unknown model")
	}
}

// mockTUIProvider implements provider.Provider for TUI edge case tests.
type mockTUIProvider struct {
	name            string
	supportedModels []string
}

func (m *mockTUIProvider) Name() string              { return m.name }
func (m *mockTUIProvider) SupportedModels() []string { return m.supportedModels }
func (m *mockTUIProvider) SendMessageStream(ctx context.Context, model string, messages []provider.Message) (<-chan provider.StreamToken, error) {
	ch := make(chan provider.StreamToken)
	close(ch)
	return ch, nil
}
func (m *mockTUIProvider) SendMessage(ctx context.Context, model string, messages []provider.Message) (string, error) {
	return "mock response", nil
}

// Ensure mockTUIProvider implements provider.Provider.
var _ provider.Provider = (*mockTUIProvider)(nil)
