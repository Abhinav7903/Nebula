package wayback

import (
	"context"
	"encoding/json"
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
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Collector) Name() string            { return "wayback" }
func (c *Collector) SupportedTypes() []string { return []string{"domain", "url"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%s&output=json&limit=20", query)
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

	var entries [][]string
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}

	var results []collectors.Result
	for _, entry := range entries {
		if len(entry) < 6 {
			continue
		}
		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "wayback",
			Type:        "wayback_snapshot",
			Title:       fmt.Sprintf("Wayback snapshot: %s", entry[2]),
			Description: fmt.Sprintf("Archived at %s", entry[1]),
			URL:         fmt.Sprintf("https://web.archive.org/web/%s/%s", entry[1], entry[2]),
			Data: map[string]any{
				"url":       entry[2],
				"timestamp": entry[1],
				"status":    entry[4],
				"mime":      entry[3],
			},
			Tags:       []string{"wayback", "archive", "historical"},
			Confidence: 0.95,
			Source:     "web.archive.org",
			FoundAt:    time.Now(),
		})
	}
	return results, nil
}
