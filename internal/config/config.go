package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultProfilePath = "profiles/carlos_gonzalez.json"
	DefaultFastModel   = "gemma-4-31b-it"
	DefaultHeavyModel  = "gemma-4-31b-it"
)

type Config struct {
	APIKey         string
	FastModel      string
	HeavyModel     string
	DefaultProfile string
	HTTPTimeout    time.Duration
}

func Load() Config {
	loadEnvFiles(".env.local", ".env")

	apiKey := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	} else {
		_ = os.Unsetenv("GEMINI_API_KEY")
	}

	if apiKey != "" && strings.TrimSpace(os.Getenv("GOOGLE_API_KEY")) == "" {
		_ = os.Unsetenv("GOOGLE_API_KEY")
	}

	fastModel := strings.TrimSpace(os.Getenv("GOJOBS_FAST_MODEL"))
	if fastModel == "" {
		fastModel = DefaultFastModel
	}

	heavyModel := strings.TrimSpace(os.Getenv("GOJOBS_HEAVY_MODEL"))
	if heavyModel == "" {
		heavyModel = DefaultHeavyModel
	}

	return Config{
		APIKey:         apiKey,
		FastModel:      fastModel,
		HeavyModel:     heavyModel,
		DefaultProfile: DefaultProfilePath,
		HTTPTimeout:    45 * time.Second,
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("missing GOOGLE_API_KEY or GEMINI_API_KEY")
	}

	return nil
}

func (c Config) ResolveModel(mode string, override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}

	return c.HeavyModel
}

func NormalizeMode(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", "heavy":
		return "heavy", nil
	case "fast":
		return "fast", nil
	default:
		return "", fmt.Errorf("invalid -mode %q: use heavy or fast", mode)
	}
}

func loadEnvFiles(paths ...string) {
	for _, path := range paths {
		_ = loadEnvFile(path)
	}
}

func loadEnvFile(path string) error {
	cleanPath := filepath.Clean(path)
	file, err := os.Open(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(os.Getenv(key)) != "" {
			continue
		}

		value := strings.TrimSpace(rawValue)
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}

	return scanner.Err()
}
