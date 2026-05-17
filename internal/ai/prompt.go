package ai

import (
	"fmt"
	"strings"
	"unicode/utf8"

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

func BuildCompactPrompt(candidate profile.Profile, page jobpage.Page, extraNote string) string {
	var builder strings.Builder

	builder.WriteString("You are writing outreach introductions for Carlos Eduardo Gonzalez Henriquez.\n")
	builder.WriteString("Use the compact dossier below and the job page snippet to produce a precise, factual intro.\n\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- Return JSON only.\n")
	builder.WriteString("- Do not repeat words or phrases.\n")
	builder.WriteString("- Use only facts present in the compact dossier, job page, or runtime note.\n")
	builder.WriteString("- If the page is sparse, rely on the most relevant proof points instead of guessing.\n")
	builder.WriteString("- Keep fit_summary to 1 sentence when possible, never more than 2.\n")
	builder.WriteString("- Keep evidence_used to at most 3 short items.\n")
	builder.WriteString("- Keep cautions to at most 2 short items.\n")
	builder.WriteString("- Keep primary_message around 110 to 160 words.\n")
	builder.WriteString("- Keep secondary_message around 85 to 130 words.\n")
	builder.WriteString("- Recommended angle must be one of: domain-business, ai-agents, product-fullstack.\n\n")

	if trimmedNote := strings.TrimSpace(extraNote); trimmedNote != "" {
		builder.WriteString("Runtime note:\n")
		builder.WriteString(trimmedNote)
		builder.WriteString("\n\n")
	}

	builder.WriteString("Compact candidate dossier:\n")
	builder.WriteString(buildCompactDossier(candidate))
	builder.WriteString("\n\n")

	builder.WriteString("Job page:\n")
	builder.WriteString(fmt.Sprintf("URL: %s\n", page.URL))
	builder.WriteString(fmt.Sprintf("Title: %s\n", page.Title))
	if page.MetaDescription != "" {
		builder.WriteString(fmt.Sprintf("Meta description: %s\n", page.MetaDescription))
	}
	builder.WriteString("Page text snippet:\n")
	builder.WriteString(trimRunes(page.Content, 400))
	builder.WriteString("\n\n")
	builder.WriteString("Return valid JSON matching the schema. If needed, reduce verbosity instead of omitting closing JSON brackets.\n")

	return builder.String()
}

func buildCompactDossier(candidate profile.Profile) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Name: %s\n", candidate.Name))
	builder.WriteString(fmt.Sprintf("Headline: %s\n", candidate.Headline))
	builder.WriteString(fmt.Sprintf("Location: %s\n", candidate.Location))

	if len(candidate.Languages) > 0 {
		builder.WriteString(fmt.Sprintf("Languages: %s\n", strings.Join(candidate.Languages, ", ")))
	}

	if len(candidate.Education) > 0 {
		item := candidate.Education[0]
		builder.WriteString(fmt.Sprintf("Education: %s in %s, %s, graduation %s\n", item.Degree, item.Institution, item.Location, item.Graduation))
	}

	writeLimitedBulletSection(&builder, "Strongest proof points", candidate.VerifiedHighlights, 5)

	if len(candidate.PrivateProjects) > 0 {
		projectBullets := flattenProjectHighlights(candidate.PrivateProjects, 2)
		writeLimitedBulletSection(&builder, "Private repo evidence", projectBullets, len(projectBullets))
	}

	if len(candidate.PreferredAngles) > 0 {
		builder.WriteString("\nAngle cues:\n")
		for _, angle := range candidate.PreferredAngles {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", angle.Name, angle.WhenToUse))
			for _, proofPoint := range limitStrings(angle.ProofPoints, 1) {
				builder.WriteString(fmt.Sprintf("  - %s\n", proofPoint))
			}
		}
	}

	writeLimitedBulletSection(&builder, "Guardrails", candidate.Guardrails, 4)
	return strings.TrimSpace(builder.String())
}

func writeLimitedBulletSection(builder *strings.Builder, title string, items []string, limit int) {
	items = limitStrings(items, limit)
	if len(items) == 0 {
		return
	}

	builder.WriteString("\n")
	builder.WriteString(title)
	builder.WriteString(":\n")
	for _, item := range items {
		builder.WriteString(fmt.Sprintf("- %s\n", item))
	}
}

func flattenProjectHighlights(projects []profile.Project, projectLimit int) []string {
	var bullets []string
	for _, project := range limitProjects(projects, projectLimit) {
		if len(project.Highlights) == 0 {
			continue
		}

		bullets = append(bullets, fmt.Sprintf("%s: %s", project.Name, project.Highlights[0]))
	}

	return bullets
}

func limitStrings(items []string, limit int) []string {
	if limit <= 0 || len(items) <= limit {
		return items
	}

	return items[:limit]
}

func limitProjects(items []profile.Project, limit int) []profile.Project {
	if limit <= 0 || len(items) <= limit {
		return items
	}

	return items[:limit]
}

func trimRunes(raw string, limit int) string {
	if utf8.RuneCountInString(raw) <= limit {
		return raw
	}

	runes := []rune(raw)
	if limit <= 3 {
		return string(runes[:limit])
	}

	return string(runes[:limit-3]) + "..."
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
				"type":     "array",
				"maxItems": 3,
				"items": map[string]any{
					"type": "string",
				},
			},
			"cautions": map[string]any{
				"type":     "array",
				"maxItems": 2,
				"items": map[string]any{
					"type": "string",
				},
			},
		},
	}
}
