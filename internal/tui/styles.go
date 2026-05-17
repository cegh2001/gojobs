package tui

import "github.com/charmbracelet/lipgloss"

// Color scheme: dark background with cyan/blue accents (ChatGPT dark mode inspired).
var (
	// AppStyle is the full application background style.
	AppStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1e2e")).
			Foreground(lipgloss.Color("#cdd6f4"))

	// HeaderStyle is the bold, colored header bar.
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#313244")).
			Foreground(lipgloss.Color("#89b4fa")).
			PaddingLeft(1).
			PaddingRight(1)

	// ChatStyle is the chat message area style.
	ChatStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1e2e")).
			Foreground(lipgloss.Color("#cdd6f4"))

	// UserMessageStyle is the user message prefix styling.
	UserMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6e3a1")).
				Bold(true)

	// AIMessageStyle is the AI message prefix styling.
	AIMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#89b4fa")).
			Bold(true)

	// InputStyle is the input area border + styling.
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475a")).
			PaddingLeft(1).
			Background(lipgloss.Color("#313244"))

	// SidebarStyle is the sidebar border + styling.
	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475a")).
			PaddingLeft(1).
			Background(lipgloss.Color("#313244"))

	// SpinnerStyle is the spinner color.
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f9e2af"))

	// HelpStyle is the dimmed footer help text.
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#585b70"))
)
