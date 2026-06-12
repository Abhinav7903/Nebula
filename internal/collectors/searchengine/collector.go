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
	"github.com/Abhinav7903/nebula/internal/collectors"
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

	extracted := c.extractNamesFromResults(results, query)
	for i := range extracted {
		extracted[i].Source = "duckduckgo.com"
	}
	results = append(results, extracted...)

	content := string(body)
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

	extracted := c.extractNamesFromResults(results, query)
	for i := range extracted {
		extracted[i].Source = "bing.com"
	}
	results = append(results, extracted...)

	content := string(body)
	extracted = c.extractProfiles(content, query)
	for i := range extracted {
		extracted[i].Source = "bing.com"
	}
	results = append(results, extracted...)

	return results, nil
}

var nonPersonNames = map[string]bool{
	"saudi arabia": true, "south africa": true, "south korea": true,
	"north korea": true, "new york": true, "new zealand": true,
	"los angeles": true, "san francisco": true, "san diego": true,
	"las vegas": true, "buenos aires": true, "rio de janeiro": true,
	"costa rica": true, "puerto rico": true, "united states": true,
	"united kingdom": true, "united nations": true, "united arab emirates": true,
	"white house": true, "supreme court": true, "european union": true,
	"hong kong": true, "sri lanka": true, "central park": true,
	"grand canyon": true, "rhode island": true,
}

var commonCapitalized = map[string]bool{
	"default": true, "profile": true, "accessibility": true, "feedback": true,
	"account": true, "rewards": true, "microsoft": true, "picture": true,
	"search": true, "filter": true, "additional": true, "results": true,
	"explore": true, "more": true, "report": true,
	"history": true, "people": true, "also": true, "copilot": true,
	"bing": true, "google": true, "youtube": true, "facebook": true,
	"twitter": true, "linkedin": true, "instagram": true, "github": true,
	"direct": true, "france": true, "livret": true, "epargne": true,
	"tube": true, "help": true, "center": true, "community": true,
	"guidelines": true, "creator": true, "support": true,
	"turn": true, "restricted": true, "mode": true, "partner": true,
	"program": true, "play": true, "official": true, "your": true,
	"see": true, "share": true, "menu": true, "home": true,
	"sign": true, "log": true, "about": true, "contact": true, "click": true,
	"read": true, "next": true, "previous": true, "related": true,
	"powered": true, "rights": true, "terms": true,
	"policy": true, "all": true, "news": true, "video": true, "image": true,
	"maps": true, "shopping": true, "books": true, "finance": true,
	"settings": true, "preferences": true, "language": true,
	"region": true, "download": true, "delete": true,
	"update": true, "install": true, "configure": true, "setup": true,
	"welcome": true, "getting": true, "started": true, "learn": true,
	"tutorial": true, "guide": true, "documentation": true, "reference": true,
	"api": true, "sdk": true, "tool": true, "tools": true, "resource": true,
	"resources": true, "blog": true, "forum": true, "discuss": true,
	"answer": true, "question": true, "topic": true, "thread": true,
	"reply": true, "post": true, "article": true, "site": true,
	"web": true, "app": true, "application": true, "service": true,
	"product": true, "change": true, "changelog": true, "license": true,
	"copyright": true, "cookie": true, "cookies": true, "data": true,
	"security": true, "password": true, "email": true, "address": true,
	"phone": true, "number": true, "code": true, "repo": true,
	"repository": true, "issue": true, "pull": true, "request": true,
	"commit": true, "branch": true, "master": true, "main": true,
	"dev": true, "develop": true, "bug": true,
	"fix": true, "hotfix": true, "docs": true, "test": true, "tests": true,
	"build": true, "ci": true, "cd": true, "deploy": true,
	"tag": true, "note": true, "notes": true,
	"star": true, "stars": true, "fork": true, "forks": true,
	"watch": true, "watcher": true, "watchers": true, "contributor": true,
	"contributors": true, "author": true, "authors": true,
	"maintainer": true, "maintainers": true, "owner": true, "owners": true,
	"member": true, "members": true, "team": true, "teams": true,
	"organization": true, "org": true, "company": true,
}

func (c *Collector) extractNamesFromResults(results []collectors.Result, query string) []collectors.Result {
	var sb strings.Builder
	for _, r := range results {
		if r.Type == "web_result" {
			sb.WriteString(r.Title)
			sb.WriteString(" ")
			sb.WriteString(r.Description)
			sb.WriteString(" ")
		}
	}
	return c.extractNames(sb.String(), query)
}

func (c *Collector) extractNames(content, query string) []collectors.Result {
	matches := reName.FindAllString(content, -1)
	seen := make(map[string]bool)
	var results []collectors.Result

	for _, name := range matches {
		if seen[name] {
			continue
		}
		seen[name] = true

		lower := strings.ToLower(name)
		if strings.Contains(lower, strings.ToLower(query)) {
			continue
		}

		if nonPersonNames[lower] {
			continue
		}

		parts := strings.Fields(name)
		if len(parts) >= 2 && (commonCapitalized[strings.ToLower(parts[0])] || commonCapitalized[strings.ToLower(parts[1])]) {
			continue
		}

		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "searchengine",
			Type:        "person_name",
			Title:       name,
			Description: fmt.Sprintf("Extracted from search results for: %s", query),
			Data: map[string]any{
				"extracted_name": name,
				"source_query":   query,
			},
			Tags:       []string{"name", "person", "extracted"},
			Confidence: 0.5,
			Source:     "search_content",
			FoundAt:    time.Now(),
		})
	}

	return results
}

func (c *Collector) extractProfiles(content, query string) []collectors.Result {
	matches := reProfile.FindAllString(content, -1)
	seen := make(map[string]bool)
	var results []collectors.Result

	for _, profile := range matches {
		if seen[profile] {
			continue
		}
		seen[profile] = true

		var platform string
		switch {
		case strings.Contains(profile, "twitter.com"):
			platform = "twitter"
		case strings.Contains(profile, "linkedin.com"):
			platform = "linkedin"
		case strings.Contains(profile, "github.com"):
			platform = "github"
		case strings.Contains(profile, "facebook.com"):
			platform = "facebook"
		case strings.Contains(profile, "instagram.com"):
			platform = "instagram"
		}

		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "searchengine",
			Type:        "social_profile",
			Title:       platform + " profile",
			Description: fmt.Sprintf("Profile found in search results for: %s", query),
			URL:         profile,
			Data: map[string]any{
				"profile_url": profile,
				"platform":    platform,
				"source_query": query,
			},
			Tags:       []string{"social", "profile", platform, "extracted"},
			Confidence: 0.6,
			Source:     "search_content",
			FoundAt:    time.Now(),
		})
	}

	return results
}