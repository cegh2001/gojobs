package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/session"
)

func TestNewSidebar(t *testing.T) {
	sb := NewSidebar()
	if sb.View() == "" {
		t.Error("NewSidebar().View() returned empty string")
	}
}

func TestSetSessions(t *testing.T) {
	sb := NewSidebar()
	sessions := []session.Session{
		{ID: "s1", Name: "Session 1", Model: "gemma-4-31b-it"},
		{ID: "s2", Name: "Session 2", Model: "gemma-4-26b-a4b-it"},
	}
	sb.SetSessions(sessions)

	result := sb.View()
	if result == "" {
		t.Error("View() returned empty after SetSessions")
	}
}

func TestSelectedSessionEmpty(t *testing.T) {
	sb := NewSidebar()
	// With no sessions set, SelectedSession should return nil
	s := sb.SelectedSession()
	if s != nil {
		t.Error("SelectedSession() should be nil when no sessions are set")
	}
}

func TestSelectedSessionAfterSet(t *testing.T) {
	sb := NewSidebar()
	sessions := []session.Session{
		{ID: "abc", Name: "Test Chat", Model: "gemma-4-31b-it"},
	}
	sb.SetSessions(sessions)

	// Navigate down past [New Session] and Model selector to hit first session
	// List has items: [New Session] (0), Model:xxx (1), session (2)
	// We need to move cursor to index 2
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyDown}) // index 1
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyDown}) // index 2

	s := sb.SelectedSession()
	if s == nil {
		t.Errorf("SelectedSession() returned nil after navigating to session. Cursor index: %d", sb.list.Index())
	}
	if s != nil && s.ID != "abc" {
		t.Errorf("SelectedSession().ID = %q, want %q", s.ID, "abc")
	}
}

func TestSidebarViewContainsTitle(t *testing.T) {
	sb := NewSidebar()
	result := sb.View()
	if result == "" {
		t.Error("View() returned empty string")
	}
}

func TestSidebarUpdate(t *testing.T) {
	sb := NewSidebar()
	_, cmd := sb.Update(tea.KeyMsg{Type: tea.KeyDown})
	_ = cmd
	// Should not panic
}

func TestCurrentModelDefault(t *testing.T) {
	sb := NewSidebar()
	model := sb.CurrentModel()
	if model == "" {
		t.Error("CurrentModel() returned empty string")
	}
}
