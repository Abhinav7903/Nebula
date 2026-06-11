package censys

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
		client: &http.Client{Timeout: 15 * time.Second},
		key:    key,
	}
}

func (c *Collector) Name() string            { return "censys" }
func (c *Collector) SupportedTypes() []string { return []string{"ipv4", "ipv6", "domain"} }
func (c *Collector) RequiresKey() bool        { return true }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	var url string
	switch qtype {
	case "ipv4", "ipv6":
		url = fmt.Sprintf("https://search.censys.io/api/v2/hosts/%s", query)
	case "domain":
		url = fmt.Sprintf("https://search.censys.io/api/v2/hosts/search?q=%s", query)
	default:
		return nil, fmt.Errorf("unsupported type: %s", qtype)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
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
		Collector:   "censys",
		Type:        "censys_host",
		Title:       fmt.Sprintf("Censys data for %s", query),
		Description: "Censys intelligence data",
		Data:        data,
		Tags:        []string{"censys", "osint"},
		Confidence:  0.9,
		Source:      "censys.io",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
