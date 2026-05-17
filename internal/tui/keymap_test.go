package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFocusAreaConstants(t *testing.T) {
	// Verify FocusArea constants are distinct
	if FocusInput == FocusSidebar {
		t.Error("FocusInput and FocusSidebar should be distinct")
	}
	if FocusInput == FocusChat {
		t.Error("FocusInput and FocusChat should be distinct")
	}
	if FocusSidebar == FocusChat {
		t.Error("FocusSidebar and FocusChat should be distinct")
	}
}

func TestHandleKeyQuit(t *testing.T) {
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyCtrlC}, FocusInput)
	if cmd == nil {
		t.Error("HandleKey(Ctrl+C) should return a quit command")
	}
}

func TestHandleKeyTabCyclesForward(t *testing.T) {
	// Tab from Input → Sidebar
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyTab}, FocusInput)
	if cmd == nil {
		t.Error("HandleKey(Tab, Input) should return focus change command")
	}
}

func TestHandleKeyShiftTabCyclesBackward(t *testing.T) {
	// Shift+Tab from Input → Chat
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyShiftTab}, FocusInput)
	if cmd == nil {
		t.Error("HandleKey(Shift+Tab, Input) should return focus change command")
	}
}

func TestHandleKeyEnterFocusInput(t *testing.T) {
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyEnter}, FocusInput)
	if cmd == nil {
		t.Error("HandleKey(Enter, Input) should return send message command")
	}
}

func TestHandleKeyEscFocusSidebar(t *testing.T) {
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyEsc}, FocusSidebar)
	if cmd == nil {
		t.Error("HandleKey(Esc, Sidebar) should return focus input command")
	}
}

func TestHandleKeyUpDownFocusSidebar(t *testing.T) {
	// Up/Down in sidebar are handled directly by the list component,
	// so HandleKey returns nil (sidebar's Update will handle them)
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyUp}, FocusSidebar)
	// nil is acceptable — the sidebar component handles navigation internally
	if cmd != nil {
		t.Log("HandleKey(Up, Sidebar) returned a command (also valid)")
	}
}

func TestHandleKeyPageUpPageDownFocusChat(t *testing.T) {
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyPgUp}, FocusChat)
	if cmd == nil {
		t.Error("HandleKey(PgUp, Chat) should return a scroll command")
	}
}

func TestHandleKeyCtrlNFocusAny(t *testing.T) {
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyCtrlN}, FocusInput)
	if cmd == nil {
		t.Error("HandleKey(Ctrl+N) should return new session command")
	}
}

func TestHandleKeyCtrlDFocusAny(t *testing.T) {
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyCtrlD}, FocusInput)
	if cmd == nil {
		t.Error("HandleKey(Ctrl+D) should return delete session command")
	}
}

func TestHandleKeyCtrlKFocusAny(t *testing.T) {
	cmd := HandleKey(tea.KeyMsg{Type: tea.KeyCtrlK}, FocusInput)
	if cmd == nil {
		t.Error("HandleKey(Ctrl+K) should return toggle model command")
	}
}
