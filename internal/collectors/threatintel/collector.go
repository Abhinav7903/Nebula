package threatintel

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

func (c *Collector) Name() string            { return "threatintel" }
func (c *Collector) SupportedTypes() []string { return []string{"ipv4", "domain", "hash_md5", "hash_sha1", "hash_sha256"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	var endpoint string
	switch qtype {
	case "ipv4":
		endpoint = fmt.Sprintf("https://www.abuseipdb.com/api/v2/check?ipAddress=%s&maxAgeInDays=90", query)
	case "domain":
		endpoint = fmt.Sprintf("https://www.abuseipdb.com/api/v2/check?domain=%s&maxAgeInDays=90", query)
	default:
		endpoint = fmt.Sprintf("https://www.abuseipdb.com/api/v2/check?ipAddress=%s&maxAgeInDays=90", query)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NebulaOSINT/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Key", "public")

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
		_ = err
	}

	// Try AlienVault OTX as secondary source
	otxResults := c.checkOTX(ctx, query, qtype)

	results := otxResults
	result := collectors.Result{
		ID:          uuid.NewString(),
		Collector:   "threatintel",
		Type:        "threat_intel",
		Title:       fmt.Sprintf("Threat intel for %s", query),
		Description: "AbuseIPDB threat intelligence data",
		Data:        data,
		Tags:        []string{"threat_intel", "abuseipdb"},
		Confidence:  0.7,
		Source:      "abuseipdb.com",
		FoundAt:     time.Now(),
	}

	return append([]collectors.Result{result}, results...), nil
}

type otxIndicator struct {
	Type        string `json:"type"`
	Indicator   string `json:"indicator"`
	Description string `json:"description"`
}

func (c *Collector) checkOTX(ctx context.Context, query string, qtype string) []collectors.Result {
	url := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/%s/%s/general", otxType(qtype), query)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "NebulaOSINT/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}

	return []collectors.Result{
		{
			ID:          uuid.NewString(),
			Collector:   "threatintel",
			Type:        "otx_intel",
			Title:       fmt.Sprintf("OTX intelligence for %s", query),
			Description: "AlienVault OTX threat data",
			Data:        data,
			Tags:        []string{"threat_intel", "otx", "alienvault"},
			Confidence:  0.7,
			Source:      "otx.alienvault.com",
			FoundAt:     time.Now(),
		},
	}
}

func otxType(qtype string) string {
	switch qtype {
	case "ipv4", "ipv6":
		return "IPv4"
	case "domain", "subdomain":
		return "domain"
	case "hash_md5", "hash_sha1", "hash_sha256":
		return "file"
	default:
		return "domain"
	}
}
