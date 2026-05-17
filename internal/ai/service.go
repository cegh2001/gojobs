package ai

import (
	"context"
	"encoding/json"
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
	Profile        profile.Profile
	Page           jobpage.Page
	ExtraNote      string
	ProgressWriter io.Writer
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
	temperature := float32(0.35)
	stopProgress := startProgressHeartbeat(ctx, req.ProgressWriter, req.Model, len(prompt))
	defer stopProgress()

	stream := s.client.Models.GenerateContentStream(ctx, req.Model, genai.Text(prompt), &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: ResponseSchema(),
		Temperature:        &temperature,
		CandidateCount:     1,
		MaxOutputTokens:    1000,
	})

	var streamedText strings.Builder
	var lastChunk string
	var sawStreamText bool

	for response, err := range stream {
		if err != nil {
			return IntroRecommendation{}, fmt.Errorf("generate content with model %q: %w", req.Model, err)
		}

		chunkText := strings.TrimSpace(response.Text())
		if chunkText == "" {
			continue
		}

		if !sawStreamText && req.ProgressWriter != nil {
			_, _ = fmt.Fprintf(req.ProgressWriter, "%s started streaming a response...\n", req.Model)
			sawStreamText = true
		}

		lastChunk = chunkText
		streamedText.WriteString(chunkText)
	}

	if err := ctx.Err(); err != nil {
		return IntroRecommendation{}, fmt.Errorf("model %q did not finish before timeout: %w", req.Model, err)
	}

	if req.ProgressWriter != nil {
		_, _ = fmt.Fprintln(req.ProgressWriter, "Model response received. Parsing JSON...")
	}

	recommendation, err := decodeRecommendationPayload(lastChunk, streamedText.String())
	if err != nil {
		return IntroRecommendation{}, err
	}

	return recommendation, nil
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
