package searchengine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/yourusername/nebula/internal/collectors"
)

type Collector struct {
	client *http.Client
}

func New() *Collector {
	return &Collector{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Collector) Name() string            { return "searchengine" }
func (c *Collector) SupportedTypes() []string { return []string{"all"} }
func (c *Collector) RequiresKey() bool        { return false }

var (
	reName    = regexp.MustCompile(`[A-Z][a-z]+ [A-Z][a-z]+`)
	reProfile = regexp.MustCompile(`https?://(?:www\.)?(?:twitter\.com|linkedin\.com|github\.com|facebook\.com|instagram\.com)/[a-zA-Z0-9_]+`)
)

func cleanHTML(s string) string {
	s = strings.ReplaceAll(s, "<strong>", "")
	s = strings.ReplaceAll(s, "</strong>", "")
	s = strings.ReplaceAll(s, "&#8211;", "–")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&raquo;", "»")
	s = strings.ReplaceAll(s, "&rsaquo;", "›")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	queries := c.buildQueries(query, qtype)

	var results []collectors.Result
	for _, q := range queries {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		res, err := c.searchMojeek(ctx, q)
		if err == nil {
			results = append(results, res...)
		}

		res, err = c.searchGoogle(ctx, q)
		if err == nil {
			results = append(results, res...)
		}

		res, err = c.searchDuckDuckGo(ctx, q)
		if err == nil {
			results = append(results, res...)
		}
	}

	return results, nil
}

func (c *Collector) buildQueries(query, qtype string) []string {
	base := []string{query}

	switch qtype {
	case "email":
		base = append(base,
			fmt.Sprintf(`"%s" site:linkedin.com`, query),
			fmt.Sprintf(`"%s" site:github.com`, query),
			fmt.Sprintf(`"%s" contact`, query),
		)
	case "person_name":
		parts := strings.Fields(query)
		if len(parts) >= 2 {
			fullName := fmt.Sprintf(`"%s %s"`, parts[0], parts[1])
			base = append(base,
				fullName,
				fullName+" github",
				fullName+" linkedin",
				fullName+" twitter",
				fullName+" site:github.com",
				fullName+" site:linkedin.com",
			)
		}
	case "username":
		base = append(base,
			query+" github",
			query+" twitter",
			query+" site:github.com",
		)
	case "phone":
		base = append(base,
			query+" site:whitepages.com",
			query+" site:truecaller.com",
		)
	case "ipv4", "ipv6":
		base = append(base, query+" site:whois.com")
	}

	return base
}

func (c *Collector) searchMojeek(ctx context.Context, query string) ([]collectors.Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://www.mojeek.com/search?q=%s", qEnc)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	content := string(body)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	var results []collectors.Result
	seenURL := make(map[string]bool)

	doc.Find("li").Each(func(_ int, s *goquery.Selection) {
		linkEl := s.Find("a.ob")
		href, exists := linkEl.Attr("href")
		if !exists || href == "" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}

		title := strings.TrimSpace(s.Find("a.title").First().Text())
		snippet := strings.TrimSpace(s.Find("p.s").First().Text())

		if title == "" {
			return
		}
		if seenURL[href] {
			return
		}
		seenURL[href] = true

		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "searchengine",
			Type:        "web_result",
			Title:       title,
			Description: snippet,
			URL:         href,
			Data: map[string]any{
				"url":     href,
				"query":   query,
				"snippet": snippet,
				"engine":  "mojeek",
			},
			Tags:       []string{"web", "search", "mojeek", "link"},
			Confidence: 0.7,
			Source:     "mojeek.com",
			FoundAt:    time.Now(),
		})
	})

	extracted := c.extractNames(content, query)
	for i := range extracted {
		extracted[i].Source = "mojeek.com"
	}
	results = append(results, extracted...)

	extracted = c.extractProfiles(content, query)
	for i := range extracted {
		extracted[i].Source = "mojeek.com"
	}
	results = append(results, extracted...)

	return results, nil
}

var googleBlockPatterns = []string{
	"unusual traffic",
	"captcha",
	"https://www.google.com/sorry/",
	"g-recaptcha",
	"your computer or network may be sending automated queries",
}

func isBlocked(body string) bool {
	lower := strings.ToLower(body)
	for _, p := range googleBlockPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func (c *Collector) searchGoogle(ctx context.Context, query string) ([]collectors.Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://www.google.com/search?q=%s&hl=en", qEnc)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if isBlocked(string(body)) {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var results []collectors.Result
	seenURL := make(map[string]bool)

	doc.Find("div.g").Each(func(_ int, s *goquery.Selection) {
		linkEl := s.Find("a").First()
		href, exists := linkEl.Attr("href")
		if !exists || href == "" {
			return
		}
		href = strings.TrimPrefix(href, "/url?q=")
		if idx := strings.IndexByte(href, '&'); idx != -1 {
			href = href[:idx]
		}
		if !strings.HasPrefix(href, "http") {
			return
		}

		title := strings.TrimSpace(s.Find("h3").First().Text())
		snippet := strings.TrimSpace(s.Find("div.VwiC3b, span.st, div[data-sncf], div.lDhqUd").First().Text())

		if title == "" || href == "" {
			return
		}
		if seenURL[href] {
			return
		}
		seenURL[href] = true

		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "searchengine",
			Type:        "web_result",
			Title:       title,
			Description: snippet,
			URL:         href,
			Data: map[string]any{
				"url":     href,
				"query":   query,
				"snippet": snippet,
				"engine":  "google",
			},
			Tags:       []string{"web", "search", "google", "link"},
			Confidence: 0.7,
			Source:     "google.com",
			FoundAt:    time.Now(),
		})
	})

	names := c.extractNames(doc.Text(), query)
	for i := range names {
		names[i].Source = "google.com"
	}
	results = append(results, names...)

	profiles := c.extractProfiles(doc.Text(), query)
	for i := range profiles {
		profiles[i].Source = "google.com"
	}
	results = append(results, profiles...)

	return results, nil
}

func (c *Collector) searchDuckDuckGo(ctx context.Context, query string) ([]collectors.Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://lite.duckduckgo.com/lite/?q=%s", qEnc)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.client.Do(req)
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

	var results []collectors.Result
	seenURL := make(map[string]bool)

	doc.Find("a.result-link").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}

		title := strings.TrimSpace(s.Text())
		if title == "" {
			return
		}
		if seenURL[href] {
			return
		}
		seenURL[href] = true

		var snippet string
		s.ParentsFiltered("tr").Next().Find("td.snippet").Each(func(_ int, sn *goquery.Selection) {
			snippet = strings.TrimSpace(sn.Text())
		})

		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "searchengine",
			Type:        "web_result",
			Title:       title,
			Description: snippet,
			URL:         href,
			Data: map[string]any{
				"url":     href,
				"query":   query,
				"snippet": snippet,
				"engine":  "duckduckgo",
			},
			Tags:       []string{"web", "search", "duckduckgo", "link"},
			Confidence: 0.7,
			Source:     "duckduckgo.com",
			FoundAt:    time.Now(),
		})
	})

	content := string(body)
	extracted := c.extractNames(content, query)
	for i := range extracted {
		extracted[i].Source = "duckduckgo.com"
	}
	results = append(results, extracted...)

	extracted = c.extractProfiles(content, query)
	for i := range extracted {
		extracted[i].Source = "duckduckgo.com"
	}
	results = append(results, extracted...)

	return results, nil
}

func (c *Collector) extractNames(content, query string) []collectors.Result {
	names := reName.FindAllString(content, 5)
	seen := make(map[string]bool)
	var results []collectors.Result
	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true
		if strings.EqualFold(n, query) {
			continue
		}
		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "searchengine",
			Type:        "associated_person",
			Title:       n,
			Description: fmt.Sprintf("Person name found in search results for '%s'", query),
			Data: map[string]any{
				"name":  n,
				"query": query,
			},
			Tags:       []string{"person", "association"},
			Confidence: 0.25,
			FoundAt:    time.Now(),
		})
	}
	return results
}

func (c *Collector) extractProfiles(content, query string) []collectors.Result {
	profiles := reProfile.FindAllString(content, 5)
	seen := make(map[string]bool)
	var results []collectors.Result
	for _, p := range profiles {
		if seen[p] {
			continue
		}
		seen[p] = true
		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "searchengine",
			Type:        "social_profile",
			Title:       fmt.Sprintf("Profile: %s", p),
			Description: fmt.Sprintf("Social profile found in search results for '%s'", query),
			URL:         p,
			Data: map[string]any{
				"url":   p,
				"query": query,
			},
			Tags:       []string{"social", "profile"},
			Confidence: 0.35,
			FoundAt:    time.Now(),
		})
	}
	return results
}
