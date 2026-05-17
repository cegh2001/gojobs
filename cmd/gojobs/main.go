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

	"gojobs/internal/ai"
	"gojobs/internal/config"
	"gojobs/internal/jobpage"
	"gojobs/internal/profile"
)

func main() {
	os.Exit(run())
}

func run() int {
	if err := runMain(context.Background(), os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	return 0
}

func runMain(parent context.Context, stdout io.Writer, stderr io.Writer, args []string) error {
	appConfig := config.Load()

	fs := flag.NewFlagSet("gojobs", flag.ContinueOnError)
	fs.SetOutput(stderr)

	jobURL := fs.String("url", "", "Job page URL to analyze")
	profilePath := fs.String("profile", appConfig.DefaultProfile, "Path to the candidate profile JSON")
	mode := fs.String("mode", "heavy", "Model mode: heavy or fast")
	modelOverride := fs.String("model", "", "Explicit model override")
	extraNote := fs.String("note", "", "Extra candidate context to inject into the prompt")
	promptOnly := fs.Bool("prompt-only", false, "Build and print the prompt without calling Google AI")
	jsonOutput := fs.Bool("json", false, "Print raw JSON instead of the human-readable report")
	timeout := fs.Duration("timeout", 4*time.Minute, "Timeout for fetching and model execution")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*jobURL) == "" {
		return errors.New("-url is required")
	}

	resolvedMode, err := config.NormalizeMode(*mode)
	if err != nil {
		return err
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
		_, err := io.WriteString(stdout, ai.BuildPrompt(candidateProfile, page, note)+"\n")
		return err
	}

	if err := appConfig.Validate(); err != nil {
		return err
	}

	service, err := ai.NewService(ctx, appConfig.APIKey)
	if err != nil {
		return err
	}

	modelName := appConfig.ResolveModel(resolvedMode, *modelOverride)
	logStep(stderr, "Calling model %s...", modelName)
	response, err := service.Analyze(ctx, ai.Request{
		Model:          modelName,
		Profile:        candidateProfile,
		Page:           page,
		ExtraNote:      note,
		ProgressWriter: stderr,
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
