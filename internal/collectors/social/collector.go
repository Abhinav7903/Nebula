package social

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/Abhinav7903/nebula/internal/collectors"
)

type Collector struct {
	client *http.Client
}

func New() *Collector {
	return &Collector{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Collector) Name() string            { return "social" }
func (c *Collector) SupportedTypes() []string { return []string{"username", "person_name"} }
func (c *Collector) RequiresKey() bool        { return false }

type socialSite struct {
	name string
	url  string
}

var sites = []socialSite{
	{"github", "https://github.com/%s"},
	{"twitter", "https://twitter.com/%s"},
	{"linkedin", "https://linkedin.com/in/%s"},
	{"reddit", "https://reddit.com/user/%s"},
	{"keybase", "https://keybase.io/%s"},
	{"hackernews", "https://news.ycombinator.com/user?id=%s"},
	{"medium", "https://medium.com/@%s"},
	{"devto", "https://dev.to/%s"},
}

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	usernames := usernamesForQuery(query, qtype)

	var results []collectors.Result
	seen := make(map[string]bool)

	for _, username := range usernames {
		for _, site := range sites {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			default:
			}

			key := site.name + ":" + username
			if seen[key] {
				continue
			}
			seen[key] = true

			profileURL := fmt.Sprintf(site.url, url.PathEscape(username))
			req, err := http.NewRequestWithContext(ctx, "GET", profileURL, nil)
			if err != nil {
				continue
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

			resp, err := c.client.Do(req)
			if err != nil {
				continue
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			if resp.StatusCode == 200 {
				results = append(results, collectors.Result{
					ID:          uuid.NewString(),
					Collector:   "social",
					Type:        "social_profile",
					Title:       fmt.Sprintf("%s profile for %s", site.name, query),
					Description: fmt.Sprintf("Found on %s (status 200)", site.name),
					URL:         profileURL,
					Data: map[string]any{
						"platform":  site.name,
						"username":  username,
						"url":       profileURL,
						"body_size": len(body),
					},
					Tags:       []string{"social", site.name},
					Confidence: 0.8,
					Source:     site.name + ".com",
					FoundAt:    time.Now(),
				})
			}
		}
	}
	return results, nil
}

func usernamesForQuery(query, qtype string) []string {
	seen := make(map[string]bool)
	var usernames []string

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			usernames = append(usernames, s)
		}
	}

	if qtype == "person_name" {
		parts := strings.Fields(query)
		add(query)
		for _, p := range parts {
			add(p)
		}
		if len(parts) >= 2 {
			add(parts[0] + parts[1])
			add(parts[0] + "." + parts[1])
			add(parts[0] + "_" + parts[1])
			add(parts[0] + "-" + parts[1])
		}
	} else {
		add(query)
	}

	return usernames
}
