package jobpage

import (
	"strings"
	"testing"
)

func TestExtractPage(t *testing.T) {
	rawHTML := `
	<html>
	  <head>
	    <title>Founding AI Engineer</title>
	    <meta name="description" content="Automate distributor workflows with AI" />
	  </head>
	  <body>
	    <header>This should not be used</header>
	    <main>
	      <h1>Founding AI Engineer</h1>
	      <p>Build AI systems for distributor operations.</p>
	      <script>console.log("noise")</script>
	    </main>
	  </body>
	</html>`

	page, err := extractPage("https://example.com/jobs/1", strings.NewReader(rawHTML))
	if err != nil {
		t.Fatalf("extractPage() error = %v", err)
	}

	if page.Title != "Founding AI Engineer" {
		t.Fatalf("unexpected title: %q", page.Title)
	}

	if page.MetaDescription != "Automate distributor workflows with AI" {
		t.Fatalf("unexpected meta description: %q", page.MetaDescription)
	}

	if strings.Contains(page.Content, "noise") {
		t.Fatalf("content should not contain script text: %q", page.Content)
	}

	if !strings.Contains(page.Content, "Build AI systems for distributor operations.") {
		t.Fatalf("content should contain main text: %q", page.Content)
	}
}

func TestExtractReadableFallbackPage(t *testing.T) {
	raw := `Title: AI Engineer at ClaimSorted | Y Combinator's Work at a Startup

URL Source: https://www.workatastartup.com/jobs/89001

Markdown Content:
[![Image 1: Y Combinator](https://bookface-static.example/logo.png)](https://www.workatastartup.com/)
About ClaimSorted

ClaimSorted is building better insurance operations.

About the role

### **Responsibilities**

* Build AI systems that improve accuracy
* Work with insurance specialists

Technology

NextJS, Typescript, Postgres, GCP

## Other jobs at ClaimSorted

Ignore this tail section`

	page, err := extractReadableFallbackPage("https://www.workatastartup.com/jobs/89001", strings.NewReader(raw))
	if err != nil {
		t.Fatalf("extractReadableFallbackPage() error = %v", err)
	}

	if page.Title != "AI Engineer at ClaimSorted | Y Combinator's Work at a Startup" {
		t.Fatalf("unexpected title: %q", page.Title)
	}

	checks := []string{"Responsibilities", "Build AI systems that improve accuracy", "Technology", "NextJS, Typescript, Postgres, GCP"}
	for _, check := range checks {
		if !strings.Contains(page.Content, check) {
			t.Fatalf("content should contain %q, got %q", check, page.Content)
		}
	}

	if !strings.HasPrefix(page.Content, "About the role") {
		t.Fatalf("content should prioritize the role section, got %q", page.Content)
	}

	forbidden := []string{"Image 1", "Ignore this tail section"}
	for _, forbiddenText := range forbidden {
		if strings.Contains(page.Content, forbiddenText) {
			t.Fatalf("content should not contain %q, got %q", forbiddenText, page.Content)
		}
	}
}

func TestIsSparsePage(t *testing.T) {
	if !isSparsePage(Page{Title: "AI Engineer", Content: "AI Engineer"}) {
		t.Fatalf("expected title-only page to be sparse")
	}

	if isSparsePage(Page{Title: "AI Engineer", Content: strings.Repeat("Detailed role context ", 20)}) {
		t.Fatalf("expected detailed content not to be sparse")
	}
}
