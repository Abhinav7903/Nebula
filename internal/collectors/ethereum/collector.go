package ethereum

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

func (c *Collector) Name() string            { return "ethereum" }
func (c *Collector) SupportedTypes() []string { return []string{"ethereum_address"} }
func (c *Collector) RequiresKey() bool        { return true }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=account&action=txlist&address=%s&sort=desc&page=1&offset=10&apikey=%s", query, c.key)
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
		Collector:   "ethereum",
		Type:        "ethereum_transactions",
		Title:       fmt.Sprintf("Ethereum transactions for %s", query),
		Description: "Recent Ethereum transactions",
		Data:        data,
		Tags:        []string{"ethereum", "blockchain", "crypto"},
		Confidence:  0.9,
		Source:      "etherscan.io",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
