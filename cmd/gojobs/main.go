package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gojobs/internal/ai"
	"gojobs/internal/config"
	"gojobs/internal/jobpage"
	"gojobs/internal/profile"
	"gojobs/internal/provider"
	"gojobs/internal/session"
	"gojobs/internal/tui"
)

func main() {
	os.Exit(run())
}

func run() int {
	args := os.Args[1:]

	// Check if TUI mode or CLI mode
	if isTUIMode(args) {
		return runTUI()
	}
	return runCLI()
}

// isTUIMode determines whether to launch the TUI based on flags and env.
// Returns true when no CLI-forcing flags (-url, -cli) are present and
// GOJOBS_MODE is not set to "cli".
func isTUIMode(args []string) bool {
	if os.Getenv("GOJOBS_MODE") == "cli" {
		return false
	}
	for _, arg := range args {
		if arg == "-url" || arg == "--url" || arg == "-cli" || arg == "--cli" {
			return false
		}
	}
	return true
}

// runCLI runs the existing one-shot CLI flow (identical to previous behavior).
func runCLI() int {
	if err := runMain(context.Background(), os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// runTUI launches the BubbleTea TUI chat interface.
//
// Smoke test checklist for manual verification:
//  1. go run ./cmd/gojobs → TUI should open
//  2. Paste a job URL in the input, press Enter → AI should respond with intro message
//  3. Type a follow-up question, press Enter → AI should respond with context awareness
//  4. Switch model to gemma-4-26b-a4b-it → send another message → should use the new model
//  5. Create a new session, switch between sessions
//  6. Delete a session
//  7. Ctrl+C to quit
//  8. go run ./cmd/gojobs -url <link> → should still work as one-shot CLI
//
// The TUI launches even without an API key; API key is validated lazily on first send.
func runTUI() int {
	cfg := config.Load()
	_ = cfg.Validate() // Don't fail — TUI can start without API key

	// Create provider router
	ctx := context.Background()
	googleProvider, err := provider.NewGoogleProvider(ctx, cfg.GoogleAPIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to initialize Google AI provider: %v\n", err)
		// Continue anyway — user will see error in TUI
	}

	router := provider.NewRouter()
	if googleProvider != nil {
		router.Register(googleProvider)
	}

	// Create session store
	sessionStore := session.NewStore("sessions", 10)

	// Create and run TUI model
	m := tui.NewModel(sessionStore, router, cfg.DefaultProfile)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func runMain(parent context.Context, stdout io.Writer, stderr io.Writer, args []string) error {
	appConfig := config.Load()
	normalizedArgs, ignoredMode := stripDeprecatedModeFlag(args)

	fs := flag.NewFlagSet("gojobs", flag.ContinueOnError)
	fs.SetOutput(stderr)

	jobURL := fs.String("url", "", "Job page URL to analyze")
	profilePath := fs.String("profile", appConfig.DefaultProfile, "Path to the candidate profile JSON")
	modelOverride := fs.String("model", "", "Explicit model override")
	extraNote := fs.String("note", "", "Extra candidate context to inject into the prompt")
	promptOnly := fs.Bool("prompt-only", false, "Build and print the prompt without calling Google AI")
	jsonOutput := fs.Bool("json", false, "Print raw JSON instead of the human-readable report")
	timeout := fs.Duration("timeout", 4*time.Minute, "Timeout for fetching and model execution")
	_ = fs.Bool("cli", false, "Force CLI mode (used by run() dispatch, ignored here)")

	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}

	if ignoredMode {
		logStep(stderr, "Ignoring deprecated -mode flag. gojobs now always uses the optimized single-model path unless -model is provided.")
	}

	if strings.TrimSpace(*jobURL) == "" {
		return errors.New("-url is required")
	}

	ctx, cancel := context.WithTimeout(parent, *timeout)
	defer cancel()

	logStep(stderr, "Loading candidate profile...")
	candidateProfile, err := profile.Load(*profilePath)
	if err != nil {
		return err
	}

	logStep(stderr, "Fetching job page...")
	fetcher := jobpage.NewFetcher(*timeout)
	page, err := fetcher.Fetch(ctx, *jobURL)
	if err != nil {
		return err
	}

	note := strings.TrimSpace(*extraNote)
	if *promptOnly {
		prompt := ai.BuildCompactPrompt(candidateProfile, page, note)

		_, err := io.WriteString(stdout, prompt+"\n")
		return err
	}

	if err := appConfig.Validate(); err != nil {
		return err
	}

	service, err := ai.NewService(ctx, appConfig.GoogleAPIKey)
	if err != nil {
		return err
	}

	modelName := appConfig.ResolveModel(*modelOverride)
	logStep(stderr, "Calling model %s with optimized prompt...", modelName)
	response, err := service.Analyze(ctx, ai.Request{
		Model:          modelName,
		Profile:        candidateProfile,
		Page:           page,
		ExtraNote:      note,
		ProgressWriter: stderr,
		CompactPrompt:  true,
	})
	if err != nil {
		return err
	}

	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(response)
	}

	renderResponse(stdout, response)
	return nil
}

func logStep(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}

	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

func renderResponse(w io.Writer, response ai.IntroRecommendation) {
	fmt.Fprintf(w, "Company: %s\n", response.CompanyName)
	fmt.Fprintf(w, "Role: %s\n", response.RoleTitle)
	fmt.Fprintf(w, "Recommended angle: %s\n", response.RecommendedAngle)
	fmt.Fprintf(w, "Fit score: %d/100\n\n", response.FitScore)
	fmt.Fprintf(w, "Fit summary:\n%s\n\n", response.FitSummary)
	fmt.Fprintf(w, "Recommended message:\n%s\n\n", response.PrimaryMessage)
	fmt.Fprintf(w, "Alternative message:\n%s\n\n", response.SecondaryMessage)

	if len(response.EvidenceUsed) > 0 {
		fmt.Fprintln(w, "Evidence used:")
		for _, item := range response.EvidenceUsed {
			fmt.Fprintf(w, "- %s\n", item)
		}
		fmt.Fprintln(w)
	}

	if len(response.Cautions) > 0 {
		fmt.Fprintln(w, "Cautions:")
		for _, item := range response.Cautions {
			fmt.Fprintf(w, "- %s\n", item)
		}
	}
}

func stripDeprecatedModeFlag(args []string) ([]string, bool) {
	normalized := make([]string, 0, len(args))
	ignored := false

	for index := 0; index < len(args); index++ {
		arg := args[index]

		switch {
		case arg == "-mode" || arg == "--mode":
			ignored = true
			if index+1 < len(args) && !strings.HasPrefix(args[index+1], "-") {
				index++
			}
		case strings.HasPrefix(arg, "-mode=") || strings.HasPrefix(arg, "--mode="):
			ignored = true
		default:
			normalized = append(normalized, arg)
		}
	}

	return normalized, ignored
}
