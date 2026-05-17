package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/provider"
	"gojobs/internal/session"
)

// =============================================================================
// Edge Case: Empty message send
// =============================================================================

func TestSendEmptyInputIsIgnored(t *testing.T) {
	m := setupTestModel(t)
	m.chatSession = session.NewSession("gemma-4-31b-it")

	// Pre-condition: input is empty
	if m.chatInput != "" {
		t.Fatal("expected empty chatInput before test")
	}

	// Send with empty input (Enter)
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnter}, t)

	// Nothing should happen — no new messages
	if m.chatLoading {
		t.Error("loading should be false after empty send")
	}
	if m.chatSession == nil {
		t.Fatal("chatSession should still exist")
	}
	if len(m.chatSession.Messages) != 0 {
		t.Errorf("expected 0 messages after empty send, got %d", len(m.chatSession.Messages))
	}
}

// =============================================================================
// Edge Case: Send while loading (rapid sends)
// =============================================================================

func TestSendWhileLoadingIsIgnored(t *testing.T) {
	m := setupTestModel(t)
	m.chatSession = session.NewSession("gemma-4-31b-it")
	m.chatLoading = true
	m.chatInput = "some text"

	msgCountBefore := len(m.chatSession.Messages)

	// Send (Enter)
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnter}, t)

	// Should be silently ignored
	if !m.chatLoading {
		t.Error("chatLoading should remain true")
	}
	if len(m.chatSession.Messages) != msgCountBefore {
		t.Errorf("no new messages expected, got %d (was %d)", len(m.chatSession.Messages), msgCountBefore)
	}
}

func TestSendWhileLoadingBlockedTextInput(t *testing.T) {
	m := setupTestModel(t)
	m.chatLoading = true

	// Try typing while loading
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Hello")}, t)

	// Text input should be ignored while loading
	if m.chatInput != "" {
		t.Errorf("chatInput should be empty while loading, got %q", m.chatInput)
	}
}

// =============================================================================
// Edge Case: Empty router (no API key)
// =============================================================================

func TestModelStartsWithEmptyRouter(t *testing.T) {
	m := setupTestModel(t)
	m.width = 80
	m.height = 40

	// View should render without crash
	view := m.View()
	if view == "" {
		t.Error("View() returned empty string with empty router")
	}

	// Type and send — should not crash, shows inline error
	m.chatInput = "Hello"
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnter}, t)

	// Should have session with user message and error message
	if m.chatSession == nil {
		t.Fatal("chatSession should be created")
	}
	if len(m.chatSession.Messages) < 1 {
		t.Errorf("expected at least 1 message, got %d", len(m.chatSession.Messages))
	}
}

// =============================================================================
// Edge Case: Stream error
// =============================================================================

func TestStreamErrorClearsLoading(t *testing.T) {
	m := setupTestModel(t)
	m.chatSession = session.NewSession("gemma-4-31b-it")
	m.chatLoading = true

	wantErr := errors.New("API key not configured")
	m = updateAndDrain(m, chatResponseMsg{err: wantErr}, t)

	if m.chatLoading {
		t.Error("chatLoading should be false after error")
	}
	if m.err == nil {
		t.Error("err should be set after error response")
	}
	if m.View() == "" {
		t.Error("View() returned empty after stream error")
	}
}

// =============================================================================
// Edge Case: Unknown model
// =============================================================================

func TestUnknownModelDoesNotCrash(t *testing.T) {
	router := provider.NewRouter()
	mockProv := &mockTUIProvider{
		name:            "google",
		supportedModels: []string{"gemma-4-31b-it", "gemma-4-26b-a4b-it"},
	}
	router.Register(mockProv)

	m := NewModel(session.NewStore(t.TempDir(), 10), router)
	m.currentModel = "unknown-model"
	m.width = 80
	m.height = 40

	// View should still render
	if m.View() == "" {
		t.Error("View() returned empty with unknown model")
	}
}

// =============================================================================
// Edge Case: NewSessionMsg (Ctrl+N) with loading
// =============================================================================

func TestNewSessionClearsLoadingState(t *testing.T) {
	m := setupTestModel(t)
	m.chatLoading = true

	// Ctrl+N
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyCtrlN}, t)

	if m.chatLoading {
		t.Error("chatLoading should be false after new session")
	}
	if m.chatSession == nil {
		t.Error("chatSession should be set after new session")
	}
}

// =============================================================================
// Edge Case: Delete session with no sessions
// =============================================================================

func TestDeleteSessionWhenNoneExist(t *testing.T) {
	m := setupTestModel(t)
	m.state = stateSessions
	m.sessions = nil
	m.cursor = 0

	// Should not crash
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyDelete}, t)

	if m.View() == "" {
		t.Error("View() should not be empty after deleting with no sessions")
	}
}

// =============================================================================
// Smoke test: View() never empty
// =============================================================================

func TestViewNeverEmpty(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
	}{
		{
			name:  "fresh model",
			setup: func(m *Model) {},
		},
		{
			name: "with sessions loaded",
			setup: func(m *Model) {
				m.sessions = []session.Session{{ID: "1", Name: "Test", Model: "gemma-4-31b-it"}}
			},
		},
		{
			name: "with chat session",
			setup: func(m *Model) {
				m.chatSession = session.NewSession("gemma-4-31b-it")
				m.chatSession.AddMessage("user", "Hello")
				m.chatMessages = m.chatSession.Messages
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
				m.chatLoading = true
			},
		},
		{
			name: "sessions state",
			setup: func(m *Model) {
				m.state = stateSessions
				m.sessions = []session.Session{{ID: "1", Name: "Test", Model: "gemma-4-31b-it"}}
			},
		},
		{
			name: "sessions state empty",
			setup: func(m *Model) {
				m.state = stateSessions
				m.sessions = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := setupTestModel(t)
			m.width = 80
			m.height = 40
			tt.setup(&m)

			view := m.View()
			if view == "" {
				t.Errorf("View() returned empty string in state: %s", tt.name)
			}
		})
	}
}

// =============================================================================
// Edge Case: Empty session list loaded
// =============================================================================

func TestSessionsLoadedEmptyList(t *testing.T) {
	m := setupTestModel(t)

	// Nil sessions
	m = updateAndDrain(m, sessionsLoadedMsg{sessions: nil}, t)
	if len(m.sessions) != 0 {
		t.Errorf("expected 0 sessions with nil, got %d", len(m.sessions))
	}

	// Empty slice
	m = updateAndDrain(m, sessionsLoadedMsg{sessions: []session.Session{}}, t)
	if len(m.sessions) != 0 {
		t.Errorf("expected 0 sessions with empty slice, got %d", len(m.sessions))
	}
}

// =============================================================================
// Edge Case: Session select out of bounds
// =============================================================================

func TestSessionSelectOutOfBounds(t *testing.T) {
	m := setupTestModel(t)
	m.state = stateSessions
	m.sessions = nil
	m.cursor = 5 // Out of bounds

	// Should not crash
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnter}, t)

	// State should still be sessions (selection failed)
	if m.state != stateSessions {
		t.Errorf("state should remain sessions when select fails, got %d", m.state)
	}
}

// =============================================================================
// Edge Case: Zero-width terminal
// =============================================================================

func TestZeroWidthTerminal(t *testing.T) {
	m := setupTestModel(t)
	m.width = 0
	m.height = 0

	// Should not crash — width-based calculations should use 1 as minimum
	view := m.View()
	if view == "" {
		t.Error("View() returned empty with zero dimensions")
	}
}

// =============================================================================
// Edge Case: Provide with stream error (not initialization error)
// =============================================================================

func TestSendChatCmdWithStreamError(t *testing.T) {
	store := session.NewStore(t.TempDir(), 10)
	router := provider.NewRouter()
	router.Register(&mockTUIProvider{
		name:            "google",
		supportedModels: []string{"gemma-4-31b-it"},
		streamTokens: []provider.StreamToken{
			{Token: "partial"},
			{Err: errors.New("connection lost")},
		},
	})

	m := NewModel(store, router)
	m.width = 80
	m.height = 40

	// Type and send — the mock stream errors immediately, so drain processes the full cycle
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")}, t)

	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnter}, t)

	// After full drain, loading should be false (stream error was processed)
	if m.chatLoading {
		t.Error("chatLoading should be false after stream error is processed")
	}
	// Error should be set
	if m.err == nil {
		t.Error("err should be set after stream error")
	}
}

// =============================================================================
// Edge Case: Toggle model updates chat session
// =============================================================================

func TestToggleModelUpdatesChatSession(t *testing.T) {
	m := setupTestModel(t)
	m.chatSession = session.NewSession("gemma-4-31b-it")
	m.chatSession.Model = m.currentModel

	m.chatSession.AddMessage("user", "test")

	// Toggle model
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyCtrlK}, t)

	if m.currentModel != "gemma-4-26b-a4b-it" {
		t.Errorf("model should toggle to gemma-4-26b-a4b-it, got %q", m.currentModel)
	}
	if m.chatSession.Model != "gemma-4-26b-a4b-it" {
		t.Errorf("chatSession.Model should update, got %q", m.chatSession.Model)
	}
}

