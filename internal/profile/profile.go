package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Profile struct {
	Name               string       `json:"name"`
	Headline           string       `json:"headline"`
	Location           string       `json:"location"`
	Languages          []string     `json:"languages"`
	Education          []Education  `json:"education"`
	OtherSkills        []string     `json:"other_skills"`
	PublicLinks        []string     `json:"public_links"`
	Positioning        []string     `json:"positioning"`
	VerifiedHighlights []string     `json:"verified_highlights"`
	Experience         []Experience `json:"experience"`
	PublicProjects     []Project    `json:"public_projects"`
	PrivateProjects    []Project    `json:"private_projects"`
	PreferredAngles    []Angle      `json:"preferred_angles"`
	Guardrails         []string     `json:"guardrails"`
}

type Education struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Location    string `json:"location"`
	Graduation  string `json:"graduation"`
}

type Experience struct {
	Company    string   `json:"company"`
	Role       string   `json:"role"`
	Period     string   `json:"period"`
	Source     string   `json:"source"`
	Highlights []string `json:"highlights"`
}

type Project struct {
	Name       string   `json:"name"`
	Source     string   `json:"source"`
	Highlights []string `json:"highlights"`
}

type Angle struct {
	Name        string   `json:"name"`
	WhenToUse   string   `json:"when_to_use"`
	ProofPoints []string `json:"proof_points"`
}

func Load(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, fmt.Errorf("read profile %q: %w", path, err)
	}

	var candidateProfile Profile
	if err := json.Unmarshal(data, &candidateProfile); err != nil {
		return Profile{}, fmt.Errorf("decode profile %q: %w", path, err)
	}

	return candidateProfile, nil
}

func (p Profile) Dossier() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Name: %s\n", p.Name))
	builder.WriteString(fmt.Sprintf("Headline: %s\n", p.Headline))
	builder.WriteString(fmt.Sprintf("Location: %s\n", p.Location))

	if len(p.Languages) > 0 {
		builder.WriteString(fmt.Sprintf("Languages: %s\n", strings.Join(p.Languages, ", ")))
	}

	if len(p.Education) > 0 {
		builder.WriteString("\nEducation:\n")
		for _, item := range p.Education {
			builder.WriteString(fmt.Sprintf("- %s | %s | %s | graduation: %s\n", item.Institution, item.Degree, item.Location, item.Graduation))
		}
	}

	if len(p.PublicLinks) > 0 {
		builder.WriteString(fmt.Sprintf("Public links: %s\n", strings.Join(p.PublicLinks, ", ")))
	}

	writeBulletSection(&builder, "Other skills", p.OtherSkills)

	writeBulletSection(&builder, "Positioning", p.Positioning)
	writeBulletSection(&builder, "Verified highlights", p.VerifiedHighlights)

	if len(p.Experience) > 0 {
		builder.WriteString("\nExperience:\n")
		for _, item := range p.Experience {
			builder.WriteString(fmt.Sprintf("- %s | %s | %s | source: %s\n", item.Company, item.Role, item.Period, item.Source))
			for _, highlight := range item.Highlights {
				builder.WriteString(fmt.Sprintf("  - %s\n", highlight))
			}
		}
	}

	if len(p.PublicProjects) > 0 {
		builder.WriteString("\nPublic projects:\n")
		for _, item := range p.PublicProjects {
			builder.WriteString(fmt.Sprintf("- %s | source: %s\n", item.Name, item.Source))
			for _, highlight := range item.Highlights {
				builder.WriteString(fmt.Sprintf("  - %s\n", highlight))
			}
		}
	}

	if len(p.PrivateProjects) > 0 {
		builder.WriteString("\nPrivate projects (verified via authenticated GitHub access):\n")
		for _, item := range p.PrivateProjects {
			builder.WriteString(fmt.Sprintf("- %s | source: %s\n", item.Name, item.Source))
			for _, highlight := range item.Highlights {
				builder.WriteString(fmt.Sprintf("  - %s\n", highlight))
			}
		}
	}

	if len(p.PreferredAngles) > 0 {
		builder.WriteString("\nPreferred angles:\n")
		for _, angle := range p.PreferredAngles {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", angle.Name, angle.WhenToUse))
			for _, proofPoint := range angle.ProofPoints {
				builder.WriteString(fmt.Sprintf("  - %s\n", proofPoint))
			}
		}
	}

	writeBulletSection(&builder, "Guardrails", p.Guardrails)
	return strings.TrimSpace(builder.String())
}

func writeBulletSection(builder *strings.Builder, title string, items []string) {
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
