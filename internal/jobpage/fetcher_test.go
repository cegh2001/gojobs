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
