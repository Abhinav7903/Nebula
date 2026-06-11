package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/nebula/internal/collectors"
)

type Collector struct {
	client *http.Client
	token  string
}

func New(token string) *Collector {
	return &Collector{
		client: &http.Client{Timeout: 15 * time.Second},
		token:  token,
	}
}

func (c *Collector) Name() string            { return "github" }
func (c *Collector) SupportedTypes() []string { return []string{"domain", "email", "username", "company_name"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	q := url.QueryEscape(query)
	url := fmt.Sprintf("https://api.github.com/search/code?q=%s&per_page=10", q)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("User-Agent", "NebulaOSINT/1.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	result := collectors.Result{
		ID:          uuid.NewString(),
		Collector:   "github",
		Type:        "github_search",
		Title:       fmt.Sprintf("GitHub results for %s", query),
		Description: "GitHub code search results",
		Data:        data,
		Tags:        []string{"github", "code_search"},
		Confidence:  0.7,
		Source:      "github.com",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
