package summary

import (
	"context"

	"github.com/Abhinav7903/nebula/internal/collectors"
)

type Summarizer interface {
	Summarize(ctx context.Context, query string, qtype string, results []collectors.Result) (string, error)
	ProviderName() string
}
