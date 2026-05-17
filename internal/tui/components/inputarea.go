package components

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// InputAreaModel wraps a textarea.Model for user input.
type InputAreaModel struct {
	textarea textarea.Model
}

// NewInputArea creates a new InputAreaModel with a placeholder prompt.
func NewInputArea() InputAreaModel {
	ta := textarea.New()
	ta.Placeholder = "Type a message or paste a job URL..."
	ta.SetHeight(5)
	ta.ShowLineNumbers = false
	ta.CharLimit = 0

	return InputAreaModel{textarea: ta}
}

// Value returns the current text content.
func (ia InputAreaModel) Value() string {
	return ia.textarea.Value()
}

// Reset clears the textarea content.
func (ia *InputAreaModel) Reset() {
	ia.textarea.Reset()
}

// Focus focuses the textarea.
func (ia *InputAreaModel) Focus() {
	ia.textarea.Focus()
}

// Blur removes focus from the textarea.
func (ia *InputAreaModel) Blur() {
	ia.textarea.Blur()
}

// Update delegates to the underlying textarea.Model.
func (ia InputAreaModel) Update(msg tea.Msg) (InputAreaModel, tea.Cmd) {
	var cmd tea.Cmd
	ia.textarea, cmd = ia.textarea.Update(msg)
	return ia, cmd
}

// View renders the textarea.
func (ia InputAreaModel) View() string {
	return ia.textarea.View()
}
