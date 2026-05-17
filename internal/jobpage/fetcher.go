package jobpage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

const maxPageRunes = 18000

const (
	minMeaningfulContentRunes = 240
	readableProxyPrefix       = "https://r.jina.ai/http://"
)

var markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
var markdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
var markdownImageLinkPattern = regexp.MustCompile(`\[!\[[^\]]*\]\([^)]+\)\]\([^)]+\)`)
var prioritizedRoleMarkers = []string{
	"about the role",
	"responsibilities",
	"requirements",
	"what you'll do",
	"what you will do",
	"about the job",
	"job description",
}

type Page struct {
	URL             string
	Title           string
	MetaDescription string
	Content         string
}

type Fetcher struct {
	client *http.Client
}

func NewFetcher(timeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: timeout},
	}
}

func (f *Fetcher) Fetch(ctx context.Context, targetURL string) (Page, error) {
	directPage, directErr := f.fetchHTMLPage(ctx, targetURL)
	if directErr == nil && !isSparsePage(directPage) {
		return directPage, nil
	}

	fallbackPage, fallbackErr := f.fetchReadablePage(ctx, targetURL)
	if fallbackErr == nil && !isSparsePage(fallbackPage) {
		if fallbackPage.Title == "" {
			fallbackPage.Title = directPage.Title
		}
		if fallbackPage.MetaDescription == "" {
			fallbackPage.MetaDescription = directPage.MetaDescription
		}
		return fallbackPage, nil
	}

	if directErr != nil {
		if fallbackErr != nil {
			return Page{}, fmt.Errorf("fetch page %q: direct fetch failed: %v; readable fallback failed: %v", targetURL, directErr, fallbackErr)
		}

		return Page{}, directErr
	}

	return directPage, nil
}

func (f *Fetcher) fetchHTMLPage(ctx context.Context, targetURL string) (Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return Page{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "gojobs/0.1 (+https://github.com/cegh2001)")

	resp, err := f.client.Do(req)
	if err != nil {
		return Page{}, fmt.Errorf("fetch page %q: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Page{}, fmt.Errorf("fetch page %q: unexpected HTTP status %s", targetURL, resp.Status)
	}

	return extractPage(targetURL, resp.Body)
}

func (f *Fetcher) fetchReadablePage(ctx context.Context, targetURL string) (Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readableProxyPrefix+targetURL, nil)
	if err != nil {
		return Page{}, fmt.Errorf("create readable fallback request: %w", err)
	}

	req.Header.Set("User-Agent", "gojobs/0.1 (+https://github.com/cegh2001)")
	req.Header.Set("Accept", "text/plain, text/markdown;q=0.9, */*;q=0.1")

	resp, err := f.client.Do(req)
	if err != nil {
		return Page{}, fmt.Errorf("fetch readable fallback for %q: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Page{}, fmt.Errorf("fetch readable fallback for %q: unexpected HTTP status %s", targetURL, resp.Status)
	}

	return extractReadableFallbackPage(targetURL, resp.Body)
}

func extractPage(targetURL string, reader io.Reader) (Page, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return Page{}, fmt.Errorf("parse HTML: %w", err)
	}

	title := collapseWhitespace(doc.Find("title").First().Text())
	metaDescription, _ := doc.Find(`meta[name="description"]`).Attr("content")
	metaDescription = collapseWhitespace(metaDescription)

	doc.Find("script,style,noscript,svg,form,nav,footer,header").Each(func(_ int, selection *goquery.Selection) {
		selection.Remove()
	})

	root := doc.Find("main").First()
	if root.Length() == 0 {
		root = doc.Find("article").First()
	}
	if root.Length() == 0 {
		root = doc.Find("body").First()
	}

	text := collapseWhitespace(root.Text())
	if text == "" {
		text = collapseWhitespace(doc.Text())
	}

	return Page{
		URL:             targetURL,
		Title:           title,
		MetaDescription: metaDescription,
		Content:         trimRunes(text, maxPageRunes),
	}, nil
}

func extractReadableFallbackPage(targetURL string, reader io.Reader) (Page, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return Page{}, fmt.Errorf("read readable fallback body: %w", err)
	}

	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	content := text
	if markerIndex := strings.Index(text, "Markdown Content:\n"); markerIndex >= 0 {
		content = text[markerIndex+len("Markdown Content:\n"):]
	}

	title := ""
	for _, line := range strings.Split(text, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "Title:") {
			title = collapseWhitespace(strings.TrimPrefix(trimmedLine, "Title:"))
			break
		}
	}

	normalizedContent := normalizeReadableFallbackContent(content)
	return Page{
		URL:     targetURL,
		Title:   title,
		Content: trimRunes(normalizedContent, maxPageRunes),
	}, nil
}

func normalizeReadableFallbackContent(raw string) string {
	if cutoff := strings.Index(raw, "## Other jobs at "); cutoff >= 0 {
		raw = raw[:cutoff]
	}
	if cutoff := strings.Index(raw, "## Hundreds of YC startups are hiring on Work at a Startup."); cutoff >= 0 {
		raw = raw[:cutoff]
	}
	raw = prioritizeRoleSection(raw)

	lines := strings.Split(raw, "\n")
	normalizedLines := make([]string, 0, len(lines))
	replacer := strings.NewReplacer("###", "", "**", "", "`", "")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		trimmedLine = markdownImageLinkPattern.ReplaceAllString(trimmedLine, "")
		trimmedLine = markdownImagePattern.ReplaceAllString(trimmedLine, "")
		trimmedLine = markdownLinkPattern.ReplaceAllString(trimmedLine, "$1")
		trimmedLine = strings.TrimSpace(replacer.Replace(trimmedLine))
		if trimmedLine == "" {
			continue
		}

		normalizedLines = append(normalizedLines, trimmedLine)
	}

	return collapseWhitespace(strings.Join(normalizedLines, "\n"))
}

func prioritizeRoleSection(raw string) string {
	normalized := strings.ToLower(raw)
	bestIndex := -1

	for _, marker := range prioritizedRoleMarkers {
		index := strings.Index(normalized, marker)
		if index < 0 {
			continue
		}

		if bestIndex == -1 || index < bestIndex {
			bestIndex = index
		}
	}

	if bestIndex > 120 {
		return raw[bestIndex:]
	}

	return raw
}

func isSparsePage(page Page) bool {
	normalizedContent := collapseWhitespace(page.Content)
	if normalizedContent == "" {
		return true
	}

	if utf8.RuneCountInString(normalizedContent) < minMeaningfulContentRunes {
		return true
	}

	normalizedTitle := collapseWhitespace(page.Title)
	return normalizedTitle != "" && strings.EqualFold(normalizedContent, normalizedTitle)
}

func collapseWhitespace(raw string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(raw), " "))
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
