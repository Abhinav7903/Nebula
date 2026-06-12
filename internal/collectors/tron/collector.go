package tron

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

func (c *Collector) Name() string            { return "tron" }
func (c *Collector) SupportedTypes() []string { return []string{"tron_address"} }
func (c *Collector) RequiresKey() bool        { return true }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://apilist.tronscan.org/api/account?address=%s", query)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NebulaOSINT/1.0")
	req.Header.Set("TRON-PRO-API-KEY", c.key)

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
		Collector:   "tron",
		Type:        "tron_account",
		Title:       fmt.Sprintf("Tron account: %s", query),
		Description: "Tron blockchain account data",
		Data:        data,
		Tags:        []string{"tron", "blockchain", "crypto"},
		Confidence:  0.9,
		Source:      "tronscan.org",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
