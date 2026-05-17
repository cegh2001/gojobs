package tui

import tea "github.com/charmbracelet/bubbletea"

// handleKeyMsg dispatches key messages in chat state (non-text keys).
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Chat scroll keys
	switch msg.Type {
	case tea.KeyUp:
		m = m.shiftChatScroll(1)
		return m, nil
	case tea.KeyDown:
		m = m.shiftChatScroll(-1)
		return m, nil
	case tea.KeyPgUp:
		m = m.shiftChatScroll(m.chatPageScrollStep())
		return m, nil
	case tea.KeyPgDown:
		m = m.shiftChatScroll(-m.chatPageScrollStep())
		return m, nil
	case tea.KeyHome:
		m.chatScroll = m.maxChatScroll()
		return m, nil
	case tea.KeyEnd:
		m.chatScroll = 0
		return m, nil
	}

	// String-based key bindings
	switch msg.String() {
	case "esc":
		return m.handleEsc()
	case "enter":
		return m.handleEnter()
	case "ctrl+n":
		return m.handleChatNew()
	case "ctrl+k":
		return m.toggleModel()
	case "pgup", "pageup":
		m = m.shiftChatScroll(m.chatPageScrollStep())
		return m, nil
	case "pgdown", "pagedown":
		m = m.shiftChatScroll(-m.chatPageScrollStep())
		return m, nil
	}

	return m, nil
}

// handleEsc toggles between chat and sessions state.
func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateChat:
		// Chat → Sessions overlay
		m.state = stateSessions
		m.cursor = 0
		// Refresh session list on entry
		return m, loadSessionsCmd(m.sessionStore)
	case stateSessions:
		// Sessions → back to chat
		m.state = stateChat
		m.cursor = 0
		return m, nil
	}
	return m, nil
}

// handleEnter sends in chat, selects in sessions.
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateChat:
		return m.handleChatSend()
	case stateSessions:
		return m.handleSessionSelect()
	}
	return m, nil
}

// handleSessionsKey dispatches key messages in sessions state.
func (m Model) handleSessionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		return m.handleCursorUp()
	case tea.KeyDown:
		return m.handleCursorDown()
	case tea.KeyDelete:
		return m.handleSessionDelete()
	}

	switch msg.String() {
	case "esc":
		return m.handleEsc()
	case "enter":
		return m.handleSessionSelect()
	case "k":
		return m.handleCursorUp()
	case "j":
		return m.handleCursorDown()
	case "delete", "del", "ctrl+d":
		return m.handleSessionDelete()
	}

	return m, nil
}

// handleSessionSelect loads the selected session into the chat view.
func (m Model) handleSessionSelect() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.sessions) {
		return m, nil
	}

	selected := &m.sessions[m.cursor]

	// Load from store to get full messages
	full, err := m.sessionStore.Get(selected.ID)
	if err != nil {
		return m, nil
	}

	m.chatSession = full
	m.chatMessages = full.Messages
	m.chatInput = ""
	m.chatLoading = false
	m.chatScroll = 0
	m.currentModel = full.Model
	m.state = stateChat
	m.cursor = 0
	m.err = nil

	return m, nil
}

// handleSessionDelete removes the selected session.
func (m Model) handleSessionDelete() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.sessions) {
		return m, nil
	}

	selected := m.sessions[m.cursor]
	_ = m.sessionStore.Delete(selected.ID)

	// Check if deleting current session
	if m.chatSession != nil && m.chatSession.ID == selected.ID {
		m.chatSession = nil
		m.chatMessages = nil
		m.chatInput = ""
		m.chatLoading = false
		m.chatScroll = 0
	}

	m.sessions = append(m.sessions[:m.cursor], m.sessions[m.cursor+1:]...)
	if m.cursor >= len(m.sessions) && m.cursor > 0 {
		m.cursor--
	}
	if len(m.sessions) == 0 {
		m.cursor = 0
	}

	return m, nil
}

// handleCursorUp moves the cursor up by one position.
func (m Model) handleCursorUp() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.cursor--
	}
	return m, nil
}

// handleCursorDown moves the cursor down by one position.
func (m Model) handleCursorDown() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.sessions)-1 {
		m.cursor++
	}
	return m, nil
}

// handleSessionsLoaded populates the session list from async load.
func (m Model) handleSessionsLoaded(msg sessionsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.sessions = nil
	} else {
		m.sessions = msg.sessions
	}
	return m, nil
}

// toggleModel cycles between gemma-4-31b-it and gemma-4-26b-a4b-it.
func (m Model) toggleModel() (tea.Model, tea.Cmd) {
	if m.currentModel == "gemma-4-31b-it" {
		m.currentModel = "gemma-4-26b-a4b-it"
	} else {
		m.currentModel = "gemma-4-31b-it"
	}
	if m.chatSession != nil {
		m.chatSession.Model = m.currentModel
	}
	return m, nil
}
