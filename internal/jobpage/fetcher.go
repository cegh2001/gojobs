package jobpage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

const maxPageRunes = 18000

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
