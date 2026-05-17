package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/session"
)

// Ensure all custom message types satisfy tea.Msg by assigning to the interface.
func TestCustomMessagesImplementTeaMsg(t *testing.T) {
	var _ tea.Msg = StreamTokenMsg{}
	var _ tea.Msg = StreamDoneMsg{}
	var _ tea.Msg = StreamErrMsg{}
	var _ tea.Msg = SessionsLoadedMsg{}
	var _ tea.Msg = SessionSelectedMsg{}
	var _ tea.Msg = ModelSelectedMsg{}
}

func TestStreamTokenMsgFields(t *testing.T) {
	msg := StreamTokenMsg{Token: "Hello"}
	if msg.Token != "Hello" {
		t.Errorf("StreamTokenMsg.Token = %q, want %q", msg.Token, "Hello")
	}
}

func TestStreamDoneMsgFields(t *testing.T) {
	msg := StreamDoneMsg{Content: "Full response text"}
	if msg.Content != "Full response text" {
		t.Errorf("StreamDoneMsg.Content = %q, want %q", msg.Content, "Full response text")
	}
}

func TestStreamErrMsgFields(t *testing.T) {
	underlying := errors.New("stream error")
	msg := StreamErrMsg{Err: underlying}
	if msg.Err == nil {
		t.Error("StreamErrMsg.Err should not be nil")
	}
	if msg.Err.Error() != "stream error" {
		t.Errorf("StreamErrMsg.Err = %q, want %q", msg.Err.Error(), "stream error")
	}
}

func TestSessionsLoadedMsgFields(t *testing.T) {
	sessions := []session.Session{
		{ID: "abc", Name: "Chat 1"},
		{ID: "def", Name: "Chat 2"},
	}
	msg := SessionsLoadedMsg{Sessions: sessions}
	if len(msg.Sessions) != 2 {
		t.Errorf("SessionsLoadedMsg should have 2 sessions, got %d", len(msg.Sessions))
	}
	if msg.Sessions[0].ID != "abc" {
		t.Errorf("first session ID = %q, want %q", msg.Sessions[0].ID, "abc")
	}
}

func TestSessionSelectedMsgFields(t *testing.T) {
	s := &session.Session{ID: "xyz", Name: "Test Session"}
	msg := SessionSelectedMsg{Session: s}
	if msg.Session == nil {
		t.Error("SessionSelectedMsg.Session should not be nil")
	}
	if msg.Session.ID != "xyz" {
		t.Errorf("SessionSelectedMsg.Session.ID = %q, want %q", msg.Session.ID, "xyz")
	}
}

func TestModelSelectedMsgFields(t *testing.T) {
	msg := ModelSelectedMsg{Model: "gemma-4-31b-it"}
	if msg.Model != "gemma-4-31b-it" {
		t.Errorf("ModelSelectedMsg.Model = %q, want %q", msg.Model, "gemma-4-31b-it")
	}
}
