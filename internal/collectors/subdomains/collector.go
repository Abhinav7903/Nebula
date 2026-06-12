package subdomains

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

func (c *Collector) Name() string            { return "subdomains" }
func (c *Collector) SupportedTypes() []string { return []string{"domain"} }
func (c *Collector) RequiresKey() bool        { return false }

type alienVaultResp struct {
	Subdomains []string `json:"subdomains"`
}

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/passive_dns", query)
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

	var data alienVaultResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var results []collectors.Result
	for _, sub := range data.Subdomains {
		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "subdomains",
			Type:        "subdomain",
			Title:       sub,
			Description: fmt.Sprintf("Subdomain of %s", query),
			Data: map[string]any{
				"subdomain": sub,
				"domain":    query,
			},
			Tags:       []string{"subdomain", "passive_dns"},
			Confidence: 0.8,
			Source:     "otx.alienvault.com",
			FoundAt:    time.Now(),
		})
	}
	return results, nil
}
