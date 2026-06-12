package github

import (
	"context"
	"encoding/json"
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
	apiURL := c.searchURL(query, qtype)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	total, _ := data["total_count"].(float64)
	if total == 0 {
		return nil, nil
	}

	resultType, tag := "github_search", "code_search"
	if qtype == "email" || qtype == "username" {
		resultType, tag = "github_user", "user_search"
	}

	items := data["items"]
	result := collectors.Result{
		ID:          uuid.NewString(),
		Collector:   "github",
		Type:        resultType,
		Title:       fmt.Sprintf("GitHub results for %s", query),
		Description: fmt.Sprintf("%d result(s) on GitHub", int(total)),
		Data: map[string]any{
			"total_count": total,
			"items":       items,
		},
		Tags:       []string{"github", tag},
		Confidence: 0.7,
		Source:     "github.com",
		FoundAt:    time.Now(),
	}
	return []collectors.Result{result}, nil
}

func (c *Collector) searchURL(query, qtype string) string {
	q := url.QueryEscape(query)
	switch qtype {
	case "email":
		return fmt.Sprintf("https://api.github.com/search/users?q=%s+in:email&per_page=10", q)
	case "username":
		return fmt.Sprintf("https://api.github.com/search/users?q=%s&per_page=10", q)
	default:
		return fmt.Sprintf("https://api.github.com/search/code?q=%s&per_page=10", q)
	}
}
