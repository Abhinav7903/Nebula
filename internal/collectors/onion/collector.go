package onion

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/nebula/internal/collectors"
)

type Collector struct {
	client *http.Client
}

func New(torClient *http.Client) *Collector {
	if torClient != nil {
		return &Collector{client: torClient}
	}
	return &Collector{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Collector) Name() string            { return "onion" }
func (c *Collector) SupportedTypes() []string { return []string{"onion"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("http://%s", query)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// truncate body for safety
	if len(body) > 4096 {
		body = body[:4096]
	}

	result := collectors.Result{
		ID:          uuid.NewString(),
		Collector:   "onion",
		Type:        "onion_service",
		Title:       fmt.Sprintf("Onion service: %s", query),
		Description: fmt.Sprintf("HTTP response from .onion (%d bytes, status %d)", len(body), resp.StatusCode),
		Data: map[string]any{
			"onion":       query,
			"status_code": resp.StatusCode,
			"headers":     resp.Header,
			"body_snippet": string(body),
		},
		Tags:       []string{"onion", "tor", "darkweb"},
		Confidence: 0.9,
		Source:     "tor_network",
		FoundAt:    time.Now(),
	}
	return []collectors.Result{result}, nil
}
