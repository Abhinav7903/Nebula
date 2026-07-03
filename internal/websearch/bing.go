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

type BingProvider struct {
	client *http.Client
}

func NewBingProvider(client *http.Client) *BingProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &BingProvider{client: client}
}

func (b *BingProvider) Name() string { return "bing" }

func (b *BingProvider) IsAvailable() bool { return true }

func (b *BingProvider) Search(ctx context.Context, query string, count int) ([]Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", qEnc, count)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := b.client.Do(req)
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

	doc.Find("li.b_algo").Each(func(_ int, s *goquery.Selection) {
		if len(results) >= count {
			return
		}
		linkEl := s.Find("h2 a").First()
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

		snippet := strings.TrimSpace(s.Find("div.b_caption p").First().Text())
		if snippet == "" {
			snippet = strings.TrimSpace(s.Find(".b_lineclamp2").First().Text())
		}

		results = append(results, Result{
			Title:       title,
			URL:         href,
			Description: snippet,
			Engine:      "bing",
		})
	})

	if len(results) > count {
		results = results[:count]
	}
	return results, nil
}
