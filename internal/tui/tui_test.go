package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/provider"
	"gojobs/internal/session"
)

// setupTestModel creates a model with temp session store and empty router.
func setupTestModel(t *testing.T) Model {
	t.Helper()

	tempDir := filepath.Join(os.TempDir(), "gojobs-tui-test-"+t.Name())
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	store := session.NewStore(tempDir, 10)
	router := provider.NewRouter()

	return NewModel(store, router, "profiles/carlos_gonzalez.json")
}

// updateAndDrain executes the returned command and feeds the result back as a message.
func updateAndDrain(m Model, msg tea.Msg, t *testing.T) Model {
	t.Helper()
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)
	// Execute up to 5 commands (batching, load sessions, spinner tick, etc.)
	for i := 0; i < 5 && cmd != nil; i++ {
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
	if m.state != stateChat {
		t.Errorf("default state = %d, want stateChat (%d)", m.state, stateChat)
	}
	if m.currentModel != "gemma-4-31b-it" {
		t.Errorf("default model = %q, want %q", m.currentModel, "gemma-4-31b-it")
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

func TestCtrlCQuits(t *testing.T) {
	m := setupTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Ctrl+C should return a quit command")
	}
}

// =============================================================================
// State Machine Tests
// =============================================================================

func TestStateMachineEscTogglesToSessions(t *testing.T) {
	m := setupTestModel(t)
	m.sessions = []session.Session{
		{ID: "s1", Name: "Chat 1", Model: "gemma-4-31b-it"},
	}

	// Esc from chat → sessions
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEsc}, t)
	if m.state != stateSessions {
		t.Errorf("state after Esc from chat = %d, want stateSessions (%d)", m.state, stateSessions)
	}

	// Esc from sessions → back to chat
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEsc}, t)
	if m.state != stateChat {
		t.Errorf("state after Esc from sessions = %d, want stateChat (%d)", m.state, stateChat)
	}
}

func TestStateMachineEnterSendsInChat(t *testing.T) {
	m := setupTestModel(t)
	// Type text
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Hello world")}
	m = updateAndDrain(m, msg, t)

	if m.chatInput != "Hello world" {
		t.Fatalf("chatInput = %q, want %q", m.chatInput, "Hello world")
	}

	// Enter with empty router — should show error inline but not crash
	// Creates new session, adds user message, fails on provider resolution
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnter}, t)

	// chatInput should be cleared
	if m.chatInput != "" {
		t.Errorf("chatInput should be empty after send, got %q", m.chatInput)
	}

	// loading should be false (error handled immediately)
	if m.chatLoading {
		t.Error("loading should be false after failed send")
	}

	// Should have created a session
	if m.chatSession == nil {
		t.Fatal("chatSession should be created after send")
	}

	// Should have user message + error message
	if len(m.chatSession.Messages) < 1 {
		t.Errorf("expected at least 1 message, got %d", len(m.chatSession.Messages))
	}
}

func TestStateMachineEnterSelectsInSessions(t *testing.T) {
	m := setupTestModel(t)

	// Create a session and save it
	store := m.sessionStore
	sess := session.NewSession("gemma-4-31b-it")
	sess.AddMessage("user", "Test message")
	_ = store.Save(sess)

	// Load sessions
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEsc}, t) // go to sessions
	if len(m.sessions) < 1 {
		t.Fatal("sessions should be loaded")
	}

	// Select the session
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnter}, t)

	// Should be back in chat state with messages loaded
	if m.state != stateChat {
		t.Errorf("state after session select = %d, want stateChat (%d)", m.state, stateChat)
	}
	if m.chatSession == nil {
		t.Fatal("chatSession should be set after session select")
	}
	if m.chatSession.ID != sess.ID {
		t.Errorf("selected session ID = %q, want %q", m.chatSession.ID, sess.ID)
	}
}

// =============================================================================
// Chat Scroll Tests
// =============================================================================

func TestChatScrollUpDown(t *testing.T) {
	m := setupTestModel(t)
	m.width = 80
	m.height = 40

	// Add enough messages to require scrolling
	m.chatSession = session.NewSession("gemma-4-31b-it")
	for i := 0; i < 50; i++ {
		m.chatSession.AddMessage("user", "Line "+string(rune('A'+i%26)))
	}
	m.chatMessages = m.chatSession.Messages

	// Scroll up
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyUp}, t)
	if m.chatScroll <= 0 {
		t.Error("chatScroll should be > 0 after Up key")
	}

	// Scroll down
	prevScroll := m.chatScroll
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyDown}, t)
	if m.chatScroll >= prevScroll {
		t.Error("chatScroll should decrease after Down key")
	}
}

func TestChatScrollHomeEnd(t *testing.T) {
	m := setupTestModel(t)
	m.width = 80
	m.height = 40

	m.chatSession = session.NewSession("gemma-4-31b-it")
	for i := 0; i < 50; i++ {
		m.chatSession.AddMessage("user", "Line "+string(rune('A'+i%26)))
	}
	m.chatMessages = m.chatSession.Messages

	// Home: jump to top
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyHome}, t)
	maxScroll := m.maxChatScroll()
	if m.chatScroll != maxScroll {
		t.Errorf("chatScroll after Home = %d, want maxScroll (%d)", m.chatScroll, maxScroll)
	}

	// End: jump to bottom
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnd}, t)
	if m.chatScroll != 0 {
		t.Errorf("chatScroll after End = %d, want 0", m.chatScroll)
	}
}

// =============================================================================
// Text Input Tests
// =============================================================================

func TestChatTextInputAccumulates(t *testing.T) {
	m := setupTestModel(t)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Hello")}
	m = updateAndDrain(m, msg, t)
	if m.chatInput != "Hello" {
		t.Errorf("chatInput = %q, want %q", m.chatInput, "Hello")
	}

	msg2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" World")}
	m = updateAndDrain(m, msg2, t)
	if m.chatInput != "Hello World" {
		t.Errorf("chatInput = %q, want %q", m.chatInput, "Hello World")
	}
}

func TestChatTextInputBackspace(t *testing.T) {
	m := setupTestModel(t)

	// Type text first
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Hello")}, t)
	// Backspace
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyBackspace}, t)
	if m.chatInput != "Hell" {
		t.Errorf("chatInput after backspace = %q, want %q", m.chatInput, "Hell")
	}

	// Backspace all the way
	for i := 0; i < 4; i++ {
		m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyBackspace}, t)
	}
	if m.chatInput != "" {
		t.Errorf("chatInput after all backspaces = %q, want empty", m.chatInput)
	}
}

func TestChatTextInputPaste(t *testing.T) {
	m := setupTestModel(t)

	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("Hello\r\nWorld"),
		Paste: true,
	}
	m = updateAndDrain(m, msg, t)

	// Pasted input should have newlines replaced with spaces
	if m.chatInput != "Hello World" {
		t.Errorf("paste result = %q, want %q", m.chatInput, "Hello World")
	}
}

// =============================================================================
// Ctrl+N and Ctrl+K Tests
// =============================================================================

func TestCtrlNNewSession(t *testing.T) {
	m := setupTestModel(t)
	// Simulate text in input
	m.chatInput = "some text"
	m.chatLoading = true

	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyCtrlN}, t)

	if m.chatSession == nil {
		t.Fatal("chatSession should be set after Ctrl+N")
	}
	if m.chatInput != "" {
		t.Error("chatInput should be cleared after new session")
	}
	if m.chatLoading {
		t.Error("chatLoading should be false after new session")
	}
	if m.chatScroll != 0 {
		t.Error("chatScroll should be reset after new session")
	}
}

func TestCtrlKToggleModel(t *testing.T) {
	m := setupTestModel(t)

	// Default is gemma-4-31b-it
	if m.currentModel != "gemma-4-31b-it" {
		t.Fatalf("default model = %q, want gemma-4-31b-it", m.currentModel)
	}

	// Toggle to gemma-4-26b-a4b-it
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyCtrlK}, t)
	if m.currentModel != "gemma-4-26b-a4b-it" {
		t.Errorf("model after first toggle = %q, want gemma-4-26b-a4b-it", m.currentModel)
	}

	// Toggle back
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyCtrlK}, t)
	if m.currentModel != "gemma-4-31b-it" {
		t.Errorf("model after second toggle = %q, want gemma-4-31b-it", m.currentModel)
	}
}

// =============================================================================
// Sessions Tests
// =============================================================================

func TestSessionsNavigation(t *testing.T) {
	m := setupTestModel(t)
	m.sessions = []session.Session{
		{ID: "s1", Name: "First"},
		{ID: "s2", Name: "Second"},
		{ID: "s3", Name: "Third"},
	}
	m.state = stateSessions
	m.cursor = 0

	// j moves down
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}, t)
	if m.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", m.cursor)
	}

	// k moves up
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, t)
	if m.cursor != 0 {
		t.Errorf("cursor after k = %d, want 0", m.cursor)
	}

	// Can't go past top
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, t)
	if m.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", m.cursor)
	}

	// Can't go past bottom
	m.cursor = 2
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}, t)
	if m.cursor != 2 {
		t.Errorf("cursor should not go past last item, got %d", m.cursor)
	}
}

func TestSessionDelete(t *testing.T) {
	m := setupTestModel(t)

	// Save a session
	sess := session.NewSession("gemma-4-31b-it")
	sess.AddMessage("user", "test")
	_ = m.sessionStore.Save(sess)

	m.sessions = []session.Session{*sess}
	m.state = stateSessions
	m.cursor = 0

	// Delete
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyDelete}, t)
	if len(m.sessions) != 0 {
		t.Errorf("sessions should be empty after delete, got %d", len(m.sessions))
	}
	if m.cursor != 0 {
		t.Errorf("cursor should be 0 after deleting only item, got %d", m.cursor)
	}
}

func TestSessionDeleteClearsCurrentIfActive(t *testing.T) {
	m := setupTestModel(t)

	sess := session.NewSession("gemma-4-31b-it")
	sess.AddMessage("user", "test")
	_ = m.sessionStore.Save(sess)

	m.chatSession = sess
	m.chatMessages = sess.Messages
	m.sessions = []session.Session{*sess}
	m.state = stateSessions
	m.cursor = 0

	// Delete the current session
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyDelete}, t)
	if m.chatSession != nil {
		t.Error("chatSession should be nil after deleting current session")
	}
	if m.chatMessages != nil {
		t.Error("chatMessages should be nil after deleting current session")
	}
}

// =============================================================================
// Stream Response Tests
// =============================================================================

func TestChatResponseMsgAppendsMessage(t *testing.T) {
	m := setupTestModel(t)
	m.chatSession = session.NewSession("gemma-4-31b-it")
	m.chatLoading = true

	m = updateAndDrain(m, chatResponseMsg{content: "AI response"}, t)

	if m.chatLoading {
		t.Error("chatLoading should be false after chatResponseMsg")
	}
	if m.chatScroll != 0 {
		t.Error("chatScroll should be 0 after response")
	}
	if len(m.chatSession.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.chatSession.Messages))
	}
	if string(m.chatSession.Messages[0].Role) != "assistant" {
		t.Errorf("message role = %q, want assistant", m.chatSession.Messages[0].Role)
	}
	if m.chatSession.Messages[0].Content != "AI response" {
		t.Errorf("message content = %q, want %q", m.chatSession.Messages[0].Content, "AI response")
	}
}

func TestChatResponseMsgHandlesError(t *testing.T) {
	m := setupTestModel(t)
	m.chatSession = session.NewSession("gemma-4-31b-it")
	m.chatLoading = true

	wantErr := errors.New("API error")
	m = updateAndDrain(m, chatResponseMsg{err: wantErr}, t)

	if m.chatLoading {
		t.Error("chatLoading should be false after error response")
	}
	if m.err == nil {
		t.Error("err should be set after error response")
	}
	if len(m.chatSession.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.chatSession.Messages))
	}
}

// =============================================================================
// Sessions Loaded Tests
// =============================================================================

func TestSessionsLoadedMsgPopulatesList(t *testing.T) {
	m := setupTestModel(t)

	sessions := []session.Session{
		{ID: "s1", Name: "Chat 1", Model: "gemma-4-31b-it"},
		{ID: "s2", Name: "Chat 2", Model: "gemma-4-26b-a4b-it"},
	}

	m = updateAndDrain(m, sessionsLoadedMsg{sessions: sessions}, t)

	if len(m.sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(m.sessions))
	}
}

// mockTUIProvider implements provider.Provider for TUI tests.
type mockTUIProvider struct {
	name            string
	supportedModels []string
	streamTokens    []provider.StreamToken // tokens to stream, empty = close immediately
	streamErr       error
}

func (m *mockTUIProvider) Name() string              { return m.name }
func (m *mockTUIProvider) SupportedModels() []string { return m.supportedModels }

func (m *mockTUIProvider) SendMessageStream(_ context.Context, _ string, _ []provider.Message) (<-chan provider.StreamToken, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan provider.StreamToken, len(m.streamTokens))
	go func() {
		defer close(ch)
		for _, token := range m.streamTokens {
			ch <- token
		}
	}()
	return ch, nil
}

func (m *mockTUIProvider) SendMessage(_ context.Context, _ string, _ []provider.Message) (string, error) {
	return "mock response", nil
}

var _ provider.Provider = (*mockTUIProvider)(nil)

func TestSendWithMockProvider(t *testing.T) {
	store := session.NewStore(t.TempDir(), 10)
	router := provider.NewRouter()
	router.Register(&mockTUIProvider{
		name:            "google",
		supportedModels: []string{"gemma-4-31b-it"},
		streamTokens: []provider.StreamToken{
			{Token: "Hello "},
			{Token: "World"},
		},
	})

	m := NewModel(store, router, "profiles/carlos_gonzalez.json")
	m.width = 80
	m.height = 40

	// Type text
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Hi")}, t)

	// Send (Enter) — the mock stream completes immediately, so drain processes the full cycle
	m = updateAndDrain(m, tea.KeyMsg{Type: tea.KeyEnter}, t)

	// After full drain, loading should be false (stream completed and response processed)
	if m.chatLoading {
		t.Error("chatLoading should be false after stream completes")
	}

	// Should have user message + assistant response
	if m.chatSession == nil {
		t.Fatal("chatSession should exist")
	}
	if len(m.chatSession.Messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(m.chatSession.Messages))
	}
	if string(m.chatSession.Messages[1].Role) != "assistant" {
		t.Errorf("second message role = %q, want assistant", m.chatSession.Messages[1].Role)
	}

	// View should be non-empty
	if m.View() == "" {
		t.Error("View() returned empty after full flow")
	}
}
