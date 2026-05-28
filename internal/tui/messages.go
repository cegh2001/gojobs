package tui

import (
	"gojobs/internal/jobpage"
	"gojobs/internal/profile"
	"gojobs/internal/session"
)

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

// urlFetchResultMsg carries the result of async URL fetching and profile loading.
type urlFetchResultMsg struct {
	page             jobpage.Page
	candidateProfile profile.Profile
	extraText        string
	err              error
}
