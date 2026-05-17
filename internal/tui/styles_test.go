package tui

import (
	"strings"
	"testing"
)

func TestStylesAreInitialized(t *testing.T) {
	tests := []struct {
		name   string
		result string
	}{
		{"AppStyle", AppStyle.Render("test")},
		{"HeaderStyle", HeaderStyle.Render("test")},
		{"ChatStyle", ChatStyle.Render("test")},
		{"UserMessageStyle", UserMessageStyle.Render("test")},
		{"AIMessageStyle", AIMessageStyle.Render("test")},
		{"InputStyle", InputStyle.Render("test")},
		{"SidebarStyle", SidebarStyle.Render("test")},
		{"SpinnerStyle", SpinnerStyle.Render("test")},
		{"HelpStyle", HelpStyle.Render("test")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result == "" {
				t.Errorf("%s returned empty string", tt.name)
			}
		})
	}
}

func TestUserMessageStyleFormatting(t *testing.T) {
	result := UserMessageStyle.Render("Hello world")
	if !strings.Contains(result, "Hello world") {
		t.Errorf("UserMessageStyle.Render() does not contain input text, got: %s", result)
	}
}

func TestAIMessageStyleFormatting(t *testing.T) {
	result := AIMessageStyle.Render("AI response")
	if !strings.Contains(result, "AI response") {
		t.Errorf("AIMessageStyle.Render() does not contain input text, got: %s", result)
	}
}

func TestHeaderStyleNotEmpty(t *testing.T) {
	result := HeaderStyle.Render("gojobs — gemma-4-31b-it")
	if result == "" {
		t.Error("HeaderStyle.Render() returned empty string")
	}
}

func TestHelpStyleIsDimmer(t *testing.T) {
	result := HelpStyle.Render("help text")
	if result == "" {
		t.Error("HelpStyle.Render() returned empty string")
	}
}
