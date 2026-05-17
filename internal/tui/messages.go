package tui

import "gojobs/internal/session"

// StreamTokenMsg carries a single token from a streaming AI response.
type StreamTokenMsg struct {
	Token string
}

// StreamDoneMsg signals that streaming has completed with the full content.
type StreamDoneMsg struct {
	Content string
}

// StreamErrMsg signals that an error occurred during streaming.
type StreamErrMsg struct {
	Err error
}

// SessionsLoadedMsg carries the sessions loaded from storage on startup.
type SessionsLoadedMsg struct {
	Sessions []session.Session
}

// SessionSelectedMsg signals that the user has selected a session from the sidebar.
type SessionSelectedMsg struct {
	Session *session.Session
}

// ModelSelectedMsg signals that the user has changed the current model.
type ModelSelectedMsg struct {
	Model string
}
