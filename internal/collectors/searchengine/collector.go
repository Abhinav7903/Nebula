package searchengine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/yourusername/nebula/internal/collectors"
)

type Collector struct {
	client       *http.Client
	apiKey       string
	searchEngineID string
}

func New() *Collector {
	return &Collector{
		client:         &http.Client{Timeout: 15 * time.Second},
		apiKey:         os.Getenv("GOOGLE_CUSTOM_SEARCH_API_KEY"),
		searchEngineID: os.Getenv("GOOGLE_CUSTOM_SEARCH_ENGINE_ID"),
	}
}

func (c *Collector) Name() string            { return "searchengine" }
func (c *Collector) SupportedTypes() []string { return []string{"all"} }
func (c *Collector) RequiresKey() bool        { return c.apiKey != "" && c.searchEngineID != "" }

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

		// Use DuckDuckGo as primary (free, reliable, no API key needed)
		res, err := c.searchDuckDuckGo(ctx, q)
		if err == nil && len(res) > 0 {
			results = append(results, res...)
		}

		// Try Bing as secondary (free scraping, often works better than Google)
		res, err = c.searchBing(ctx, q)
		if err == nil && len(res) > 0 {
			results = append(results, res...)
		}

		// Only use Google API if credentials are provided (paid but high quality)
		if c.apiKey != "" && c.searchEngineID != "" {
			res, err = c.searchGoogleAPI(ctx, q)
			if err == nil && len(res) > 0 {
				results = append(results, res...)
			}
		}

		// Skip direct Google scraping - too many CAPTCHAs
		// Skip Mojeek - returns too much noise
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

type GoogleSearchResponse struct {
	Kind       string `json:"kind"`
	URL        struct {
		Type      string `json:"type"`
		Template  string `json:"template"`
	} `json:"url"`
	Queries    struct {
		Request []struct {
			Title       string `json:"title"`
			TotalResults string `json:"totalResults"`
			SearchTerms string `json:"searchTerms"`
			Count       int    `json:"count"`
			StartIndex  int    `json:"startIndex"`
			InputEncoding string `json:"inputEncoding"`
			OutputEncoding string `json:"outputEncoding"`
		} `json:"request"`
	} `json:"queries"`
	Context    struct {
		Title string `json:"title"`
	} `json:"context"`
	Items      []struct {
		Kind            string `json:"kind"`
		Title           string `json:"title"`
		HTMLTitle       string `json:"htmlTitle"`
		Link            string `json:"link"`
		DisplayLink     string `json:"displayLink"`
		Snippet         string `json:"snippet"`
		HTMLSnippet     string `json:"htmlSnippet"`
		CSEImage        struct {
			Src string `json:"src"`
		} `json:"cse_image"`
		CSEPublicURL struct {
			Href string `json:"href"`
		} `json:"cse_public_url"`
		Mime            string `json:"mime"`
		FileFormat      string `json:"fileFormat"`
		ImageObject     struct {
			Width  string `json:"width"`
			Height string `json:"height"`
		} `json:"imageObject"`
		Pagemap         struct {
			CSEThumbnail []struct {
				Src    string `json:"src"`
				Width  string `json:"width"`
				Height string `json:"height"`
			} `json:"cse_thumbnail,omitempty"`
			Metatags   []map[string]string `json:"metatags,omitempty"`
		} `json:"pagemap,omitempty"`
	} `json:"items"`
	SearchInformation struct {
		SearchTime       float64 `json:"searchTime"`
		FormattedSearchTime string `json:"formattedSearchTime"`
		TotalResults     string `json:"totalResults"`
		FormattedTotalResults string `json:"formattedTotalResults"`
	} `json:"searchInformation"`
	SERPCustomVariables struct {
		SerpapiIframeSrc string `json:"serpapi_iframe_src"`
		SerpapiIframeSrcV2 string `json:"serpapi_iframe_src_v2"`
	} `json:"serpapi_custom_variables,omitempty"`
}

func (c *Collector) searchGoogleAPI(ctx context.Context, query string) ([]collectors.Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s",
		c.apiKey,
		c.searchEngineID,
		qEnc,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var googleResp GoogleSearchResponse
	if err := json.Unmarshal(body, &googleResp); err != nil {
		return nil, err
	}

	var results []collectors.Result
	seenURL := make(map[string]bool)

	for _, item := range googleResp.Items {
		if item.Link == "" || item.Title == "" {
			continue
		}
		if seenURL[item.Link] {
			continue
		}
		seenURL[item.Link] = true

		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "searchengine",
			Type:        "web_result",
			Title:       item.Title,
			Description: item.Snippet,
			URL:         item.Link,
			Data: map[string]any{
				"url":        item.Link,
				"query":      query,
				"snippet":    item.Snippet,
				"engine":     "google_api",
				"displayLink": item.DisplayLink,
			},
			Tags:       []string{"web", "search", "google_api", "link"},
			Confidence: 0.85,
			Source:     "google.com (API)",
			FoundAt:    time.Now(),
		})
	}

	return results, nil
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

func (c *Collector) searchBing(ctx context.Context, query string) ([]collectors.Result, error) {
	qEnc := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s", qEnc)

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

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var results []collectors.Result
	seenURL := make(map[string]bool)

	doc.Find("li.b_algo").Each(func(_ int, s *goquery.Selection) {
		linkEl := s.Find("h2 a").First()
		href, exists := linkEl.Attr("href")
		if !exists || href == "" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}

		title := strings.TrimSpace(linkEl.Text())
		snippet := strings.TrimSpace(s.Find("div.b_caption p").First().Text())

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
				"engine":  "bing",
			},
			Tags:       []string{"web", "search", "bing", "link"},
			Confidence: 0.7,
			Source:     "bing.com",
			FoundAt:    time.Now(),
		})
	})

	content := string(body)
	extracted := c.extractNames(content, query)
	for i := range extracted {
		extracted[i].Source = "bing.com"
	}
	results = append(results, extracted...)

	extracted = c.extractProfiles(content, query)
	for i := range extracted {
		extracted[i].Source = "bing.com"
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
