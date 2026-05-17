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

	if appConfig.Model != "test-model" {
		t.Fatalf("Model = %q, want %q", appConfig.Model, "test-model")
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

	if appConfig.Model != "file-model" {
		t.Fatalf("Model = %q, want %q", appConfig.Model, "file-model")
	}
}

func TestResolveModelUsesDefaultUnlessOverridden(t *testing.T) {
	appConfig := Config{
		Model: "gemma-4-31b-it",
	}

	if got := appConfig.ResolveModel(""); got != "gemma-4-31b-it" {
		t.Fatalf("ResolveModel() = %q, want %q", got, "gemma-4-31b-it")
	}

	if got := appConfig.ResolveModel("custom-model"); got != "custom-model" {
		t.Fatalf("ResolveModel(override) = %q, want %q", got, "custom-model")
	}
}
