package components

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/session"
)

// sessionItem is a list.Item implementation for session entries.
type sessionItem struct {
	id    string
	name  string
	model string
}

func (s sessionItem) Title() string       { return s.name }
func (s sessionItem) Description() string { return fmt.Sprintf("Model: %s", s.model) }
func (s sessionItem) FilterValue() string { return s.name }

// sidebarDelegate is the default list item delegate.
type sidebarDelegate struct{}

func (d sidebarDelegate) Height() int                               { return 1 }
func (d sidebarDelegate) Spacing() int                              { return 0 }
func (d sidebarDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d sidebarDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	sit, ok := item.(sessionItem)
	if !ok {
		return
	}
	var line string
	if index == m.Index() {
		line = fmt.Sprintf("> %s", sit.Title())
	} else {
		line = fmt.Sprintf("  %s", sit.Title())
	}
	fmt.Fprint(w, line)
}

// newSessionItem represents the "New Session" action.
type newSessionItem struct{}

func (n newSessionItem) Title() string       { return "[New Session]" }
func (n newSessionItem) Description() string { return "Create a new chat session" }
func (n newSessionItem) FilterValue() string { return "[New Session]" }

// modelSelectorItem represents the model toggle action.
type modelSelectorItem struct {
	model string
}

func (m modelSelectorItem) Title() string       { return fmt.Sprintf("Model: %s ▼", m.model) }
func (m modelSelectorItem) Description() string { return "Press Enter to toggle model" }
func (m modelSelectorItem) FilterValue() string { return m.model }

// SidebarModel wraps a list.Model for session navigation and model selection.
type SidebarModel struct {
	list     list.Model
	sessions []session.Session
	model    string
}

// NewSidebar creates a new SidebarModel with default settings.
func NewSidebar() SidebarModel {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 20, 20)
	l.Title = "Sessions"

	return SidebarModel{
		list:  l,
		model: "gemma-4-31b-it",
	}
}

// SetSessions populates the sidebar with sessions.
func (sb *SidebarModel) SetSessions(sessions []session.Session) {
	sb.sessions = sessions

	var items []list.Item
	items = append(items, newSessionItem{})
	items = append(items, modelSelectorItem{model: sb.model})
	for _, s := range sessions {
		items = append(items, sessionItem{id: s.ID, name: s.Name, model: s.Model})
	}

	sb.list.SetItems(items)
}

// SelectedSession returns the currently selected session, or nil.
func (sb SidebarModel) SelectedSession() *session.Session {
	idx := sb.list.Index()
	// Skip [New Session] (0) and model selector (1)
	sessIdx := idx - 2
	if sessIdx < 0 || sessIdx >= len(sb.sessions) {
		return nil
	}
	return &sb.sessions[sessIdx]
}

// CurrentModel returns the current model selection.
func (sb SidebarModel) CurrentModel() string {
	return sb.model
}

// Update delegates to the underlying list.Model.
func (sb SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	var cmd tea.Cmd
	sb.list, cmd = sb.list.Update(msg)
	return sb, cmd
}

// View renders the sidebar.
func (sb SidebarModel) View() string {
	return sb.list.View()
}
