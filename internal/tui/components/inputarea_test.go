package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewInputArea(t *testing.T) {
	ia := NewInputArea()
	if ia.View() == "" {
		t.Error("NewInputArea().View() returned empty string")
	}
}

func TestInputAreaValue(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	// Simulate typing by sending key messages — capture the returned model
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	val := ia.Value()
	if val != "Hello" {
		t.Errorf("Value() = %q, want %q", val, "Hello")
	}
}

func TestInputAreaReset(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	ia, _ = ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	ia.Reset()
	if ia.Value() != "" {
		t.Errorf("Value() after Reset() = %q, want empty", ia.Value())
	}
}

func TestInputAreaFocusAndBlur(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia.Blur()
	// Verify View() still works after focus/blur
	if ia.View() == "" {
		t.Error("View() returned empty after Focus/Blur")
	}
}

func TestInputAreaViewContainsPlaceholder(t *testing.T) {
	ia := NewInputArea()
	result := ia.View()
	if !strings.Contains(result, "Type a message") && !strings.Contains(result, "job URL") {
		t.Errorf("View() does not contain placeholder text: %s", result)
	}
}

func TestInputAreaUpdate(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia, cmd := ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// textarea Update may return a command (e.g., for blinking cursor)
	_ = cmd
	if ia.Value() != "a" {
		t.Errorf("Value() after key press = %q, want %q", ia.Value(), "a")
	}
}
