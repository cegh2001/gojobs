package tui

import (
	"gojobs/internal/jobpage"
	"gojobs/internal/profile"
	"gojobs/internal/provider"
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

// chatStreamStartedMsg signals that a streaming response has started
// and carries the channel for reading tokens.
type chatStreamStartedMsg struct {
	ch <-chan provider.StreamToken
}

// chatStreamTokenMsg carries a single token from the streaming response.
type chatStreamTokenMsg struct {
	token string
}

// chatStreamEndMsg signals that the stream has completed successfully.
type chatStreamEndMsg struct{}

// chatStreamErrorMsg signals a stream-level error.
type chatStreamErrorMsg struct {
	err error
}
