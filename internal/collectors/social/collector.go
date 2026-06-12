package social

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	var results []collectors.Result
	for _, site := range sites {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		profileURL := fmt.Sprintf(site.url, query)
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
					"platform": site.name,
					"username": query,
					"url":      profileURL,
					"body_size": len(body),
				},
				Tags:       []string{"social", site.name},
				Confidence: 0.8,
				Source:     site.name + ".com",
				FoundAt:    time.Now(),
			})
		}
	}
	return results, nil
}
