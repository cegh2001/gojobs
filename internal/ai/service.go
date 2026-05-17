package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gojobs/internal/jobpage"
	"gojobs/internal/profile"

	"google.golang.org/genai"
)

type Request struct {
	Model     string
	Profile   profile.Profile
	Page      jobpage.Page
	ExtraNote string
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

	response, err := s.client.Models.GenerateContent(ctx, req.Model, genai.Text(prompt), &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: ResponseSchema(),
		Temperature:        &temperature,
		CandidateCount:     1,
		MaxOutputTokens:    1400,
	})
	if err != nil {
		return IntroRecommendation{}, fmt.Errorf("generate content with model %q: %w", req.Model, err)
	}

	payload := strings.TrimSpace(response.Text())
	if payload == "" {
		return IntroRecommendation{}, fmt.Errorf("model returned an empty response")
	}

	var recommendation IntroRecommendation
	if err := json.Unmarshal([]byte(payload), &recommendation); err != nil {
		return IntroRecommendation{}, fmt.Errorf("decode response JSON: %w; raw response: %s", err, payload)
	}

	return recommendation, nil
}
