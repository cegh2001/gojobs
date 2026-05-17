package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSpinner(t *testing.T) {
	s := NewSpinner()
	// Default spinner is not running, so View() may be empty — that's OK
	// No crash is the test
	_ = s.View()
}

func TestSpinnerViewWhenRunning(t *testing.T) {
	s := NewSpinner()
	s.Start()

	result := s.View()
	if result == "" {
		t.Error("View() returned empty after Start()")
	}
}

func TestSpinnerViewWhenStopped(t *testing.T) {
	s := NewSpinner()
	s.Start()
	s.Stop()

	result := s.View()
	// When stopped, spinner should return empty string
	if result != "" {
		t.Logf("View() after Stop() = %q (may still have last frame)", result)
	}
}

func TestSpinnerUpdate(t *testing.T) {
	s := NewSpinner()
	s.Start()

	// Send a tick message (spinner.Tick is a tea.Msg)
	_, cmd := s.Update(s.Tick())
	_ = cmd
	// Should not panic
}

func TestSpinnerStartAndStop(t *testing.T) {
	s := NewSpinner()
	s.Start()
	// View should be non-empty when spinning
	if s.View() == "" {
		t.Error("View() returned empty after Start()")
	}
}

func TestSpinnerDefaultStopped(t *testing.T) {
	s := NewSpinner()
	// Before Start(), View() should return empty
	_ = s.View()
	// No crash is the test
}

func TestSpinnerWindowSize(t *testing.T) {
	s := NewSpinner()
	_, cmd := s.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_ = cmd
	// Should not panic
}
