package tui

import "gojobs/internal/session"

// chatResponseMsg carries the final accumulated response from the provider.
type chatResponseMsg struct {
	content string
	err     error
}

// sessionsLoadedMsg carries the session list loaded from storage.
type sessionsLoadedMsg struct {
	sessions []session.Session
	err      error
}
