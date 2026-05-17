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
