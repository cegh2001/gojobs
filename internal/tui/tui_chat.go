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

// handleURLSend initializes the async fetch of a job page and candidate profile loading.
// It returns immediately to let the Bubbletea TUI draw the spinner without blocking the main loop.
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

	// Save session history
	_ = m.sessionStore.Save(m.chatSession)

	return m, m.fetchAndLoadCmd(url, extraText)
}

// fetchAndLoadCmd returns a tea.Cmd that performs the heavy fetching and profile loading concurrently.
func (m Model) fetchAndLoadCmd(url string, extraText string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		fetcher := jobpage.NewFetcher(45 * time.Second)
		page, err := fetcher.Fetch(ctx, url)
		if err != nil {
			return urlFetchResultMsg{err: err}
		}

		candidateProfile, err := profile.Load(m.profilePath)
		if err != nil {
			return urlFetchResultMsg{err: err}
		}

		return urlFetchResultMsg{
			page:             page,
			candidateProfile: candidateProfile,
			extraText:        extraText,
		}
	}
}

// handleURLFetchResult processes the result of the async URL fetching and profile loading.
func (m Model) handleURLFetchResult(msg urlFetchResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		errContent := fmt.Sprintf("Error: %v", msg.err)
		m.chatSession.AddMessage("assistant", errContent)
		m.chatMessages = m.chatSession.Messages
		m.chatLoading = false
		m.err = msg.err
		_ = m.sessionStore.Save(m.chatSession)
		return m, nil
	}

	// Build context
	contextPrompt := buildChatPrompt(msg.candidateProfile, msg.page, msg.extraText)

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

	return m, m.sendChatStreamCmd(prov, providerMsgs)
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

	return m, m.sendChatStreamCmd(prov, providerMsgs)
}

// sendChatStreamCmd initiates a streaming message send and returns a command
// that starts the stream. Non-blocking — the actual tokens arrive as separate messages.
func (m Model) sendChatStreamCmd(prov provider.Provider, messages []provider.Message) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		ch, err := prov.SendMessageStream(ctx, m.currentModel, messages)
		if err != nil {
			return chatResponseMsg{err: fmt.Errorf("send message: %w", err)}
		}
		return chatStreamStartedMsg{ch: ch}
	}
}

// readStreamTokenCmd reads the next token from the streaming channel.
// Returns chatStreamEndMsg when the channel closes.
func readStreamTokenCmd(ch <-chan provider.StreamToken) tea.Cmd {
	return func() tea.Msg {
		token, ok := <-ch
		if !ok {
			return chatStreamEndMsg{}
		}
		if token.Err != nil {
			return chatStreamErrorMsg{err: fmt.Errorf("stream error: %w", token.Err)}
		}
		return chatStreamTokenMsg{token: token.Token}
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
// and job page. Produces the same high-quality analysis as the CLI but conversationally.
func buildChatPrompt(candidate profile.Profile, page jobpage.Page, extraNote string) string {
	var builder strings.Builder

	builder.WriteString("You are writing outreach introductions for Carlos Eduardo Gonzalez Henriquez.\n")
	builder.WriteString("Your job is to read the job page and the candidate dossier, then produce a recommendation that is specific, factual, and commercially aware.\n\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- Use only facts that appear in the dossier, the job page, or the runtime note.\n")
	builder.WriteString("- Do not invent leadership titles, funding, relocation plans, visa status, or employer names.\n")
	builder.WriteString("- If the page contains founder names, you may use them in the greeting.\n")
	builder.WriteString("- Prefer proof points over generic enthusiasm.\n")
	builder.WriteString("- Write in English unless the job page is predominantly in Spanish.\n")
	builder.WriteString("- Include a brief assessment of why this is a good fit (1-2 sentences).\n")
	builder.WriteString("- Provide at least 2 alternative introduction messages with different angles:\n")
	builder.WriteString("  - domain-business: emphasizes industry knowledge and business impact\n")
	builder.WriteString("  - ai-agents: emphasizes LLM/RAG/MCP/agent architecture\n")
	builder.WriteString("  - product-fullstack: emphasizes React/Next.js/full-stack delivery\n")
	builder.WriteString("- Each message should be around 110 to 160 words.\n")
	builder.WriteString("- Mention concrete projects (G-Aereo, RAG systems, MCP servers) when relevant.\n")
	builder.WriteString("- Include a fit score (0-100) and 1-2 sentence summary.\n")
	builder.WriteString("- List 2-3 specific evidence items that support the recommendation.\n")
	builder.WriteString("- Include 1-2 cautions about things to avoid mentioning.\n")
	builder.WriteString("- End with a helpful suggestion for the candidate (e.g., prepare CV, portfolio link).\n\n")

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
	builder.WriteString(trimRunes(page.Content, 800))
	builder.WriteString("\n\n")

	builder.WriteString("Write a comprehensive analysis. Include fit assessment, at least 2 message options with different angles, evidence, cautions, and a helpful suggestion.")

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
