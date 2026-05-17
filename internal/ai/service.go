package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"gojobs/internal/jobpage"
	"gojobs/internal/profile"

	"google.golang.org/genai"
)

type Request struct {
	Model          string
	FallbackModel  string
	Profile        profile.Profile
	Page           jobpage.Page
	ExtraNote      string
	ProgressWriter io.Writer
	CompactPrompt  bool
}

type Service struct {
	client *genai.Client
}

func NewService(ctx context.Context, apiKey string) (*Service, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create Google Gen AI client: %w", err)
	}

	return &Service{client: client}, nil
}

func (s *Service) Analyze(ctx context.Context, req Request) (IntroRecommendation, error) {
	prompt := BuildPrompt(req.Profile, req.Page, req.ExtraNote)
	if req.CompactPrompt {
		prompt = BuildCompactPrompt(req.Profile, req.Page, req.ExtraNote)
	}

	firstAttemptCtx := ctx
	firstAttemptCancel := func() {}
	if req.FallbackModel != "" {
		firstAttemptCtx, firstAttemptCancel = deriveFastAttemptContext(ctx, 35*time.Second)
	}
	defer firstAttemptCancel()

	recommendation, err := s.generateRecommendation(firstAttemptCtx, req.Model, prompt, req.ProgressWriter)
	if err == nil {
		return recommendation, nil
	}

	var structuredErr *structuredOutputError
	if req.FallbackModel != "" && req.FallbackModel != req.Model && ctx.Err() == nil && (errors.As(err, &structuredErr) || errors.Is(err, context.DeadlineExceeded) || isFallbackableModelError(err)) {
		if req.ProgressWriter != nil {
			if errors.Is(err, context.DeadlineExceeded) || isFallbackableModelError(err) {
				_, _ = fmt.Fprintf(req.ProgressWriter, "%s hit the fast-mode time budget. Retrying with %s...\n", req.Model, req.FallbackModel)
			} else {
				_, _ = fmt.Fprintf(req.ProgressWriter, "%s returned invalid structured output. Retrying with %s...\n", req.Model, req.FallbackModel)
			}
		}

		return s.generateRecommendation(ctx, req.FallbackModel, prompt, req.ProgressWriter)
	}

	return IntroRecommendation{}, err
}

func (s *Service) generateRecommendation(ctx context.Context, model string, prompt string, progressWriter io.Writer) (IntroRecommendation, error) {
	temperature := float32(0.2)
	stopProgress := startProgressHeartbeat(ctx, progressWriter, model, len(prompt))
	defer stopProgress()

	var response *genai.GenerateContentResponse
	var err error
	for attempt := 1; attempt <= 2; attempt++ {
		response, err = s.client.Models.GenerateContent(ctx, model, genai.Text(prompt), &genai.GenerateContentConfig{
			ResponseMIMEType:   "application/json",
			ResponseJsonSchema: ResponseSchema(),
			Temperature:        &temperature,
			CandidateCount:     1,
			MaxOutputTokens:    900,
		})
		if err == nil {
			break
		}

		if ctx.Err() != nil {
			return IntroRecommendation{}, fmt.Errorf("model %q did not finish before timeout: %w", model, ctx.Err())
		}

		if attempt == 1 && isRetryableModelError(err) {
			if progressWriter != nil {
				_, _ = fmt.Fprintf(progressWriter, "%s returned a transient upstream error. Retrying once...\n", model)
			}
			continue
		}

		return IntroRecommendation{}, fmt.Errorf("generate content with model %q: %w", model, err)
	}

	if progressWriter != nil {
		_, _ = fmt.Fprintln(progressWriter, "Model response received. Parsing JSON...")
	}

	recommendation, err := decodeRecommendationPayload(strings.TrimSpace(response.Text()))
	if err != nil {
		return IntroRecommendation{}, &structuredOutputError{err: err}
	}

	return recommendation, nil
}

type structuredOutputError struct {
	err error
}

func (e *structuredOutputError) Error() string {
	return e.err.Error()
}

func (e *structuredOutputError) Unwrap() error {
	return e.err
}

func decodeRecommendationPayload(payloads ...string) (IntroRecommendation, error) {
	var lastErr error

	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" {
			continue
		}

		var recommendation IntroRecommendation
		if err := json.Unmarshal([]byte(payload), &recommendation); err == nil {
			return recommendation, nil
		} else {
			lastErr = fmt.Errorf("decode response JSON: %w; raw response: %s", err, payload)
		}
	}

	if lastErr != nil {
		return IntroRecommendation{}, lastErr
	}

	return IntroRecommendation{}, fmt.Errorf("model returned an empty response")
}

func startProgressHeartbeat(ctx context.Context, writer io.Writer, model string, promptChars int) func() {
	if writer == nil {
		return func() {}
	}

	_, _ = fmt.Fprintf(writer, "Sending %d chars of grounded context to %s...\n", promptChars, model)

	done := make(chan struct{})
	var once sync.Once

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				_, _ = fmt.Fprintf(writer, "Still waiting for %s...\n", model)
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(done)
		})
	}
}

func deriveFastAttemptContext(parent context.Context, maxBudget time.Duration) (context.Context, context.CancelFunc) {
	deadline, ok := parent.Deadline()
	if !ok {
		return context.WithTimeout(parent, maxBudget)
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return parent, func() {}
	}

	budget := remaining / 3
	if budget > maxBudget {
		budget = maxBudget
	}
	if budget < 10*time.Second {
		return parent, func() {}
	}

	return context.WithTimeout(parent, budget)
}

func isRetryableModelError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "Error 500") ||
		strings.Contains(message, "Status: INTERNAL") ||
		strings.Contains(message, "Error 503") ||
		strings.Contains(message, "Status: UNAVAILABLE")
}

func isFallbackableModelError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "Error 504") ||
		strings.Contains(message, "DEADLINE_EXCEEDED")
}
