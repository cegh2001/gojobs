package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotEnvLocal(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_FAST_MODEL", "")
	t.Setenv("GOJOBS_HEAVY_MODEL", "")

	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env.local")
	content := "GEMINI_API_KEY=test-key\nGOJOBS_FAST_MODEL=test-fast\n"
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

	if appConfig.FastModel != "test-fast" {
		t.Fatalf("FastModel = %q, want %q", appConfig.FastModel, "test-fast")
	}

	if appConfig.HeavyModel != DefaultHeavyModel {
		t.Fatalf("HeavyModel = %q, want %q", appConfig.HeavyModel, DefaultHeavyModel)
	}
}

func TestLoadPrefersExistingEnvironment(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "env-key")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOJOBS_FAST_MODEL", "")
	t.Setenv("GOJOBS_HEAVY_MODEL", "")

	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env.local")
	content := "GOOGLE_API_KEY=file-key\nGOJOBS_FAST_MODEL=file-fast\n"
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

	if appConfig.FastModel != "file-fast" {
		t.Fatalf("FastModel = %q, want %q", appConfig.FastModel, "file-fast")
	}
}

func TestResolveModelAlwaysUsesHeavyUnlessOverridden(t *testing.T) {
	appConfig := Config{
		FastModel:  "gemma-4-26b-a4b-it",
		HeavyModel: "gemma-4-31b-it",
	}

	if got := appConfig.ResolveModel("fast", ""); got != "gemma-4-31b-it" {
		t.Fatalf("ResolveModel(fast) = %q, want %q", got, "gemma-4-31b-it")
	}

	if got := appConfig.ResolveModel("heavy", "custom-model"); got != "custom-model" {
		t.Fatalf("ResolveModel(override) = %q, want %q", got, "custom-model")
	}
}
