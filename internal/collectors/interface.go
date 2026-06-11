package collectors

import (
	"context"
	"time"
)

type Result struct {
	ID          string         `json:"id"`
	Collector   string         `json:"collector"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	URL         string         `json:"url,omitempty"`
	Data        map[string]any `json:"data"`
	Tags        []string       `json:"tags"`
	Confidence  float64        `json:"confidence"`
	Source      string         `json:"source"`
	FoundAt     time.Time      `json:"found_at"`
}

type Collector interface {
	Name() string
	SupportedTypes() []string
	RequiresKey() bool
	Execute(ctx context.Context, query string, qtype string) ([]Result, error)
}
