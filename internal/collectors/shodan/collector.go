package shodan

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
	key    string
}

func New(key string) *Collector {
	return &Collector{
		client: &http.Client{Timeout: 15 * time.Second},
		key:    key,
	}
}

func (c *Collector) Name() string            { return "shodan" }
func (c *Collector) SupportedTypes() []string { return []string{"ipv4", "ipv6", "cidr", "domain"} }
func (c *Collector) RequiresKey() bool        { return true }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	var url string
	switch qtype {
	case "ipv4", "ipv6":
		url = fmt.Sprintf("https://api.shodan.io/shodan/host/%s?key=%s", query, c.key)
	case "domain":
		url = fmt.Sprintf("https://api.shodan.io/dns/resolve?hostnames=%s&key=%s", query, c.key)
	default:
		return nil, fmt.Errorf("unsupported type: %s", qtype)
	}

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

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	result := collectors.Result{
		ID:          uuid.NewString(),
		Collector:   "shodan",
		Type:        "shodan_host",
		Title:       fmt.Sprintf("Shodan data for %s", query),
		Description: "Shodan intelligence data",
		Data:        data,
		Tags:        []string{"shodan", "osint"},
		Confidence:  0.9,
		Source:      "shodan.io",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
