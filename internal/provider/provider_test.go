package provider

import (
	"context"
	"errors"
	"testing"
)

// mockProvider implements the Provider interface for testing.
type mockProvider struct {
	name            string
	supportedModels []string
	streamTokens    []StreamToken
	completeText    string
	completeErr     error
	streamErr       error
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) SupportedModels() []string { return m.supportedModels }

func (m *mockProvider) SendMessageStream(ctx context.Context, model string, messages []Message) (<-chan StreamToken, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}

	ch := make(chan StreamToken, len(m.streamTokens))
	go func() {
		defer close(ch)
		for _, t := range m.streamTokens {
			select {
			case ch <- t:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func (m *mockProvider) SendMessage(ctx context.Context, model string, messages []Message) (string, error) {
	if m.completeErr != nil {
		return "", m.completeErr
	}
	return m.completeText, nil
}

// Ensure mockProvider implements Provider at compile time.
var _ Provider = (*mockProvider)(nil)

func TestRouterResolvesGemmaModelsToGoogleProvider(t *testing.T) {
	googleProv := &mockProvider{
		name:            "google",
		supportedModels: []string{"gemma-4-31b-it", "gemma-4-26b-a4b-it"},
	}

	router := NewRouter()
	router.Register(googleProv)

	tests := []struct {
		model   string
		wantErr bool
	}{
		{model: "gemma-4-31b-it", wantErr: false},
		{model: "gemma-4-26b-a4b-it", wantErr: false},
		{model: "unknown-model", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			prov, err := router.Resolve(tt.model)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Resolve() expected error, got nil")
				}
				if prov != nil {
					t.Fatalf("Resolve() expected nil provider on error, got %v", prov)
				}
				return
			}

			if err != nil {
				t.Fatalf("Resolve() unexpected error: %v", err)
			}

			if prov.Name() != "google" {
				t.Fatalf("Name() = %q, want %q", prov.Name(), "google")
			}

			if len(prov.SupportedModels()) != 2 {
				t.Fatalf("len(SupportedModels()) = %d, want 2", len(prov.SupportedModels()))
			}
		})
	}
}

func TestRouterAllModelsAggregatesFromAllProviders(t *testing.T) {
	googleProv := &mockProvider{
		name:            "google",
		supportedModels: []string{"gemma-4-31b-it", "gemma-4-26b-a4b-it"},
	}

	futureProv := &mockProvider{
		name:            "deepseek",
		supportedModels: []string{"deepseek-chat"},
	}

	router := NewRouter()
	router.Register(googleProv)
	router.Register(futureProv)

	models := router.AllModels()

	if len(models) != 3 {
		t.Fatalf("len(AllModels()) = %d, want 3", len(models))
	}

	modelSet := make(map[string]bool)
	for _, m := range models {
		modelSet[m] = true
	}

	expected := []string{"gemma-4-31b-it", "gemma-4-26b-a4b-it", "deepseek-chat"}
	for _, exp := range expected {
		if !modelSet[exp] {
			t.Fatalf("AllModels() missing %q", exp)
		}
	}
}

func TestRouterResolveReturnsErrorWithAvailableModels(t *testing.T) {
	googleProv := &mockProvider{
		name:            "google",
		supportedModels: []string{"gemma-4-31b-it", "gemma-4-26b-a4b-it"},
	}

	router := NewRouter()
	router.Register(googleProv)

	_, err := router.Resolve("nonexistent-model")
	if err == nil {
		t.Fatal("Resolve() expected error for unknown model, got nil")
	}

	errStr := err.Error()
	if errStr == "" {
		t.Fatal("Resolve() returned empty error message")
	}

	// Error should mention available models
	for _, expectedModel := range googleProv.SupportedModels() {
		if !contains(errStr, expectedModel) {
			t.Fatalf("error message %q does not mention available model %q", errStr, expectedModel)
		}
	}
}

func TestRoleConstantsHaveExpectedValues(t *testing.T) {
	tests := []struct {
		role Role
		want string
	}{
		{role: RoleSystem, want: "system"},
		{role: RoleUser, want: "user"},
		{role: RoleAssistant, want: "assistant"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if string(tt.role) != tt.want {
				t.Fatalf("Role = %q, want %q", tt.role, tt.want)
			}
		})
	}
}

func TestMessageStructHasExpectedFields(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Hello, world!",
	}

	if msg.Role != RoleUser {
		t.Fatalf("Role = %q, want %q", msg.Role, RoleUser)
	}

	if msg.Content != "Hello, world!" {
		t.Fatalf("Content = %q, want %q", msg.Content, "Hello, world!")
	}
}

func TestStreamTokenStructHasExpectedFields(t *testing.T) {
	t.Run("token with text", func(t *testing.T) {
		token := StreamToken{Token: "hello", Err: nil}
		if token.Token != "hello" {
			t.Fatalf("Token = %q, want %q", token.Token, "hello")
		}
		if token.Err != nil {
			t.Fatalf("Err = %v, want nil", token.Err)
		}
	})

	t.Run("token with error", func(t *testing.T) {
		wantErr := errors.New("test error")
		token := StreamToken{Token: "", Err: wantErr}
		if token.Err != wantErr {
			t.Fatalf("Err = %v, want %v", token.Err, wantErr)
		}
		if token.Token != "" {
			t.Fatalf("Token = %q, want empty", token.Token)
		}
	})
}

func TestMockProviderSendMessageStreamYieldsTokens(t *testing.T) {
	mock := &mockProvider{
		name:            "test",
		supportedModels: []string{"test-model"},
		streamTokens: []StreamToken{
			{Token: "Hello "},
			{Token: "world"},
		},
	}

	ch, err := mock.SendMessageStream(context.Background(), "test-model", nil)
	if err != nil {
		t.Fatalf("SendMessageStream() error = %v", err)
	}

	var tokens []string
	for token := range ch {
		if token.Err != nil {
			t.Fatalf("unexpected stream error: %v", token.Err)
		}
		tokens = append(tokens, token.Token)
	}

	if len(tokens) != 2 {
		t.Fatalf("len(tokens) = %d, want 2", len(tokens))
	}

	combined := tokens[0] + tokens[1]
	if combined != "Hello world" {
		t.Fatalf("combined tokens = %q, want %q", combined, "Hello world")
	}
}

func TestMockProviderSendMessageReturnsText(t *testing.T) {
	mock := &mockProvider{
		name:         "test",
		completeText: "complete response",
	}

	text, err := mock.SendMessage(context.Background(), "test-model", nil)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if text != "complete response" {
		t.Fatalf("SendMessage() = %q, want %q", text, "complete response")
	}
}

func TestMockProviderSendMessageReturnsError(t *testing.T) {
	wantErr := errors.New("api error")
	mock := &mockProvider{
		name:        "test",
		completeErr: wantErr,
	}

	_, err := mock.SendMessage(context.Background(), "test-model", nil)
	if err == nil {
		t.Fatal("SendMessage() expected error, got nil")
	}
}

func TestMockProviderSendMessageStreamReturnsError(t *testing.T) {
	wantErr := errors.New("stream init error")
	mock := &mockProvider{
		name:      "test",
		streamErr: wantErr,
	}

	ch, err := mock.SendMessageStream(context.Background(), "test-model", nil)
	if err != wantErr {
		t.Fatalf("SendMessageStream() error = %v, want %v", err, wantErr)
	}
	if ch != nil {
		t.Fatal("SendMessageStream() expected nil channel on error")
	}
}

func TestRouterRegisterOverwritesByName(t *testing.T) {
	first := &mockProvider{
		name:            "google",
		supportedModels: []string{"old-model"},
	}

	second := &mockProvider{
		name:            "google",
		supportedModels: []string{"new-model"},
	}

	router := NewRouter()
	router.Register(first)
	router.Register(second)

	models := router.AllModels()
	if len(models) != 1 {
		t.Fatalf("len(AllModels()) = %d, want 1 after overwrite", len(models))
	}

	if models[0] != "new-model" {
		t.Fatalf("AllModels()[0] = %q, want %q", models[0], "new-model")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
