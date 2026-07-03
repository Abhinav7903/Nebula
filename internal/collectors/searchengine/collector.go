package searchengine

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/Abhinav7903/nebula/internal/collectors"
	"github.com/Abhinav7903/nebula/internal/websearch"
	"github.com/google/uuid"
)

type Collector struct {
	engine *websearch.Engine
}

func New(engine *websearch.Engine) *Collector {
	return &Collector{engine: engine}
}

func (c *Collector) Name() string             { return "searchengine" }
func (c *Collector) SupportedTypes() []string { return []string{"all"} }
func (c *Collector) RequiresKey() bool        { return false }

var (
	reWord    = regexp.MustCompile(`[A-Za-z]+`)
	reProfile = regexp.MustCompile(`https?://(?:www\.)?(?:twitter\.com|linkedin\.com|github\.com|facebook\.com|instagram\.com)/[a-zA-Z0-9_]+`)

	commonNouns = map[string]bool{
		"account": true, "ticket": true, "notification": true, "notifications": true,
		"display": true, "manager": true, "remote": true,
		"discover": true, "platform": true, "workplace": true, "digital": true,
		"laden": true, "descarga": true, "download": true, "downloaden": true,
		"herunterladen": true, "install": true,
		"computer": true, "mobile": true, "mobilger": true, "mobilgerät": true,
		"access": true, "accessibility": true, "feedback": true, "rewards": true,
		"picture": true, "filter": true, "additional": true, "explore": true,
		"report": true, "history": true, "people": true, "also": true,
		"copilot": true, "direct": true, "france": true, "livret": true,
		"epargne": true, "tube": true, "center": true, "creator": true,
		"partner": true, "program": true, "official": true,
		"menu": true, "home": true, "sign": true, "click": true,
		"next": true, "previous": true, "related": true, "powered": true,
		"rights": true, "terms": true, "policy": true, "maps": true,
		"shopping": true, "books": true, "finance": true, "preferences": true,
		"language": true, "region": true, "delete": true, "configure": true,
		"setup": true, "welcome": true, "getting": true, "started": true,
		"tutorial": true, "guide": true, "documentation": true, "reference": true,
		"sdk": true, "resource": true, "resources": true, "blog": true,
		"forum": true, "discuss": true, "answer": true, "question": true,
		"topic": true, "thread": true, "reply": true, "article": true,
		"application": true, "service": true, "product": true,
		"changelog": true, "license": true, "copyright": true, "cookie": true,
		"cookies": true, "password": true, "code": true, "repo": true,
		"repository": true, "issue": true, "pull": true, "commit": true,
		"branch": true, "master": true, "develop": true, "bug": true,
		"hotfix": true, "docs": true, "build": true, "deploy": true,
		"tag": true, "note": true, "notes": true, "star": true, "stars": true,
		"fork": true, "forks": true, "watch": true, "watcher": true,
		"watchers": true, "contributor": true, "contributors": true,
		"author": true, "authors": true, "maintainer": true, "maintainers": true,
		"member": true, "members": true, "organization": true, "org": true,
		"company": true, "email": true, "address": true, "phone": true,
		"number": true, "data": true, "security": true, "api": true,
		"tool": true, "tools": true, "video": true, "image": true,
		"news": true, "search": true, "results": true, "more": true,
		"about": true, "contact": true, "read": true, "see": true,
		"share": true, "log": true, "update": true,
		"default": true, "profile": true, "settings": true, "turn": true,
		"mode": true, "restricted": true, "community": true, "guidelines": true,
		"play": true, "app": true, "web": true, "site": true, "test": true,
		"tests": true, "dev": true, "ci": true, "cd": true,
		"viewer": true, "smart": true, "content": true, "tap": true,
		"sunday": true, "upload": true, "starten": true,
		"ihren": true, "ihr": true, "sie": true, "mit": true,
		"the": true, "october": true, "oct": true, "angels": true,
		"your": true, "quick": true, "support": true,
	}
)

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	queries := buildQueries(query, qtype)

	var results []collectors.Result
	seenURL := make(map[string]bool)
	seenDerived := make(map[string]bool)

	for _, q := range queries {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		opts := websearch.Options{
			Count:     10,
			Type:      websearch.SearchTypeAuto,
			LiveCrawl: websearch.LiveCrawlFallback,
		}

		webResults, err := c.engine.Search(ctx, q, opts)
		if err != nil || len(webResults) == 0 {
			continue
		}

		queryResults := make([]collectors.Result, 0, len(webResults))
		for _, wr := range webResults {
			if seenURL[wr.URL] {
				continue
			}
			seenURL[wr.URL] = true

			confidence := confidenceForEngine(wr.Engine)
			if wr.Score > 10 {
				confidence += 0.05
			}
			if confidence > 0.95 {
				confidence = 0.95
			}

			result := collectors.Result{
				ID:          uuid.NewString(),
				Collector:   "searchengine",
				Type:        "web_result",
				Title:       wr.Title,
				Description: wr.Description,
				URL:         wr.URL,
				Data: map[string]any{
					"url":     wr.URL,
					"query":   q,
					"snippet": wr.Description,
					"engine":  wr.Engine,
					"rank":    wr.Rank,
					"score":   wr.Score,
				},
				Tags:       []string{"web", "search", wr.Engine, "link"},
				Confidence: confidence,
				Source:     sourceForEngine(wr.Engine),
				FoundAt:    time.Now(),
			}
			results = append(results, result)
			queryResults = append(queryResults, result)
		}

		if shouldExtractProfiles(qtype) {
			for _, profile := range extractProfilesFromResults(webResults, q) {
				key := "profile|" + profile.URL
				if seenDerived[key] {
					continue
				}
				seenDerived[key] = true
				results = append(results, profile)
			}
		}

		if shouldExtractNames(qtype) {
			for _, name := range extractNamesFromResults(queryResults, q) {
				key := "name|" + strings.ToLower(name.Title)
				if seenDerived[key] {
					continue
				}
				seenDerived[key] = true
				results = append(results, name)
			}
		}
	}

	return results, nil
}

func buildQueries(query, qtype string) []string {
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

func shouldExtractProfiles(qtype string) bool {
	switch qtype {
	case "email", "person_name", "username":
		return true
	default:
		return false
	}
}

func shouldExtractNames(qtype string) bool {
	switch qtype {
	case "email", "person_name":
		return true
	default:
		return false
	}
}

func confidenceForEngine(engine string) float64 {
	switch engine {
	case "google":
		return 0.85
	case "bing":
		return 0.75
	case "duckduckgo":
		return 0.7
	case "mojeek":
		return 0.6
	default:
		return 0.7
	}
}

func sourceForEngine(engine string) string {
	switch engine {
	case "google":
		return "google.com"
	case "bing":
		return "bing.com"
	case "duckduckgo":
		return "duckduckgo.com"
	case "mojeek":
		return "mojeek.com"
	default:
		return engine + ".com"
	}
}

func extractNamesFromResults(results []collectors.Result, query string) []collectors.Result {
	var sb strings.Builder
	for _, r := range results {
		if r.Type == "web_result" {
			sb.WriteString(r.Title)
			sb.WriteString(" ")
			sb.WriteString(r.Description)
			sb.WriteString(" ")
		}
	}
	return extractNames(sb.String(), query)
}

func isCapitalized(s string) bool {
	if len(s) == 0 {
		return false
	}
	return unicode.IsUpper(rune(s[0]))
}

func extractNames(content, query string) []collectors.Result {
	words := reWord.FindAllString(content, -1)

	freq := make(map[string]int)
	for i := 0; i < len(words)-1; i++ {
		if len(words[i]) >= 3 && len(words[i+1]) >= 3 &&
			isCapitalized(words[i]) && isCapitalized(words[i+1]) {
			name := words[i] + " " + words[i+1]
			freq[name]++
		}
	}

	var results []collectors.Result

	for name, count := range freq {
		lower := strings.ToLower(name)
		if strings.Contains(lower, strings.ToLower(query)) {
			continue
		}
		if count < 3 {
			continue
		}

		parts := strings.Fields(name)
		if len(parts) < 2 {
			continue
		}
		if commonNouns[strings.ToLower(parts[0])] || commonNouns[strings.ToLower(parts[1])] {
			continue
		}

		if len(results) >= 5 {
			break
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

func extractProfilesFromResults(webResults []websearch.Result, query string) []collectors.Result {
	var sb strings.Builder
	for _, r := range webResults {
		sb.WriteString(r.Title)
		sb.WriteString(" ")
		sb.WriteString(r.Description)
		sb.WriteString(" ")
		sb.WriteString(r.URL)
		sb.WriteString(" ")
	}
	content := sb.String()

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
				"profile_url":  profile,
				"platform":     platform,
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
