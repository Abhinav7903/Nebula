package dnsdumpster

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/nebula/internal/collectors"
)

type Collector struct {
	client *http.Client
}

func New() *Collector {
	return &Collector{
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Collector) Name() string            { return "dnsdumpster" }
func (c *Collector) SupportedTypes() []string { return []string{"domain", "subdomain"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	url := fmt.Sprintf("https://dnsdumpster.com/domain/%s", query)
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

	content := string(body)
	var results []collectors.Result

	subRe := regexp.MustCompile(`([a-zA-Z0-9][a-zA-Z0-9\-\.]+\.` + regexp.QuoteMeta(query) + `)`)
	matches := subRe.FindAllString(content, -1)
	seen := make(map[string]bool)
	for _, m := range matches {
		m = strings.TrimSpace(m)
		if seen[m] || len(m) > 255 {
			continue
		}
		seen[m] = true
		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "dnsdumpster",
			Type:        "subdomain",
			Title:       m,
			Description: fmt.Sprintf("Subdomain found via DNSDumpster for %s", query),
			Data: map[string]any{
				"subdomain": m,
				"domain":    query,
			},
			Tags:       []string{"subdomain", "dnsdumpster"},
			Confidence: 0.6,
			Source:     "dnsdumpster.com",
			FoundAt:    time.Now(),
		})
	}
	return results, nil
}
