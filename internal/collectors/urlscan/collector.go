package urlscan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/nebula/internal/collectors"
)

type Collector struct {
	client *http.Client
	key    string
}

func New(key string) *Collector {
	return &Collector{
		client: &http.Client{Timeout: 30 * time.Second},
		key:    key,
	}
}

func (c *Collector) Name() string            { return "urlscan" }
func (c *Collector) SupportedTypes() []string { return []string{"domain", "url", "ipv4"} }
func (c *Collector) RequiresKey() bool        { return true }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://urlscan.io/api/v1/search/?q=%s&size=10", query)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("API-Key", c.key)
	req.Header.Set("User-Agent", "NebulaOSINT/1.0")
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

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	result := collectors.Result{
		ID:          uuid.NewString(),
		Collector:   "urlscan",
		Type:        "urlscan_search",
		Title:       fmt.Sprintf("URLScan results for %s", query),
		Description: "URL/domain scan results",
		Data:        data,
		Tags:        []string{"urlscan", "sandbox"},
		Confidence:  0.8,
		Source:      "urlscan.io",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
