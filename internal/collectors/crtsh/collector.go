package crtsh

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
}

func New() *Collector {
	return &Collector{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Collector) Name() string            { return "crtsh" }
func (c *Collector) SupportedTypes() []string { return []string{"domain", "subdomain", "url"} }
func (c *Collector) RequiresKey() bool        { return false }

type crtEntry struct {
	ID        int      `json:"id"`
	NameValue string   `json:"name_value"`
	Issuer    string   `json:"issuer_name"`
	NotAfter  string   `json:"not_after"`
	NotBefore string   `json:"not_before"`
}

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://crt.sh/?q=%s&output=json", query)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NebulaOSINT/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var entries []crtEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}

	var results []collectors.Result
	seen := make(map[string]bool)
	for _, e := range entries {
		if seen[e.NameValue] {
			continue
		}
		seen[e.NameValue] = true
		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "crtsh",
			Type:        "certificate",
			Title:       fmt.Sprintf("Certificate for %s", e.NameValue),
			Description: fmt.Sprintf("Issuer: %s, Valid until: %s", e.Issuer, e.NotAfter),
			Data: map[string]any{
				"common_name": e.NameValue,
				"issuer":      e.Issuer,
				"not_before":  e.NotBefore,
				"not_after":   e.NotAfter,
			},
			Tags:       []string{"certificate", "tls", "ssl"},
			Confidence: 0.95,
			Source:     "crt.sh",
			FoundAt:    time.Now(),
		})
	}
	return results, nil
}
