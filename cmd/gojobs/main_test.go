package main

import (
	"os"
	"testing"
)

// =============================================================================
// isTUIMode Tests
// =============================================================================

func TestIsTUIModeNoArgs(t *testing.T) {
	if !isTUIMode(nil) {
		t.Error("isTUIMode(nil) should be true with no args")
	}
	if !isTUIMode([]string{}) {
		t.Error("isTUIMode([]) should be true with empty args")
	}
}

func TestIsTUIModeWithURLFlag(t *testing.T) {
	if isTUIMode([]string{"-url", "https://example.com/job/123"}) {
		t.Error("isTUIMode with -url flag should be false")
	}
	if isTUIMode([]string{"--url", "https://example.com/job/123"}) {
		t.Error("isTUIMode with --url flag should be false")
	}
	// -url with no value (just flag presence) should still be CLI mode
	if isTUIMode([]string{"-url"}) {
		t.Error("isTUIMode with just -url should be false")
	}
}

func TestIsTUIModeWithCLIFlag(t *testing.T) {
	if isTUIMode([]string{"-cli"}) {
		t.Error("isTUIMode with -cli flag should be false")
	}
	if isTUIMode([]string{"--cli"}) {
		t.Error("isTUIMode with --cli flag should be false")
	}
	// -cli with other flags still forces CLI mode
	if isTUIMode([]string{"-cli", "-json"}) {
		t.Error("isTUIMode with -cli and other flags should be false")
	}
}

func TestIsTUIModeWithEnvVar(t *testing.T) {
	// Save and restore env
	oldVal, hadKey := os.LookupEnv("GOJOBS_MODE")
	defer func() {
		if hadKey {
			os.Setenv("GOJOBS_MODE", oldVal)
		} else {
			os.Unsetenv("GOJOBS_MODE")
		}
	}()

	os.Setenv("GOJOBS_MODE", "cli")

	if isTUIMode([]string{}) {
		t.Error("isTUIMode should be false when GOJOBS_MODE=cli, even with no args")
	}
}

func TestIsTUIModeEnvVarOverridesURLFlag(t *testing.T) {
	oldVal, hadKey := os.LookupEnv("GOJOBS_MODE")
	defer func() {
		if hadKey {
			os.Setenv("GOJOBS_MODE", oldVal)
		} else {
			os.Unsetenv("GOJOBS_MODE")
		}
	}()

	// -url with GOJOBS_MODE=cli: CLI wins (because of -url flag anyway)
	os.Setenv("GOJOBS_MODE", "cli")

	if isTUIMode([]string{"-url", "https://example.com"}) {
		t.Error("isTUIMode should be false with -url and GOJOBS_MODE=cli")
	}
}

func TestIsTUIModeWithUnknownFlags(t *testing.T) {
	// Flags not matching -url, --url, -cli, --cli should still be TUI mode
	if !isTUIMode([]string{"-json"}) {
		t.Error("isTUIMode with -json only should be true (fallback to TUI)")
	}
	if !isTUIMode([]string{"-model", "gemma-4-31b-it"}) {
		t.Error("isTUIMode with -model only should be true (fallback to TUI)")
	}
}

// =============================================================================
// runMain backward compatibility test (integration-style)
// =============================================================================

func TestRunMainURLFlagRequired(t *testing.T) {
	// Test that runMain returns an error when -url is not provided.
	// This tests the existing CLI behavior is preserved.
	err := runMain(t.Context(), os.Stdout, os.Stderr, []string{})
	if err == nil {
		t.Error("runMain with no args should return error (-url is required)")
	}
}
