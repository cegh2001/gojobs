package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotEnvLocal(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_MODEL", "")
	t.Setenv("GOJOBS_FAST_MODEL", "")
	t.Setenv("GOJOBS_HEAVY_MODEL", "")

	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env.local")
	content := "GEMINI_API_KEY=test-key\nGOJOBS_MODEL=test-model\n"
	if err := os.WriteFile(envFile, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	appConfig := Load()
	if appConfig.APIKey != "test-key" {
		t.Fatalf("APIKey = %q, want %q", appConfig.APIKey, "test-key")
	}

	if appConfig.GoogleAPIKey != "test-key" {
		t.Fatalf("GoogleAPIKey = %q, want %q", appConfig.GoogleAPIKey, "test-key")
	}

	if appConfig.Model != "test-model" {
		t.Fatalf("Model = %q, want %q", appConfig.Model, "test-model")
	}

	if appConfig.DefaultModel != "test-model" {
		t.Fatalf("DefaultModel = %q, want %q", appConfig.DefaultModel, "test-model")
	}
}

func TestLoadPrefersExistingEnvironment(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "env-key")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_MODEL", "")
	t.Setenv("GOJOBS_FAST_MODEL", "")
	t.Setenv("GOJOBS_HEAVY_MODEL", "")

	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env.local")
	content := "GOOGLE_API_KEY=file-key\nGOJOBS_MODEL=file-model\n"
	if err := os.WriteFile(envFile, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	appConfig := Load()
	if appConfig.APIKey != "env-key" {
		t.Fatalf("APIKey = %q, want %q", appConfig.APIKey, "env-key")
	}

	if appConfig.GoogleAPIKey != "env-key" {
		t.Fatalf("GoogleAPIKey = %q, want %q", appConfig.GoogleAPIKey, "env-key")
	}

	if appConfig.Model != "file-model" {
		t.Fatalf("Model = %q, want %q", appConfig.Model, "file-model")
	}

	if appConfig.DefaultModel != "file-model" {
		t.Fatalf("DefaultModel = %q, want %q", appConfig.DefaultModel, "file-model")
	}
}

func TestResolveModelUsesDefaultUnlessOverridden(t *testing.T) {
	appConfig := Config{
		Model:        "gemma-4-31b-it",
		DefaultModel: "gemma-4-31b-it",
	}

	if got := appConfig.ResolveModel(""); got != "gemma-4-31b-it" {
		t.Fatalf("ResolveModel() = %q, want %q", got, "gemma-4-31b-it")
	}

	if got := appConfig.ResolveModel("custom-model"); got != "custom-model" {
		t.Fatalf("ResolveModel(override) = %q, want %q", got, "custom-model")
	}
}

func TestLoadPopulatesGoogleAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "google-key-123")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_MODEL", "")

	appConfig := Load()

	if appConfig.GoogleAPIKey != "google-key-123" {
		t.Fatalf("GoogleAPIKey = %q, want %q", appConfig.GoogleAPIKey, "google-key-123")
	}
}

func TestLoadFallsBackToGeminiAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "gemini-key-456")
	t.Setenv("GOJOBS_MODEL", "")

	appConfig := Load()

	if appConfig.GoogleAPIKey != "gemini-key-456" {
		t.Fatalf("GoogleAPIKey = %q, want %q", appConfig.GoogleAPIKey, "gemini-key-456")
	}
}

func TestLoadSetsDefaultModelToDefault(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_MODEL", "")

	appConfig := Load()

	if appConfig.DefaultModel != "gemma-4-31b-it" {
		t.Fatalf("DefaultModel = %q, want %q", appConfig.DefaultModel, "gemma-4-31b-it")
	}
}

func TestLoadSetsDefaultModelFromEnv(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_MODEL", "gemma-4-26b-a4b-it")

	appConfig := Load()

	if appConfig.DefaultModel != "gemma-4-26b-a4b-it" {
		t.Fatalf("DefaultModel = %q, want %q", appConfig.DefaultModel, "gemma-4-26b-a4b-it")
	}
}

func TestLoadSetsAvailableModelsDefaults(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_MODEL", "")

	appConfig := Load()

	if len(appConfig.AvailableModels) != 2 {
		t.Fatalf("len(AvailableModels) = %d, want 2", len(appConfig.AvailableModels))
	}

	if appConfig.AvailableModels[0] != "gemma-4-31b-it" {
		t.Fatalf("AvailableModels[0] = %q, want %q", appConfig.AvailableModels[0], "gemma-4-31b-it")
	}

	if appConfig.AvailableModels[1] != "gemma-4-26b-a4b-it" {
		t.Fatalf("AvailableModels[1] = %q, want %q", appConfig.AvailableModels[1], "gemma-4-26b-a4b-it")
	}
}

func TestValidateReturnsErrorWithoutAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	appConfig := Load()

	if appConfig.Validate() == nil {
		t.Fatal("Validate() expected error when no API key set, got nil")
	}
}

func TestValidateReturnsNilWithAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "has-key")
	t.Setenv("GEMINI_API_KEY", "")

	appConfig := Load()

	if appConfig.Validate() != nil {
		t.Fatalf("Validate() unexpected error: %v", appConfig.Validate())
	}
}

func TestBackwardCompatAPIKeyMirrorsGoogleAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "compat-key")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_MODEL", "")

	appConfig := Load()

	if appConfig.APIKey != appConfig.GoogleAPIKey {
		t.Fatalf("APIKey = %q, GoogleAPIKey = %q — should be equal", appConfig.APIKey, appConfig.GoogleAPIKey)
	}
}

func TestBackwardCompatModelMirrorsDefaultModel(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_MODEL", "custom-backward-model")

	appConfig := Load()

	if appConfig.Model != appConfig.DefaultModel {
		t.Fatalf("Model = %q, DefaultModel = %q — should be equal", appConfig.Model, appConfig.DefaultModel)
	}
}

func TestGoogleAPIKeyPrefersGoogleOverGemini(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "google-first")
	t.Setenv("GEMINI_API_KEY", "gemini-second")
	t.Setenv("GOJOBS_MODEL", "")

	appConfig := Load()

	if appConfig.GoogleAPIKey != "google-first" {
		t.Fatalf("GoogleAPIKey = %q, want %q (GOOGLE_API_KEY should take priority)", appConfig.GoogleAPIKey, "google-first")
	}
}
