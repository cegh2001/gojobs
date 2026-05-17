package ai

import (
	"fmt"
	"strings"

	"gojobs/internal/jobpage"
	"gojobs/internal/profile"
)

func BuildPrompt(candidate profile.Profile, page jobpage.Page, extraNote string) string {
	var builder strings.Builder

	builder.WriteString("You are writing outreach introductions for Carlos Eduardo Gonzalez Henriquez.\n")
	builder.WriteString("Your job is to read the job page and the candidate dossier, then produce a recommendation that is specific, factual, and commercially aware.\n\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- Use only facts that appear in the dossier, the job page, or the runtime note.\n")
	builder.WriteString("- Do not invent leadership titles, funding, relocation plans, visa status, or employer names.\n")
	builder.WriteString("- If the page contains founder names, you may use them in the greeting.\n")
	builder.WriteString("- Prefer proof points over generic enthusiasm.\n")
	builder.WriteString("- Write in English unless the job page is predominantly in Spanish.\n")
	builder.WriteString("- Keep the primary message around 140 to 190 words.\n")
	builder.WriteString("- Keep the alternative message around 110 to 170 words.\n")
	builder.WriteString("- Recommended angle must be one of: domain-business, ai-agents, product-fullstack.\n")
	builder.WriteString("- If a claim is uncertain, omit it instead of hedging.\n\n")

	if trimmedNote := strings.TrimSpace(extraNote); trimmedNote != "" {
		builder.WriteString("Runtime note:\n")
		builder.WriteString(trimmedNote)
		builder.WriteString("\n\n")
	}

	builder.WriteString("Candidate dossier:\n")
	builder.WriteString(candidate.Dossier())
	builder.WriteString("\n\n")

	builder.WriteString("Job page:\n")
	builder.WriteString(fmt.Sprintf("URL: %s\n", page.URL))
	builder.WriteString(fmt.Sprintf("Title: %s\n", page.Title))
	if page.MetaDescription != "" {
		builder.WriteString(fmt.Sprintf("Meta description: %s\n", page.MetaDescription))
	}
	builder.WriteString("Page text:\n")
	builder.WriteString(page.Content)
	builder.WriteString("\n\n")

	builder.WriteString("Return JSON only. Evaluate the best angle, explain the fit briefly, and draft two intro messages rooted in the candidate's strongest matching evidence.\n")

	return builder.String()
}

func ResponseSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"company_name",
			"role_title",
			"recommended_angle",
			"fit_score",
			"fit_summary",
			"primary_message",
			"secondary_message",
			"evidence_used",
			"cautions",
		},
		"properties": map[string]any{
			"company_name": map[string]any{
				"type": "string",
			},
			"role_title": map[string]any{
				"type": "string",
			},
			"recommended_angle": map[string]any{
				"type": "string",
				"enum": []string{"domain-business", "ai-agents", "product-fullstack"},
			},
			"fit_score": map[string]any{
				"type":    "integer",
				"minimum": 0,
				"maximum": 100,
			},
			"fit_summary": map[string]any{
				"type": "string",
			},
			"primary_message": map[string]any{
				"type": "string",
			},
			"secondary_message": map[string]any{
				"type": "string",
			},
			"evidence_used": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
			"cautions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
	}
}
