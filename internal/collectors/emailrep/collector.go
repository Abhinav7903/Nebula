package emailrep

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

func (c *Collector) Name() string            { return "emailrep" }
func (c *Collector) SupportedTypes() []string { return []string{"email"} }
func (c *Collector) RequiresKey() bool        { return true }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://emailrep.io/%s", query)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if c.key != "" {
		req.Header.Set("Key", c.key)
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
		Collector:   "emailrep",
		Type:        "email_reputation",
		Title:       fmt.Sprintf("EmailRep for %s", query),
		Description: "Email reputation data",
		Data:        data,
		Tags:        []string{"email", "reputation"},
		Confidence:  0.8,
		Source:      "emailrep.io",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
