package pastebin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (c *Collector) Name() string            { return "pastebin" }
func (c *Collector) SupportedTypes() []string { return []string{"email", "username", "domain"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	q := url.QueryEscape(query)
	url := fmt.Sprintf("https://psbdmp.ws/api/v3/search?q=%s&size=10", q)
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

	dumps, _ := data["data"].([]any)
	var results []collectors.Result
	for _, d := range dumps {
		dm, _ := d.(map[string]any)
		id, _ := dm["id"].(string)
		title, _ := dm["title"].(string)
		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "pastebin",
			Type:        "pastebin_dump",
			Title:       fmt.Sprintf("Paste: %s", title),
			Description: fmt.Sprintf("Pastebin dump related to %s", query),
			URL:         fmt.Sprintf("https://psbdmp.ws/dump/%s", id),
			Data:        dm,
			Tags:        []string{"pastebin", "dump"},
			Confidence:  0.6,
			Source:      "psbdmp.ws",
			FoundAt:     time.Now(),
		})
	}
	return results, nil
}
