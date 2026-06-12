package whois

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

func (c *Collector) Name() string            { return "whois" }
func (c *Collector) SupportedTypes() []string { return []string{"domain", "subdomain", "ipv4", "ipv6"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://rdap.org/%s/%s", rdapPath(qtype), query)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
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
		Collector:   "whois",
		Type:        "whois",
		Title:       "RDAP lookup for " + query,
		Description: fmt.Sprintf("RDAP %s query completed", qtype),
		Data:        data,
		Tags:        []string{"whois", "rdap"},
		Confidence:  0.9,
		Source:      "rdap.org",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}

func rdapPath(qtype string) string {
	switch qtype {
	case "ipv4", "ipv6":
		return "ip"
	case "domain", "subdomain":
		return "domain"
	default:
		return "domain"
	}
}
