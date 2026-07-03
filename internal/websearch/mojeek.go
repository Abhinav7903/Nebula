package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type MojeekProvider struct {
	client *http.Client
}

func NewMojeekProvider(client *http.Client) *MojeekProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &MojeekProvider{client: client}
}

func (m *MojeekProvider) Name() string { return "mojeek" }

func (m *MojeekProvider) IsAvailable() bool { return true }

func (m *MojeekProvider) Search(ctx context.Context, query string, count int) ([]Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://www.mojeek.com/search?q=%s&fmt=html", qEnc)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var results []Result
	seenURL := make(map[string]bool)

	doc.Find("li.results-item, .results-item, .result").Each(func(_ int, s *goquery.Selection) {
		if len(results) >= count {
			return
		}
		linkEl := s.Find("a").First()
		href, exists := linkEl.Attr("href")
		if !exists || href == "" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}
		title := strings.TrimSpace(linkEl.Text())
		if title == "" || seenURL[href] {
			return
		}
		seenURL[href] = true

		snippet := strings.TrimSpace(s.Find("p, .snippet, .description").First().Text())

		results = append(results, Result{
			Title:       title,
			URL:         href,
			Description: snippet,
			Engine:      "mojeek",
		})
	})

	if len(results) > count {
		results = results[:count]
	}
	return results, nil
}
