package ai

import (
	"strings"
	"testing"

	"gojobs/internal/jobpage"
	"gojobs/internal/profile"
)

func TestBuildPromptIncludesGuardrailsAndEvidence(t *testing.T) {
	candidateProfile := profile.Profile{
		Name:               "Carlos Eduardo Gonzalez Henriquez",
		Headline:           "AI Engineer and Full Stack Developer",
		Location:           "La Guaira, Venezuela",
		VerifiedHighlights: []string{"Built G-Aereo logistics flows", "Built RAG systems"},
		Guardrails:         []string{"Do not invent relocation plans"},
	}
	page := jobpage.Page{
		URL:     "https://example.com/jobs/1",
		Title:   "Founding AI Engineer",
		Content: "We need someone who understands logistics and LLM workflows.",
	}

	prompt := BuildPrompt(candidateProfile, page, "Open to discussing on-site roles if needed.")

	checks := []string{
		"Use only facts that appear in the dossier, the job page, or the runtime note.",
		"Built G-Aereo logistics flows",
		"Open to discussing on-site roles if needed.",
		"Founding AI Engineer",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Fatalf("prompt should contain %q, got %q", check, prompt)
		}
	}
}

func TestBuildCompactPromptIsShorterAndKeepsKeyFacts(t *testing.T) {
	candidateProfile := profile.Profile{
		Name:               "Carlos Eduardo Gonzalez Henriquez",
		Headline:           "AI Engineer and Full Stack Developer",
		Location:           "La Guaira, Venezuela",
		Languages:          []string{"Spanish", "English B2"},
		VerifiedHighlights: []string{"Built G-Aereo logistics flows", "Built RAG systems", "Built human handover UI"},
		PrivateProjects: []profile.Project{{
			Name:       "mcp-sisprot",
			Highlights: []string{"Private MCP server project with SmartOLT integrations"},
		}},
		PreferredAngles: []profile.Angle{{
			Name:        "ai-agents",
			WhenToUse:   "Use when the role is about LLMs and MCP.",
			ProofPoints: []string{"Built RAG systems", "Built MCP infrastructure"},
		}},
		Guardrails: []string{"Do not invent relocation plans"},
	}
	page := jobpage.Page{
		URL:     "https://example.com/jobs/1",
		Title:   "Founding AI Engineer",
		Content: strings.Repeat("LLM workflow ", 500),
	}

	fullPrompt := BuildPrompt(candidateProfile, page, "")
	compactPrompt := BuildCompactPrompt(candidateProfile, page, "")

	if len(compactPrompt) >= len(fullPrompt) {
		t.Fatalf("compact prompt should be shorter than full prompt: compact=%d full=%d", len(compactPrompt), len(fullPrompt))
	}

	checks := []string{
		"Built G-Aereo logistics flows",
		"mcp-sisprot",
		"Founding AI Engineer",
		"Do not invent relocation plans",
	}

	for _, check := range checks {
		if !strings.Contains(compactPrompt, check) {
			t.Fatalf("compact prompt should contain %q, got %q", check, compactPrompt)
		}
	}

	if strings.Contains(compactPrompt, strings.Repeat("LLM workflow ", 50)) {
		t.Fatalf("compact prompt should trim oversized page content, got %q", compactPrompt)
	}
}
