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

type DuckDuckGoProvider struct {
	client *http.Client
}

func NewDuckDuckGoProvider(client *http.Client) *DuckDuckGoProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &DuckDuckGoProvider{client: client}
}

func (d *DuckDuckGoProvider) Name() string { return "duckduckgo" }

func (d *DuckDuckGoProvider) IsAvailable() bool { return true }

func (d *DuckDuckGoProvider) Search(ctx context.Context, query string, count int) ([]Result, error) {
	results, err := d.searchLite(ctx, query, count)
	if err != nil || len(results) == 0 {
		html, htmlErr := d.searchHTML(ctx, query, count)
		if htmlErr == nil && len(html) > 0 {
			return html, nil
		}
	}
	return results, err
}

func (d *DuckDuckGoProvider) searchLite(ctx context.Context, query string, count int) ([]Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://lite.duckduckgo.com/lite/?q=%s", qEnc)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := d.client.Do(req)
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

	doc.Find("a.result-link").Each(func(_ int, s *goquery.Selection) {
		if len(results) >= count {
			return
		}
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}
		title := strings.TrimSpace(s.Text())
		if title == "" || seenURL[href] {
			return
		}
		seenURL[href] = true

		var snippet string
		s.ParentsFiltered("tr").Next().Find("td.snippet").Each(func(_ int, sn *goquery.Selection) {
			snippet = strings.TrimSpace(sn.Text())
		})

		results = append(results, Result{
			Title:       title,
			URL:         href,
			Description: snippet,
			Engine:      "duckduckgo",
		})
	})

	if len(results) > count {
		results = results[:count]
	}
	return results, nil
}

func (d *DuckDuckGoProvider) searchHTML(ctx context.Context, query string, count int) ([]Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", qEnc)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := d.client.Do(req)
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

	doc.Find("a.result__a").Each(func(_ int, s *goquery.Selection) {
		if len(results) >= count {
			return
		}
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}
		href = cleanDuckDuckGoRedirect(href)
		if !strings.HasPrefix(href, "http") {
			return
		}
		title := strings.TrimSpace(s.Text())
		if title == "" || seenURL[href] {
			return
		}
		seenURL[href] = true

		snippet := strings.TrimSpace(s.Parent().Find(".result__snippet").Text())
		if snippet == "" {
			snippet = strings.TrimSpace(s.ParentsFiltered(".result").Find(".result__snippet").Text())
		}

		results = append(results, Result{
			Title:       title,
			URL:         href,
			Description: snippet,
			Engine:      "duckduckgo",
		})
	})

	if len(results) > count {
		results = results[:count]
	}
	return results, nil
}

func cleanDuckDuckGoRedirect(raw string) string {
	return cleanSearchRedirect(raw)
}
