package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerModel wraps a spinner.Model for the loading indicator.
type SpinnerModel struct {
	spinner spinner.Model
	running bool
}

// NewSpinner creates a new SpinnerModel with a "Thinking..." message.
func NewSpinner() SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af"))

	return SpinnerModel{spinner: s}
}

// Start starts the spinner animation.
func (sm *SpinnerModel) Start() {
	sm.running = true
}

// Stop stops the spinner animation.
func (sm *SpinnerModel) Stop() {
	sm.running = false
}

// Update delegates to the underlying spinner.Model.
func (sm SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
	var cmd tea.Cmd
	sm.spinner, cmd = sm.spinner.Update(msg)
	return sm, cmd
}

// View renders the spinner with "Thinking..." label, or empty when stopped.
func (sm SpinnerModel) View() string {
	if sm.running {
		return sm.spinner.View() + " Thinking..."
	}
	return ""
}

// Tick returns the spinner tick message for animation.
func (sm SpinnerModel) Tick() tea.Msg {
	return sm.spinner.Tick()
}
