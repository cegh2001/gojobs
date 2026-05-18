package tui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/jobpage"
	"gojobs/internal/profile"
	"gojobs/internal/provider"
	"gojobs/internal/session"
)

// handleChatTextInput accumulates text input from the user.
func (m Model) handleChatTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.chatLoading {
		return m, nil
	}

	// Handle paste: normalize and append
	if msg.Paste {
		m.chatInput += normalizePastedInput(string(msg.Runes))
		return m, nil
	}

	// Handle backspace
	if msg.Type == tea.KeyBackspace {
		if len(m.chatInput) > 0 {
			runes := []rune(m.chatInput)
			m.chatInput = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	// Handle regular key runes
	for _, r := range msg.Runes {
		m.chatInput += string(r)
	}
	return m, nil
}

// handleChatSend processes the current input and sends it to the AI provider.
// If the input contains a URL (http/https), it fetches the job page, loads the profile,
// builds a grounded prompt, and uses that as context for the AI.
// Any extra text is passed as a runtime note.
func (m Model) handleChatSend() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.chatInput)
	if input == "" || m.chatLoading {
		return m, nil
	}

	// Create session if needed
	if m.chatSession == nil {
		m.chatSession = session.NewSession(m.currentModel)
		m.chatMessages = nil
	}

	// Check if input contains a URL
	url, extraText := extractURL(input)
	if url != "" {
		return m.handleURLSend(url, extraText)
	}

	return m.handleTextSend(input)
}

// extractURL finds the first URL in the input and returns it plus any surrounding text as a note.
func extractURL(input string) (url string, note string) {
	lower := strings.ToLower(input)
	for _, prefix := range []string{"https://", "http://"} {
		idx := strings.Index(lower, prefix)
		if idx < 0 {
			continue
		}
		// Extract URL: from prefix to next space or end of string
		urlStart := idx
		urlEnd := len(input)
		if spaceIdx := strings.Index(input[urlStart:], " "); spaceIdx >= 0 {
			urlEnd = urlStart + spaceIdx
		}

		url = input[urlStart:urlEnd]
		// Build extra note from surrounding text
		before := strings.TrimSpace(input[:urlStart])
		after := strings.TrimSpace(input[urlEnd:])
		if before != "" && after != "" {
			note = before + " " + after
		} else if before != "" {
			note = before
		} else if after != "" {
			note = after
		}
		return url, note
	}
	return "", ""
}

// handleURLSend fetches a job page from a URL, builds a grounded prompt with
// the candidate profile, and sends it to the AI for analysis.
// extraText is any additional user text surrounding the URL (used as a runtime note).
func (m Model) handleURLSend(url string, extraText string) (tea.Model, tea.Cmd) {
	// Show the full user input in the chat
	displayText := url
	if extraText != "" {
		displayText = extraText + " " + url
	}
	m.chatSession.AddMessage("user", displayText)
	m.chatMessages = m.chatSession.Messages
	m.chatInput = ""
	m.chatLoading = true
	m.chatScroll = 0
	m.err = nil

	// Fetch job page
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	fetcher := jobpage.NewFetcher(45 * time.Second)
	page, err := fetcher.Fetch(ctx, url)
	if err != nil {
		errContent := fmt.Sprintf("Error fetching job page: %v", err)
		m.chatSession.AddMessage("assistant", errContent)
		m.chatMessages = m.chatSession.Messages
		m.chatLoading = false
		m.err = err
		_ = m.sessionStore.Save(m.chatSession)
		return m, nil
	}

	// Load profile
	candidateProfile, profileErr := profile.Load(m.profilePath)
	if profileErr != nil {
		errContent := fmt.Sprintf("Error loading profile: %v", profileErr)
		m.chatSession.AddMessage("assistant", errContent)
		m.chatMessages = m.chatSession.Messages
		m.chatLoading = false
		m.err = profileErr
		_ = m.sessionStore.Save(m.chatSession)
		return m, nil
	}

	// Build context
	contextPrompt := buildChatPrompt(candidateProfile, page, extraText)

	// Build a rich message: prepend context to the AI-facing user message.
	// This avoids SystemInstruction API issues with Gemma 4 Chat.
	enrichedMsg := fmt.Sprintf(
		"%s\n\nBased on the context above, write a tailored introduction message for this job posting.",
		contextPrompt,
	)

	// Persist
	_ = m.sessionStore.Save(m.chatSession)

	// Resolve provider
	prov, err := m.providerRouter.Resolve(m.currentModel)
	if err != nil {
		errContent := fmt.Sprintf("Error: model %q — %v", m.currentModel, err)
		m.chatSession.AddMessage("assistant", errContent)
		m.chatMessages = m.chatSession.Messages
		m.chatLoading = false
		m.err = err
		_ = m.sessionStore.Save(m.chatSession)
		return m, nil
	}

	// Build provider messages: use enriched message instead of the raw URL.
	// The enriched message includes full context for the AI.
	providerMsgs := []provider.Message{
		{Role: provider.RoleUser, Content: enrichedMsg},
	}

	return m, m.sendChatCmd(prov, providerMsgs)
}

// handleTextSend sends a regular text message to the AI (non-URL, follow-up).
func (m Model) handleTextSend(input string) (tea.Model, tea.Cmd) {
	m.chatSession.AddMessage("user", input)
	m.chatMessages = m.chatSession.Messages
	m.chatInput = ""
	m.chatLoading = true
	m.chatScroll = 0
	m.err = nil

	// Persist
	_ = m.sessionStore.Save(m.chatSession)

	// Resolve provider
	prov, err := m.providerRouter.Resolve(m.currentModel)
	if err != nil {
		errContent := fmt.Sprintf("Error: model %q — %v", m.currentModel, err)
		m.chatSession.AddMessage("assistant", errContent)
		m.chatMessages = m.chatSession.Messages
		m.chatLoading = false
		m.err = err
		_ = m.sessionStore.Save(m.chatSession)
		return m, nil
	}

	// Build provider messages from session history
	var providerMsgs []provider.Message
	for _, msg := range m.chatMessages {
		providerMsgs = append(providerMsgs, provider.Message{
			Role:    provider.Role(msg.Role),
			Content: msg.Content,
		})
	}

	return m, m.sendChatCmd(prov, providerMsgs)
}

// sendChatCmd creates a goroutine that reads stream tokens, accumulates them,
// and sends a single chatResponseMsg when streaming completes.
func (m Model) sendChatCmd(prov provider.Provider, messages []provider.Message) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		ch, err := prov.SendMessageStream(ctx, m.currentModel, messages)
		if err != nil {
			return chatResponseMsg{err: fmt.Errorf("send message: %w", err)}
		}

		var content string
		for token := range ch {
			if token.Err != nil {
				return chatResponseMsg{err: fmt.Errorf("stream error: %w", token.Err)}
			}
			content += token.Token
		}

		return chatResponseMsg{content: content}
	}
}

// handleChatResponse appends the AI response to the chat history.
func (m Model) handleChatResponse(msg chatResponseMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		errContent := fmt.Sprintf("Error: %v", msg.err)
		m.chatSession.AddMessage("assistant", errContent)
		m.chatMessages = m.chatSession.Messages
		m.chatLoading = false
		m.err = msg.err
		m.chatScroll = 0
		_ = m.sessionStore.Save(m.chatSession)
		return m, nil
	}

	if msg.content == "" {
		msg.content = "(no response)"
	}

	m.chatSession.AddMessage("assistant", msg.content)
	m.chatMessages = m.chatSession.Messages
	m.chatLoading = false
	m.chatScroll = 0
	m.err = nil
	_ = m.sessionStore.Save(m.chatSession)

	return m, nil
}

// handleChatNew creates a new chat session.
func (m Model) handleChatNew() (tea.Model, tea.Cmd) {
	// Save current session first
	if m.chatSession != nil {
		_ = m.sessionStore.Save(m.chatSession)
	}

	m.chatSession = session.NewSession(m.currentModel)
	m.chatMessages = nil
	m.chatInput = ""
	m.chatLoading = false
	m.chatScroll = 0
	m.err = nil

	// Add to sessions list
	m.sessions = append([]session.Session{*m.chatSession}, m.sessions...)

	return m, nil
}

// normalizePastedInput replaces newline characters with spaces.
func normalizePastedInput(text string) string {
	return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(text)
}

// buildChatPrompt creates a chat-friendly context prompt from the candidate profile
// and job page. Unlike BuildCompactPrompt (which enforces JSON schema for one-shot
// CLI use), this version asks for a conversational introduction message with no
// structured output requirements.
func buildChatPrompt(candidate profile.Profile, page jobpage.Page, extraNote string) string {
	var builder strings.Builder

	builder.WriteString("You are writing outreach introductions for Carlos Eduardo Gonzalez Henriquez.\n")
	builder.WriteString("Use the compact dossier below and the job page to write a tailored, conversational introduction message.\n\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- Use only facts present in the dossier, job page, or runtime note.\n")
	builder.WriteString("- Write in a natural, conversational tone — this is a chat, not a formal document.\n")
	builder.WriteString("- Keep the message around 110 to 160 words.\n")
	builder.WriteString("- Write in English unless the job page is predominantly in Spanish.\n")
	builder.WriteString("- If the page is sparse, rely on the most relevant proof points instead of guessing.\n")
	builder.WriteString("- Do not invent details about relocation, visa, salary, or employer names.\n\n")

	if trimmedNote := strings.TrimSpace(extraNote); trimmedNote != "" {
		builder.WriteString("Runtime note:\n")
		builder.WriteString(trimmedNote)
		builder.WriteString("\n\n")
	}

	builder.WriteString("Candidate dossier:\n")
	builder.WriteString(candidate.Dossier())
	builder.WriteString("\n\n")

	builder.WriteString("Job page:\n")
	builder.WriteString(fmt.Sprintf("URL: %s\n", page.URL))
	builder.WriteString(fmt.Sprintf("Title: %s\n", page.Title))
	if page.MetaDescription != "" {
		builder.WriteString(fmt.Sprintf("Meta description: %s\n", page.MetaDescription))
	}
	builder.WriteString("Page content:\n")
	builder.WriteString(trimRunes(page.Content, 600))
	builder.WriteString("\n\n")

	builder.WriteString("Write a tailored introduction message for this specific job. Be specific, factual, and persuasive. Mention concrete skills and projects that match the role.")

	return builder.String()
}

// trimRunes truncates a string to at most `limit` runes, appending "..." if truncated.
func trimRunes(raw string, limit int) string {
	if utf8.RuneCountInString(raw) <= limit {
		return raw
	}
	runes := []rune(raw)
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}
