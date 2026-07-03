package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type GoogleProvider struct {
	client         *http.Client
	apiKey         string
	searchEngineID string
}

func NewGoogleProvider(client *http.Client, apiKey, searchEngineID string) *GoogleProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &GoogleProvider{
		client:         client,
		apiKey:         apiKey,
		searchEngineID: searchEngineID,
	}
}

func (g *GoogleProvider) Name() string { return "google" }

func (g *GoogleProvider) IsAvailable() bool { return true }

func (g *GoogleProvider) Search(ctx context.Context, query string, count int) ([]Result, error) {
	results, htmlErr := g.searchHTML(ctx, query, count)
	if len(results) > 0 {
		return results, nil
	}

	if g.apiKey != "" && g.searchEngineID != "" {
		apiResults, apiErr := g.searchAPI(ctx, query, count)
		if len(apiResults) > 0 {
			return apiResults, nil
		}
		if htmlErr != nil {
			return nil, fmt.Errorf("google html failed: %v; api failed: %v", htmlErr, apiErr)
		}
		return nil, apiErr
	}

	if htmlErr != nil {
		return nil, htmlErr
	}
	return nil, fmt.Errorf("google search returned no organic results")
}

type googleAPIResponse struct {
	Items []struct {
		Title       string `json:"title"`
		Link        string `json:"link"`
		DisplayLink string `json:"displayLink"`
		Snippet     string `json:"snippet"`
	} `json:"items"`
	SearchInformation struct {
		TotalResults string `json:"totalResults"`
	} `json:"searchInformation"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (g *GoogleProvider) searchAPI(ctx context.Context, query string, count int) ([]Result, error) {
	if count < 1 {
		count = 10
	}
	if count > 10 {
		count = 10
	}
	qEnc := url.QueryEscape(query)
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		url.QueryEscape(g.apiKey), url.QueryEscape(g.searchEngineID), qEnc, count,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp googleAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("google api error: %s", apiResp.Error.Message)
	}

	var results []Result
	for i, item := range apiResp.Items {
		if item.Link == "" || item.Title == "" {
			continue
		}
		results = append(results, Result{
			Title:       item.Title,
			URL:         item.Link,
			Description: item.Snippet,
			Engine:      "google",
			Rank:        i + 1,
		})
	}

	return results, nil
}

func (g *GoogleProvider) searchHTML(ctx context.Context, query string, count int) ([]Result, error) {
	if count < 1 {
		count = 10
	}
	if count > 20 {
		count = 20
	}
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://www.google.com/search?q=%s&hl=en&num=%d&pws=0&filter=0", qEnc, count)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google html status: %s", resp.Status)
	}

	content := string(body)
	if isGoogleBlocked(content) {
		return nil, fmt.Errorf("google blocked request or returned a consent/captcha page")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	var results []Result
	seenURL := make(map[string]bool)

	addResult := func(linkEl *goquery.Selection, container *goquery.Selection) {
		if len(results) >= count {
			return
		}
		href, exists := linkEl.Attr("href")
		if !exists || href == "" {
			return
		}
		href = cleanGoogleURL(href)
		cleanURL, ok := normalizeResultURL(href)
		if !ok || seenURL[canonicalResultKey(cleanURL)] || isSearchNoiseURL(cleanURL) {
			return
		}

		title := normalizeWhitespace(linkEl.Find("h3").First().Text())
		if title == "" {
			title = normalizeWhitespace(linkEl.Text())
		}
		if title == "" {
			return
		}

		snippet := googleSnippet(container)
		seenURL[canonicalResultKey(cleanURL)] = true
		results = append(results, Result{
			Title:       title,
			URL:         cleanURL,
			Description: snippet,
			Engine:      "google",
			Rank:        len(results) + 1,
		})
	}

	doc.Find("#search a[href]").Each(func(_ int, linkEl *goquery.Selection) {
		if len(results) >= count || linkEl.Find("h3").Length() == 0 {
			return
		}
		container := linkEl.Closest("div.g, div.MjjYud, div[data-hveid], div")
		addResult(linkEl, container)
	})

	if len(results) == 0 {
		doc.Find("div.g, div[data-hveid]").Each(func(_ int, s *goquery.Selection) {
			if len(results) >= count {
				return
			}
			addResult(s.Find("a[href]").First(), s)
		})
	}

	return results, nil
}

func googleSnippet(container *goquery.Selection) string {
	selectors := []string{
		"div.VwiC3b",
		"span.aCOpRe",
		"span.st",
		"div[data-sncf]",
		".IsZvec",
		".kb0PBd",
	}
	for _, selector := range selectors {
		snippet := normalizeWhitespace(container.Find(selector).First().Text())
		if snippet != "" {
			return snippet
		}
	}
	return ""
}

var googleBlockPatterns = []string{
	"unusual traffic",
	"captcha",
	"https://www.google.com/sorry/",
	"g-recaptcha",
	"consent.google.com",
	"before you continue to google",
	"your computer or network may be sending automated queries",
}

func isGoogleBlocked(body string) bool {
	lower := strings.ToLower(body)
	for _, p := range googleBlockPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func cleanGoogleURL(raw string) string {
	return cleanSearchRedirect(raw)
}
