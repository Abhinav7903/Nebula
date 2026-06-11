package bitcoin

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

func (c *Collector) Name() string            { return "bitcoin" }
func (c *Collector) SupportedTypes() []string { return []string{"bitcoin_address"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://blockchain.info/rawaddr/%s?limit=10", query)
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
		Collector:   "bitcoin",
		Type:        "bitcoin_address_info",
		Title:       fmt.Sprintf("Bitcoin address info for %s", query),
		Description: "Bitcoin address transactions and balance",
		Data:        data,
		Tags:        []string{"bitcoin", "blockchain", "crypto"},
		Confidence:  0.9,
		Source:      "blockchain.info",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
