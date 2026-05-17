package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/provider"
	"gojobs/internal/session"
)

func setupTestModel(t *testing.T) Model {
	t.Helper()

	tempDir := filepath.Join(os.TempDir(), "gojobs-tui-test-"+t.Name())
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	store := session.NewStore(tempDir, 10)
	router := provider.NewRouter()

	return NewModel(store, router)
}

// updateAndDrain executes the returned command and feeds the result back as a message.
// This simulates how the BubbleTea runtime processes commands.
func updateAndDrain(m Model, msg tea.Msg, t *testing.T) Model {
	t.Helper()
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)
	// Execute up to 3 commands (batching, etc.)
	for i := 0; i < 3 && cmd != nil; i++ {
		nextMsg := cmd()
		if nextMsg == nil {
			break
		}
		newModel, nextCmd := m.Update(nextMsg)
		m = newModel.(Model)
		cmd = nextCmd
	}
	return m
}

func TestNewModel(t *testing.T) {
	m := setupTestModel(t)
	if m.View() == "" {
		t.Error("NewModel().View() returned empty string")
	}
}

func TestInitReturnsCommands(t *testing.T) {
	m := setupTestModel(t)
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil, expected commands")
	}
}

func TestWindowSizeMsgUpdatesDimensions(t *testing.T) {
	m := setupTestModel(t)
	m = updateAndDrain(m, tea.WindowSizeMsg{Width: 120, Height: 40}, t)

	// Verify View() is non-empty after resize
	if m.View() == "" {
		t.Error("View() returned empty after WindowSizeMsg")
	}
}

func TestFocusTabCycling(t *testing.T) {
	m := setupTestModel(t)

	// Default focus should be FocusInput
	if m.focus != FocusInput {
		t.Errorf("default focus = %d, want FocusInput (%d)", m.focus, FocusInput)
	}

	// Tab should cycle to Sidebar
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyTab}, t)
	if m.focus != FocusSidebar {
		t.Errorf("focus after Tab = %d, want FocusSidebar (%d)", m.focus, FocusSidebar)
	}

	// Tab again should cycle to Chat
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyTab}, t)
	if m.focus != FocusChat {
		t.Errorf("focus after 2nd Tab = %d, want FocusChat (%d)", m.focus, FocusChat)
	}

	// Tab again should cycle back to Input
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyTab}, t)
	if m.focus != FocusInput {
		t.Errorf("focus after 3rd Tab = %d, want FocusInput (%d)", m.focus, FocusInput)
	}
}

func TestShiftTabCycling(t *testing.T) {
	m := setupTestModel(t)

	// Shift+Tab from Input should go to Chat
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyShiftTab}, t)
	if m.focus != FocusChat {
		t.Errorf("focus after Shift+Tab = %d, want FocusChat (%d)", m.focus, FocusChat)
	}
}

func TestStreamTokenMsgAppendsToChat(t *testing.T) {
	m := setupTestModel(t)

	// Send a token
	m = updateAndDrain(m, StreamTokenMsg{Token: "Hello"}, t)

	// View should be non-empty
	if m.View() == "" {
		t.Error("View() returned empty after StreamTokenMsg")
	}
}

func TestStreamDoneMsgFinalizes(t *testing.T) {
	m := setupTestModel(t)

	// First add a user message to the current session
	m.currentSession = session.NewSession("gemma-4-31b-it")
	m.currentSession.AddMessage("user", "Test message")

	// Send done
	m = updateAndDrain(m, StreamDoneMsg{Content: "Final content"}, t)

	// Loading should be false
	if m.loading {
		t.Error("loading should be false after StreamDoneMsg")
	}

	// View should be non-empty
	if m.View() == "" {
		t.Error("View() returned empty after StreamDoneMsg")
	}
}

func TestStreamErrMsgShowsError(t *testing.T) {
	m := setupTestModel(t)
	m.loading = true

	m = updateAndDrain(m, StreamErrMsg{Err: os.ErrNotExist}, t)

	if m.loading {
		t.Error("loading should be false after StreamErrMsg")
	}
	if m.err == nil {
		t.Error("err should be set after StreamErrMsg")
	}
}

func TestSessionsLoadedMsgPopulatesSidebar(t *testing.T) {
	m := setupTestModel(t)

	sessions := []session.Session{
		{ID: "s1", Name: "Chat 1", Model: "gemma-4-31b-it"},
	}

	m = updateAndDrain(m, SessionsLoadedMsg{Sessions: sessions}, t)

	if len(m.sessions) != 1 {
		t.Errorf("expected 1 session after SessionsLoadedMsg, got %d", len(m.sessions))
	}
}

func TestSelectModelMsgUpdatesCurrentModel(t *testing.T) {
	m := setupTestModel(t)

	m = updateAndDrain(m, ModelSelectedMsg{Model: "gemma-4-26b-a4b-it"}, t)

	if m.currentModel != "gemma-4-26b-a4b-it" {
		t.Errorf("currentModel = %q, want %q", m.currentModel, "gemma-4-26b-a4b-it")
	}
}
