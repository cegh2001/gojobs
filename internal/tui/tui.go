package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gojobs/internal/provider"
	"gojobs/internal/session"
	"gojobs/internal/tui/components"
)

// Model is the main BubbleTea model for the TUI application.
type Model struct {
	sessionStore   *session.Store
	providerRouter *provider.Router
	sessions       []session.Session
	currentSession *session.Session
	currentModel   string
	chatView       components.ChatViewModel
	inputArea      components.InputAreaModel
	sidebar        components.SidebarModel
	spinner        components.SpinnerModel
	focus          FocusArea
	loading        bool
	width          int
	height         int
	err            error
}

// NewModel creates a new TUI Model with the given dependencies.
func NewModel(store *session.Store, router *provider.Router) Model {
	return Model{
		sessionStore:   store,
		providerRouter: router,
		currentModel:   "gemma-4-31b-it",
		chatView:       components.NewChatView(),
		inputArea:      components.NewInputArea(),
		sidebar:        components.NewSidebar(),
		spinner:        components.NewSpinner(),
		focus:          FocusInput,
	}
}

// Init returns the initial commands for the TUI.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadSessionsCmd(m.sessionStore),
		tea.EnterAltScreen,
		func() tea.Msg { return tea.EnableMouseCellMotion() },
	)
}

// Update handles messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case StreamTokenMsg:
		m.chatView.AppendToken(msg.Token)
		return m, nil

	case StreamDoneMsg:
		m.loading = false
		m.spinner.Stop()
		m.chatView.AppendMessage("assistant", msg.Content)
		if m.currentSession != nil {
			m.currentSession.AddMessage("assistant", msg.Content)
			_ = m.sessionStore.Save(m.currentSession)
		}
		return m, nil

	case StreamErrMsg:
		m.loading = false
		m.spinner.Stop()
		m.err = msg.Err
		return m, nil

	case SessionsLoadedMsg:
		m.sessions = msg.Sessions
		m.sidebar.SetSessions(msg.Sessions)
		return m, nil

	case SessionSelectedMsg:
		m.currentSession = msg.Session
		if msg.Session != nil {
			m.chatView.LoadHistory(msg.Session.Messages)
			m.currentModel = msg.Session.Model
		}
		return m, nil

	case ModelSelectedMsg:
		m.currentModel = msg.Model
		return m, nil

	case FocusChangeMsg:
		m.focus = msg.Area
		return m, nil

	case SendMessageMsg:
		// Ignore sends while already processing a request
		if m.loading {
			return m, nil
		}
		// Ignore empty input
		if strings.TrimSpace(m.inputArea.Value()) == "" {
			return m, nil
		}
		if m.currentSession == nil {
			m.currentSession = session.NewSession(m.currentModel)
			m.sessions = append(m.sessions, *m.currentSession)
			m.sidebar.SetSessions(m.sessions)
		}
		m.currentSession.AddMessage("user", m.inputArea.Value())
		m.chatView.AppendMessage("user", m.inputArea.Value())
		m.inputArea.Reset()
		m.loading = true
		m.spinner.Start()
		return m, nil

	case NewSessionMsg:
		m.currentSession = session.NewSession(m.currentModel)
		m.chatView.Clear()
		m.inputArea.Reset()
		m.sessions = append([]session.Session{*m.currentSession}, m.sessions...)
		m.sidebar.SetSessions(m.sessions)
		m.loading = false
		m.spinner.Stop()
		return m, nil

	case DeleteSessionMsg:
		if m.currentSession != nil {
			_ = m.sessionStore.Delete(m.currentSession.ID)
			for i, s := range m.sessions {
				if s.ID == m.currentSession.ID {
					m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
					break
				}
			}
			if len(m.sessions) > 0 {
				m.currentSession = &m.sessions[0]
				m.chatView.LoadHistory(m.currentSession.Messages)
			} else {
				m.currentSession = nil
				m.chatView.Clear()
			}
			m.sidebar.SetSessions(m.sessions)
		}
		return m, nil

	case ToggleModelMsg:
		if m.currentModel == "gemma-4-31b-it" {
			m.currentModel = "gemma-4-26b-a4b-it"
		} else {
			m.currentModel = "gemma-4-31b-it"
		}
		if m.currentSession != nil {
			m.currentSession.Model = m.currentModel
		}
		return m, nil

	case ScrollUpMsg:
		// Scroll viewport up handled by chat view
		return m, nil

	case ScrollDownMsg:
		// Scroll viewport down handled by chat view
		return m, nil
	}

	return m, nil
}

// View renders the complete TUI layout.
func (m Model) View() string {
	header := HeaderStyle.Render(fmt.Sprintf("gojobs — %s", m.currentModel))

	sidebarView := SidebarStyle.Width(m.sidebarWidth()).Height(m.contentHeight()).Render(m.sidebar.View())

	chatHeight := m.contentHeight() - 5 // 5 lines for input area
	m.chatView.SetSize(m.chatWidth(), chatHeight)

	var loadingIndicator string
	if m.loading {
		loadingIndicator = m.spinner.View()
	}

	chatContent := lipgloss.JoinVertical(lipgloss.Left,
		m.chatView.View(),
		loadingIndicator,
	)

	inputView := InputStyle.Width(m.chatWidth()).Render(m.inputArea.View())

	mainContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Height(chatHeight).MaxHeight(chatHeight).Render(chatContent),
		inputView,
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebarView,
		mainContent,
	)

	footer := HelpStyle.Render("Tab: switch focus | Enter: send | Ctrl+C: quit | Ctrl+N: new | Ctrl+D: delete | Ctrl+K: toggle model")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		footer,
	)
}

// handleKeyMsg dispatches key messages based on current focus.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for custom key bindings first
	cmd := HandleKey(msg, m.focus)
	if cmd != nil {
		return m, cmd
	}

	// Delegate to focused component
	switch m.focus {
	case FocusInput:
		var c tea.Cmd
		m.inputArea, c = m.inputArea.Update(msg)
		return m, c
	case FocusSidebar:
		var c tea.Cmd
		m.sidebar, c = m.sidebar.Update(msg)
		return m, c
	case FocusChat:
		var c tea.Cmd
		m.chatView, c = m.chatView.Update(msg)
		return m, c
	}

	return m, nil
}

// sidebarWidth calculates the width of the sidebar based on total width.
func (m Model) sidebarWidth() int {
	if m.width < 100 {
		return 0
	}
	return m.width * 20 / 100
}

// chatWidth calculates the width of the chat area.
func (m Model) chatWidth() int {
	return m.width - m.sidebarWidth()
}

// contentHeight calculates the available height for the main content area.
func (m Model) contentHeight() int {
	if m.height < 3 {
		return 0
	}
	return m.height - 2 // header + footer
}

// loadSessionsCmd returns a command that loads sessions from the store.
func loadSessionsCmd(store *session.Store) tea.Cmd {
	return func() tea.Msg {
		sessions, err := store.List()
		if err != nil {
			return StreamErrMsg{Err: err}
		}
		return SessionsLoadedMsg{Sessions: sessions}
	}
}
