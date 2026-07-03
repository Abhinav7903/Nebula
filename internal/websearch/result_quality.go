package websearch

import (
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var queryTokenRe = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9._-]*`)

var noisyQueryParams = map[string]bool{
	"fbclid":  true,
	"gclid":   true,
	"gbraid":  true,
	"igshid":  true,
	"mc_cid":  true,
	"mc_eid":  true,
	"msclkid": true,
	"oq":      true,
	"sa":      true,
	"source":  true,
	"sxsrf":   true,
	"usg":     true,
	"ved":     true,
	"wbraid":  true,
}

var searchHosts = []string{
	"google.",
	"bing.com",
	"duckduckgo.com",
	"mojeek.com",
}

func normalizeProviderResults(results []Result, provider, query string, count, providerOrder int) []Result {
	if count <= 0 {
		count = 10
	}

	out := make([]Result, 0, len(results))
	for i, r := range results {
		r.Title = normalizeWhitespace(r.Title)
		r.Description = normalizeWhitespace(r.Description)
		if r.Engine == "" {
			r.Engine = provider
		}
		if r.Rank <= 0 {
			r.Rank = i + 1
		}
		cleanURL, ok := normalizeResultURL(r.URL)
		if !ok {
			continue
		}
		r.URL = cleanURL
		if r.Title == "" || isSearchNoiseURL(r.URL) {
			continue
		}
		if violatesSiteFilter(query, r.URL) {
			continue
		}
		r.Score = scoreResult(query, r, providerOrder, count)
		out = append(out, r)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Rank < out[j].Rank
	})
	if len(out) > count {
		out = out[:count]
	}
	return out
}

func scoreResult(query string, r Result, providerOrder, count int) float64 {
	if count <= 0 {
		count = 10
	}
	if r.Rank <= 0 {
		r.Rank = count
	}

	score := engineWeight(r.Engine)
	score += math.Max(0, 3.0-(float64(r.Rank-1)*0.22))
	score -= float64(providerOrder) * 0.05

	phrase := strings.ToLower(strings.TrimSpace(strings.Trim(query, `"`)))
	title := strings.ToLower(r.Title)
	desc := strings.ToLower(r.Description)
	rawURL := strings.ToLower(r.URL)

	if phrase != "" {
		if strings.Contains(title, phrase) {
			score += 3.0
		}
		if strings.Contains(desc, phrase) {
			score += 1.5
		}
		if strings.Contains(rawURL, url.QueryEscape(phrase)) || strings.Contains(rawURL, strings.ReplaceAll(phrase, " ", "-")) {
			score += 0.75
		}
	}

	for _, token := range queryTokens(query) {
		if strings.Contains(title, token) {
			score += 0.9
		}
		if strings.Contains(desc, token) {
			score += 0.45
		}
		if strings.Contains(rawURL, token) {
			score += 0.35
		}
	}

	if r.Description == "" {
		score -= 0.8
	}
	if len(r.Title) < 8 {
		score -= 0.4
	}
	for _, domain := range siteFilterDomains(query) {
		if hostMatchesDomain(r.URL, domain) {
			score += 2.0
		}
	}

	return score
}

func engineWeight(engine string) float64 {
	switch strings.ToLower(engine) {
	case "google":
		return 5.0
	case "duckduckgo":
		return 4.2
	case "bing":
		return 3.9
	case "mojeek":
		return 3.0
	default:
		return 2.5
	}
}

func queryTokens(query string) []string {
	matches := queryTokenRe.FindAllString(strings.ToLower(query), -1)
	out := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, token := range matches {
		if strings.HasPrefix(token, "site") || len(token) < 2 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

func siteFilterDomains(query string) []string {
	fields := strings.Fields(query)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.Trim(strings.ToLower(f), `"'()`)
		if !strings.HasPrefix(f, "site:") {
			continue
		}
		domain := strings.TrimPrefix(f, "site:")
		domain = strings.TrimPrefix(domain, "www.")
		if domain != "" {
			out = append(out, domain)
		}
	}
	return out
}

func violatesSiteFilter(query, rawURL string) bool {
	domains := siteFilterDomains(query)
	if len(domains) == 0 {
		return false
	}
	for _, domain := range domains {
		if hostMatchesDomain(rawURL, domain) {
			return false
		}
	}
	return true
}

func hostMatchesDomain(rawURL, domain string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	domain = strings.TrimPrefix(strings.ToLower(domain), "www.")
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func normalizeResultURL(raw string) (string, bool) {
	raw = strings.TrimSpace(cleanSearchRedirect(raw))
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.RawQuery = cleanQuery(u.Query())
	if len(u.Path) > 1 {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String(), true
}

func canonicalResultKey(raw string) string {
	u, ok := normalizeResultURL(raw)
	if !ok {
		return ""
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return strings.ToLower(u)
	}
	parsed.Host = strings.TrimPrefix(parsed.Host, "www.")
	return strings.ToLower(parsed.String())
}

func cleanSearchRedirect(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.HasPrefix(raw, "/url?") || strings.HasPrefix(raw, "https://www.google.com/url?") || strings.HasPrefix(raw, "http://www.google.com/url?") {
		if u, err := url.Parse(raw); err == nil {
			if target := u.Query().Get("q"); target != "" {
				return target
			}
		}
	}

	if strings.HasPrefix(raw, "/interstitial?") || strings.Contains(raw, "google.com/interstitial?") {
		if u, err := url.Parse(raw); err == nil {
			if target := u.Query().Get("url"); target != "" {
				return target
			}
		}
	}

	if strings.Contains(raw, "duckduckgo.com/l/?") || strings.HasPrefix(raw, "/l/?") {
		if u, err := url.Parse(raw); err == nil {
			if target := u.Query().Get("uddg"); target != "" {
				return target
			}
		}
	}

	return raw
}

func cleanQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	for key := range values {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || noisyQueryParams[lower] {
			values.Del(key)
		}
	}
	return values.Encode()
}

func isSearchNoiseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return true
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	path := strings.ToLower(u.Path)
	for _, marker := range searchHosts {
		if strings.Contains(host, marker) {
			return path == "" || path == "/" || path == "/search" || path == "/url" || path == "/l/"
		}
	}
	return false
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
