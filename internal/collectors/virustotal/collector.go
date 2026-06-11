package virustotal

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

func (c *Collector) Name() string            { return "virustotal" }
func (c *Collector) SupportedTypes() []string { return []string{"domain", "ipv4", "url", "hash_md5", "hash_sha1", "hash_sha256"} }
func (c *Collector) RequiresKey() bool        { return true }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	var endpoint string
	switch qtype {
	case "domain", "ipv4":
		endpoint = fmt.Sprintf("https://www.virustotal.com/api/v3/domains/%s", query)
	case "url":
		endpoint = fmt.Sprintf("https://www.virustotal.com/api/v3/urls/%s", query)
	case "hash_md5", "hash_sha1", "hash_sha256":
		endpoint = fmt.Sprintf("https://www.virustotal.com/api/v3/files/%s", query)
	default:
		return nil, fmt.Errorf("unsupported type: %s", qtype)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apikey", c.key)
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
		Collector:   "virustotal",
		Type:        "virustotal_report",
		Title:       fmt.Sprintf("VirusTotal report for %s", query),
		Description: "Threat intelligence data from VirusTotal",
		Data:        data,
		Tags:        []string{"virustotal", "threat_intel"},
		Confidence:  0.9,
		Source:      "virustotal.com",
		FoundAt:     time.Now(),
	}
	return []collectors.Result{result}, nil
}
