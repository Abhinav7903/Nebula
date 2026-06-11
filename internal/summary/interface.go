package summary

import (
	"context"

	"github.com/yourusername/nebula/internal/collectors"
)

type Summarizer interface {
	Summarize(ctx context.Context, query string, qtype string, results []collectors.Result) (string, error)
	ProviderName() string
}
