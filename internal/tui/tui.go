package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/provider"
	"gojobs/internal/session"
)

// tuiState represents the current view state in the TUI state machine.
type tuiState int

const (
	stateChat     tuiState = iota // default: chat with AI
	stateSessions                 // session list overlay
)

// Model is the main Bubbletea model for the gojobs TUI.
type Model struct {
	// State machine
	state tuiState

	// Dependencies
	sessionStore   *session.Store
	providerRouter *provider.Router

	// UI
	spinner spinner.Model
	width   int
	height  int
	err     error

	// Chat state
	chatSession  *session.Session
	chatMessages []session.Message // cached from chatSession
	chatInput    string
	chatLoading  bool
	chatScroll   int // 0 = bottom, positive = scroll up
	currentModel string
	profilePath  string // path to candidate profile JSON

	// Sessions state
	sessions []session.Session
	cursor   int
}

// NewModel creates a new TUI Model with the given dependencies.
func NewModel(store *session.Store, router *provider.Router, profilePath string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	return Model{
		state:          stateChat,
		sessionStore:   store,
		providerRouter: router,
		spinner:        sp,
		currentModel:   "gemma-4-31b-it",
		profilePath:    profilePath,
	}
}

// Init returns the initial commands: spinner tick + load sessions.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadSessionsCmd(m.sessionStore))
}

// Update handles messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit: Ctrl+C always works
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Paste handling in chat state
		if m.state == stateChat && msg.Paste {
			return m.handleChatTextInput(msg)
		}

		// Text input in chat state: route to chat text handler
		if m.state == stateChat {
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace {
				return m.handleChatTextInput(msg)
			}
		}

		// All other keys: dispatch via state-aware handler
		if m.state == stateSessions {
			return m.handleSessionsKey(msg)
		}
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case chatResponseMsg:
		return m.handleChatResponse(msg)

	case sessionsLoadedMsg:
		return m.handleSessionsLoaded(msg)

	case spinner.TickMsg:
		if m.chatLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	default:
		return m, nil
	}
}

// View renders the current TUI view based on state.
func (m Model) View() string {
	switch m.state {
	case stateChat:
		return m.viewChat()
	case stateSessions:
		return m.viewSessions()
	default:
		return "Cargando..."
	}
}

// loadSessionsCmd returns a command that loads sessions from the store.
func loadSessionsCmd(store *session.Store) tea.Cmd {
	return func() tea.Msg {
		sessions, err := store.List()
		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}
