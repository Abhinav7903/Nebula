package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
)

type FetchedPage struct {
	URL        string              `json:"url"`
	Title      string              `json:"title"`
	Content    string              `json:"content"`
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Engine     string              `json:"engine"`
}

type FetchOptions struct {
	MaxSize   int
	Timeout   time.Duration
	UserAgent string
	Format    string // "text" (default), "markdown", or "html"
}

const maxFetchSize = 5 * 1024 * 1024 // 5MB

func DefaultFetchOptions() FetchOptions {
	return FetchOptions{
		MaxSize:   maxFetchSize,
		Timeout:   15 * time.Second,
		UserAgent: "nebula/1.0",
		Format:    "text",
	}
}

func (e *Engine) FetchURL(ctx context.Context, urlStr string, opts FetchOptions) (*FetchedPage, error) {
	if opts.MaxSize == 0 {
		opts = DefaultFetchOptions()
	}
	if opts.Timeout == 0 {
		opts.Timeout = 15 * time.Second
	}
	if opts.UserAgent == "" {
		opts.UserAgent = DefaultFetchOptions().UserAgent
	}

	fetchCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", opts.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, int64(opts.MaxSize))
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	raw := string(body)
	contentType := resp.Header.Get("Content-Type")

	var content string
	switch opts.Format {
	case "html":
		content = raw
	case "markdown":
		if strings.Contains(contentType, "text/html") {
			content, err = md.ConvertString(raw)
			if err != nil {
				content = raw
			}
		} else {
			content = raw
		}
	default: // "text"
		if strings.Contains(contentType, "text/html") {
			content = extractText(raw)
		} else {
			content = raw
		}
	}
	if len(content) > 50000 {
		content = content[:50000]
	}

	title := extractTitle(raw)

	headers := make(map[string][]string)
	for k, v := range resp.Header {
		headers[k] = v
	}

	page := &FetchedPage{
		URL:        urlStr,
		Title:      title,
		Content:    content,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Engine:     "fetch",
	}

	return page, nil
}

func (e *Engine) FetchURLs(ctx context.Context, urls []string, opts FetchOptions) []*FetchedPage {
	type result struct {
		page *FetchedPage
		err  error
	}

	ch := make(chan result, len(urls))
	for _, u := range urls {
		go func(urlStr string) {
			page, err := e.FetchURL(ctx, urlStr, opts)
			if err != nil {
				ch <- result{err: err}
				return
			}
			ch <- result{page: page}
		}(u)
	}

	var pages []*FetchedPage
	for i := 0; i < len(urls); i++ {
		r := <-ch
		if r.err == nil && r.page != nil {
			pages = append(pages, r.page)
		}
	}
	return pages
}

func extractText(raw string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err != nil {
		return ""
	}
	doc.Find("script, style, nav, header, footer, aside, .sidebar, .nav, .menu, .footer, .ad, noscript, iframe, svg, form, .cookie, .popup, .modal").Remove()
	text := doc.Find("article, main, [role=main], body").First().Text()
	text = strings.Join(strings.Fields(text), " ")
	return text
}

func extractTitle(htmlStr string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if len(title) > 200 {
		title = title[:200]
	}
	return title
}
