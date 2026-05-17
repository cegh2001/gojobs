package tui

import tea "github.com/charmbracelet/bubbletea"

// FocusArea represents which UI component currently has focus.
type FocusArea int

const (
	FocusInput   FocusArea = iota
	FocusSidebar
	FocusChat
)

// Custom tea.Msg types for focus management commands.
type (
	// FocusChangeMsg signals a focus change between UI areas.
	FocusChangeMsg struct {
		Area FocusArea
	}

	// SendMessageMsg signals that the user wants to send the input text.
	SendMessageMsg struct {
		Content string
	}

	// NewSessionMsg signals that the user wants to create a new session.
	NewSessionMsg struct{}

	// DeleteSessionMsg signals that the user wants to delete the current session.
	DeleteSessionMsg struct{}

	// ToggleModelMsg signals that the user wants to toggle the model.
	ToggleModelMsg struct{}

	// ScrollUpMsg signals scrolling up in the chat view.
	ScrollUpMsg struct{}

	// ScrollDownMsg signals scrolling down in the chat view.
	ScrollDownMsg struct{}
)

// HandleKey dispatches a key message based on the current focus area.
// Returns an appropriate tea.Cmd or nil if the key is not handled.
func HandleKey(msg tea.KeyMsg, focus FocusArea) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit

	case tea.KeyTab:
		return focusCmd(cycleForward(focus))

	case tea.KeyShiftTab:
		return focusCmd(cycleBackward(focus))

	case tea.KeyEnter:
		if focus == FocusInput {
			return func() tea.Msg { return SendMessageMsg{} }
		}
		return nil

	case tea.KeyEsc:
		if focus == FocusSidebar {
			return focusCmd(FocusInput)
		}
		return nil

	case tea.KeyCtrlN:
		return func() tea.Msg { return NewSessionMsg{} }

	case tea.KeyCtrlD:
		return func() tea.Msg { return DeleteSessionMsg{} }

	case tea.KeyCtrlK:
		return func() tea.Msg { return ToggleModelMsg{} }

	case tea.KeyUp, tea.KeyDown:
		if focus == FocusSidebar {
			// Return a command that the sidebar component will handle
			return nil
		}
		return nil

	case tea.KeyPgUp:
		if focus == FocusChat {
			return func() tea.Msg { return ScrollUpMsg{} }
		}
		return nil

	case tea.KeyPgDown:
		if focus == FocusChat {
			return func() tea.Msg { return ScrollDownMsg{} }
		}
		return nil
	}

	return nil
}

// focusCmd creates a command that sends a FocusChangeMsg.
func focusCmd(area FocusArea) tea.Cmd {
	return func() tea.Msg {
		return FocusChangeMsg{Area: area}
	}
}

// cycleForward returns the next focus area in the Tab cycle.
func cycleForward(focus FocusArea) FocusArea {
	switch focus {
	case FocusInput:
		return FocusSidebar
	case FocusSidebar:
		return FocusChat
	case FocusChat:
		return FocusInput
	default:
		return FocusInput
	}
}

// cycleBackward returns the previous focus area in the Shift+Tab cycle.
func cycleBackward(focus FocusArea) FocusArea {
	switch focus {
	case FocusInput:
		return FocusChat
	case FocusSidebar:
		return FocusInput
	case FocusChat:
		return FocusSidebar
	default:
		return FocusInput
	}
}
