package summary

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/nebula/internal/collectors"
	"github.com/yourusername/nebula/internal/metrics"
)

type GroqSummarizer struct {
	client  *http.Client
	apiKey  string
	model   string
	endpoint string
}

func NewGroq(apiKey, model string) *GroqSummarizer {
	return &GroqSummarizer{
		client:   &http.Client{Timeout: 30 * time.Second},
		apiKey:   apiKey,
		model:    model,
		endpoint: "https://api.groq.com/openai/v1/chat/completions",
	}
}

func (g *GroqSummarizer) ProviderName() string { return "groq" }

func (g *GroqSummarizer) Summarize(ctx context.Context, query string, qtype string, results []collectors.Result) (string, error) {
	start := time.Now()
	defer func() {
		metrics.AISummaryDuration.Observe(time.Since(start).Seconds())
	}()

	if g.apiKey == "" {
		return summaryFallback(results), nil
	}

	resultsJSON, _ := json.MarshalIndent(results, "", "  ")

	systemPrompt := fmt.Sprintf(
		"You are an OSINT analyst. Given the following intelligence results about %s '%s', "+
			"produce a concise 3-5 paragraph intelligence summary. Highlight key findings, "+
			"anomalies, and risk indicators. Be factual, terse, and structured.",
		qtype, query,
	)

	payload := map[string]any{
		"model": g.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": fmt.Sprintf("Query: %s (%s)\n\nResults:\n%s", query, qtype, string(resultsJSON))},
		},
		"max_tokens": 1024,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", g.endpoint, bytes.NewReader(body))
	if err != nil {
		return summaryFallback(results), err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return summaryFallback(results), err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return summaryFallback(results), err
	}

	var groqResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		return summaryFallback(results), err
	}

	if len(groqResp.Choices) == 0 || groqResp.Choices[0].Message.Content == "" {
		return summaryFallback(results), fmt.Errorf("empty groq response")
	}

	return groqResp.Choices[0].Message.Content, nil
}

func summaryFallback(results []collectors.Result) string {
	if len(results) == 0 {
		return "No results found."
	}

	byCollector := make(map[string]int)
	for _, r := range results {
		byCollector[r.Collector]++
	}

	summary := fmt.Sprintf("Found %d results across %d collectors:\n\n", len(results), len(byCollector))
	for name, count := range byCollector {
		summary += fmt.Sprintf("- %s: %d results\n", name, count)
	}
	return summary
}
