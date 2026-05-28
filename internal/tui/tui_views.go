package tui

import (
	"fmt"
	"strings"

	"gojobs/internal/session"

	"github.com/mattn/go-runewidth"
)

// viewChat renders the main chat view: header, messages area, separator, input line, status line, help bar.
func (m Model) viewChat() string {
	messagesHeight := m.chatMessageAreaHeight()

	var sb strings.Builder

	// Header
	sb.WriteString(titleStyle.Render(fmt.Sprintf("gojobs — %s", m.currentModel)) + "\n")

	// Messages area
	messagesLines := m.renderChatMessages(messagesHeight)
	sb.WriteString(messagesLines)

	// Fill remaining space
	usedLines := strings.Count(messagesLines, "\n")
	for i := usedLines; i < messagesHeight; i++ {
		sb.WriteString("\n")
	}

	// Separator
	sb.WriteString(helpStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Input line
	sb.WriteString(m.renderChatInputLine())
	// Status line
	sb.WriteString("\n" + m.renderChatStatusLine())

	// Help bar
	sb.WriteString("\n" + helpStyle.Render("Ctrl+N: nuevo | Esc: sesiones | Ctrl+K: modelo | Ctrl+C: salir"))

	return sb.String()
}

// renderChatMessages renders the scrollable messages pane.
func (m Model) renderChatMessages(maxLines int) string {
	var sb strings.Builder
	lines := m.buildChatTranscriptLines()

	if len(lines) == 0 {
		return ""
	}

	if maxLines < 1 {
		maxLines = 1
	}

	scroll := m.clampChatScroll()
	start := len(lines) - maxLines - scroll
	if start < 0 {
		start = 0
	}
	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
	}

	for _, line := range lines[start:end] {
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// buildChatTranscriptLines converts chat messages into wrapped, styled lines.
func (m Model) buildChatTranscriptLines() []string {
	if len(m.chatMessages) == 0 {
		return []string{
			senseiStyle.Bold(true).Render("🤖 AI"),
			"¡Hola! Soy tu asistente de búsqueda laboral. Pegá una URL de oferta o preguntame lo que necesites.",
			"",
		}
	}

	lines := make([]string, 0, len(m.chatMessages)*3)
	for _, msg := range m.chatMessages {
		lines = append(lines, renderWrappedMessage(msg.Role, msg.Content, m.width)...)
	}

	return lines
}

// renderWrappedMessage wraps and styles a single message with appropriate prefix.
func renderWrappedMessage(role session.Role, content string, width int) []string {
	prefix := "🤖 AI"
	style := senseiStyle
	if role == session.RoleUser {
		prefix = "Tú"
		style = userStyle
	}

	header := style.Bold(true).Render(prefix)
	wrappedLines := wrapTextByWidth(content, width)
	if len(wrappedLines) == 0 {
		wrappedLines = []string{""}
	}

	lines := make([]string, 0, len(wrappedLines)+2)
	lines = append(lines, header)
	for _, line := range wrappedLines {
		lines = append(lines, line)
	}
	lines = append(lines, "") // blank line for spacing between messages

	return lines
}

// renderChatInputLine renders the input line or placeholder when empty.
func (m Model) renderChatInputLine() string {
	prefixText := "Tú: "
	prefix := userStyle.Render(prefixText)
	if m.chatLoading {
		return prefix + m.spinner.View()
	}

	if m.chatInput == "" {
		return prefix + infoStyle.Render("pegá una URL o escribí tu mensaje")
	}

	visibleWidth := m.width - runewidth.StringWidth(prefixText)
	if visibleWidth < 1 {
		visibleWidth = 1
	}

	return prefix + inputStyle.Render(clipInputTail(m.chatInput, visibleWidth))
}

// renderChatStatusLine renders the status line — loading indicator, scroll info, or key hints.
func (m Model) renderChatStatusLine() string {
	if m.chatLoading {
		return spinnerStyle.Render(m.spinner.View() + " pensando...")
	}

	if m.statusNotification != "" {
		return infoStyle.Render(m.statusNotification)
	}

	scroll := m.clampChatScroll()
	maxScroll := m.maxChatScroll()
	if maxScroll > 0 && scroll > 0 {
		return infoStyle.Render(fmt.Sprintf("Viendo mensajes anteriores (%d/%d) · End: volver al final", scroll, maxScroll))
	}

	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	return infoStyle.Render("Enter: enviar | ↑/↓: scroll | PgUp/PgDn: salto")
}

// viewSessions renders the session list overlay.
func (m Model) viewSessions() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("📋 Sesiones") + "\n")
	sb.WriteString(helpStyle.Render("j/k: navegar  enter: seleccionar  esc: volver") + "\n\n")

	if len(m.sessions) == 0 {
		sb.WriteString(infoStyle.Render("No hay sesiones guardadas.") + "\n")
		return sb.String()
	}

	for i, sess := range m.sessions {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("→ ")
		}

		timeStr := sess.UpdatedAt.Format("02/01 15:04")
		msgCount := len(sess.Messages)
		sb.WriteString(fmt.Sprintf("%s%s  %s  (%d mensajes)\n",
			cursor,
			truncateForList(sess.Name, 40),
			infoStyle.Render(timeStr),
			msgCount,
		))
	}

	sb.WriteString("\n" + helpStyle.Render("enter: cargar  del: eliminar  esc: volver"))
	return sb.String()
}

// --- Scroll helpers ---

// shiftChatScroll adjusts the scroll position by delta (positive = scroll up/back).
func (m Model) shiftChatScroll(delta int) Model {
	if delta == 0 {
		return m
	}

	next := m.chatScroll + delta
	if next < 0 {
		next = 0
	}

	maxScroll := m.maxChatScroll()
	if next > maxScroll {
		next = maxScroll
	}

	m.chatScroll = next
	return m
}

// maxChatScroll returns the maximum scroll offset (total lines - visible area).
func (m Model) maxChatScroll() int {
	totalLines := len(m.buildChatTranscriptLines())
	maxLines := m.chatMessageAreaHeight()
	if totalLines <= maxLines {
		return 0
	}
	return totalLines - maxLines
}

// clampChatScroll returns the scroll position clamped to valid range.
func (m Model) clampChatScroll() int {
	if m.chatScroll < 0 {
		return 0
	}
	maxScroll := m.maxChatScroll()
	if m.chatScroll > maxScroll {
		return maxScroll
	}
	return m.chatScroll
}

// chatPageScrollStep returns the number of lines to scroll for PgUp/PgDn.
func (m Model) chatPageScrollStep() int {
	step := m.chatMessageAreaHeight() - 1
	if step < 1 {
		step = 1
	}
	return step
}

// chatMessageAreaHeight returns the available height for messages.
func (m Model) chatMessageAreaHeight() int {
	height := m.height - chatHeaderLines() - chatFooterLines()
	if height < 1 {
		height = 1
	}
	return height
}

// chatHeaderLines returns the number of lines used by the header.
func chatHeaderLines() int { return 1 }

// chatFooterLines returns the number of lines used by footer (separator + input + status + help).
func chatFooterLines() int { return 4 }

// --- Utility functions ---

// wrapTextByWidth wraps text to fit within the given display width.
func wrapTextByWidth(text string, width int) []string {
	if width < 1 {
		return []string{text}
	}

	var lines []string
	for _, rawLine := range strings.Split(text, "\n") {
		if rawLine == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(rawLine)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		current := words[0]
		currentWidth := runewidth.StringWidth(current)
		for _, word := range words[1:] {
			for _, piece := range splitWordByWidth(word, width) {
				pieceWidth := runewidth.StringWidth(piece)
				if currentWidth == 0 {
					current = piece
					currentWidth = pieceWidth
					continue
				}

				if currentWidth+1+pieceWidth <= width {
					current += " " + piece
					currentWidth += 1 + pieceWidth
					continue
				}

				lines = append(lines, current)
				current = piece
				currentWidth = pieceWidth
			}
		}

		if current != "" {
			lines = append(lines, current)
		}
	}

	return lines
}

// splitWordByWidth splits a word that is wider than the display width.
func splitWordByWidth(word string, width int) []string {
	if runewidth.StringWidth(word) <= width {
		return []string{word}
	}

	var pieces []string
	var current strings.Builder
	currentWidth := 0
	for _, r := range word {
		runeWidth := runewidth.RuneWidth(r)
		if currentWidth+runeWidth > width && current.Len() > 0 {
			pieces = append(pieces, current.String())
			current.Reset()
			currentWidth = 0
		}

		current.WriteRune(r)
		currentWidth += runeWidth
	}

	if current.Len() > 0 {
		pieces = append(pieces, current.String())
	}

	return pieces
}

// clipInputTail clips the input text from the left to fit within the given width.
func clipInputTail(text string, width int) string {
	if width < 1 {
		return ""
	}

	if runewidth.StringWidth(text) <= width {
		return text
	}

	ellipsis := "…"
	ellipsisWidth := runewidth.StringWidth(ellipsis)
	if width <= ellipsisWidth {
		return tailTextByWidth(text, width)
	}

	return ellipsis + tailTextByWidth(text, width-ellipsisWidth)
}

// tailTextByWidth returns the last N display-width characters of text.
func tailTextByWidth(text string, width int) string {
	if width < 1 {
		return ""
	}

	runes := []rune(text)
	currentWidth := 0
	start := len(runes)
	for start > 0 {
		runeWidth := runewidth.RuneWidth(runes[start-1])
		if currentWidth+runeWidth > width {
			break
		}
		currentWidth += runeWidth
		start--
	}

	return string(runes[start:])
}

// truncateForList truncates a string for display in the session list.
func truncateForList(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
