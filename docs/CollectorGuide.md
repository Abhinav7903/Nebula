# Collector Guide

## Step-by-Step: Adding a New Collector

### 1. Create the package

```bash
mkdir internal/collectors/myservice
```

### 2. Implement the Collector interface

Create `internal/collectors/myservice/collector.go`:

```go
package myservice

import (
    "context"
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

func (c *Collector) Name() string            { return "myservice" }
func (c *Collector) SupportedTypes() []string { return []string{"domain", "ipv4"} }
func (c *Collector) RequiresKey() bool        { return true }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
    // Make API call, parse response, return results
}
```

### 3. Register the collector

Add one line to `cmd/nebula/main.go` in `registerCollectors()`:

```go
whenEnabled(cfg.Collectors.MyService, myservice.New(cfg.Collectors.MyService.Key))
```

### 4. Add config

Add to `configs/config.yaml`:

```yaml
  myservice: { enabled: false, key: "${MYSERVICE_KEY}" }
```

### 5. Add config struct

Add to `internal/config/config.go`:

```go
type CollectorsConfig struct {
    // ...
    MyService CollectorItem `yaml:"myservice"`
}
```

### 6. Add env var

Add to `.env.example`:

```
MYSERVICE_KEY=
```

That's it. No other files need to change.

## Collector Interface

```go
type Collector interface {
    Name() string                    // Unique collector name
    SupportedTypes() []string        // Query types this collector handles
    RequiresKey() bool               // True if API key is required
    Execute(ctx, query, qtype) ([]Result, error)
}
```

## Result Structure

```go
type Result struct {
    ID          string            `json:"id"`
    Collector   string            `json:"collector"`
    Type        string            `json:"type"`
    Title       string            `json:"title"`
    Description string            `json:"description"`
    URL         string            `json:"url,omitempty"`
    Data        map[string]any    `json:"data"`
    Tags        []string          `json:"tags"`
    Confidence  float64           `json:"confidence"`
    Source      string            `json:"source"`
    FoundAt     time.Time         `json:"found_at"`
}
```

## Best Practices

1. Always respect context cancellation
2. Set appropriate HTTP timeouts
3. Handle rate limits gracefully
4. Log errors, don't panic
5. Use UUID for result IDs
6. Tag results appropriately
7. Set meaningful confidence scores
8. Include source attribution
