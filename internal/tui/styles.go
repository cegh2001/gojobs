package tui

import "github.com/charmbracelet/lipgloss"

// Styles inspired by the GoDojo TUI pattern — clean, terminal-native colors.
var (
	senseiStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))   // blue for AI
	userStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))  // yellow/gold for user
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))  // gray for info
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	inputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)
