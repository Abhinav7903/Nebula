package websearch

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"
)

type SearchType string

const (
	SearchTypeAuto SearchType = "auto"
	SearchTypeFast SearchType = "fast"
	SearchTypeDeep SearchType = "deep"
)

type LiveCrawlMode string

const (
	LiveCrawlFallback  LiveCrawlMode = "fallback"
	LiveCrawlPreferred LiveCrawlMode = "preferred"
)

type Result struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	Engine      string  `json:"engine"`
	Content     string  `json:"content,omitempty"`
	Rank        int     `json:"rank"`
	Score       float64 `json:"score,omitempty"`
}

type Options struct {
	Count     int
	Type      SearchType
	LiveCrawl LiveCrawlMode
}

type Provider interface {
	Name() string
	Search(ctx context.Context, query string, count int) ([]Result, error)
	IsAvailable() bool
}

type Engine struct {
	providers []Provider
	client    *http.Client
	mu        sync.RWMutex
}

func New(client *http.Client) *Engine {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Engine{
		client: client,
	}
}

func (e *Engine) AddProvider(p Provider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.providers = append(e.providers, p)
}

func (e *Engine) Providers() []Provider {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Provider, len(e.providers))
	copy(out, e.providers)
	return out
}

func (e *Engine) Search(ctx context.Context, query string, opts Options) ([]Result, error) {
	e.mu.RLock()
	providers := make([]Provider, len(e.providers))
	copy(providers, e.providers)
	e.mu.RUnlock()

	if len(providers) == 0 {
		return nil, fmt.Errorf("no search providers registered")
	}

	if opts.Count <= 0 {
		opts.Count = 10
	}
	if opts.Type == "" {
		opts.Type = SearchTypeAuto
	}
	if opts.LiveCrawl == "" {
		opts.LiveCrawl = LiveCrawlFallback
	}

	timeout := 10 * time.Second
	if opts.Type == SearchTypeDeep {
		timeout = 30 * time.Second
	} else if opts.Type == SearchTypeFast {
		timeout = 5 * time.Second
	}

	type providerResult struct {
		name    string
		order   int
		results []Result
		err     error
	}

	var wg sync.WaitGroup
	ch := make(chan providerResult, len(providers))

	for i, p := range providers {
		if !p.IsAvailable() {
			continue
		}
		wg.Add(1)
		go func(order int, prov Provider) {
			defer wg.Done()
			searchCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			results, err := prov.Search(searchCtx, query, opts.Count)
			ch <- providerResult{name: prov.Name(), order: order, results: results, err: err}
		}(i, p)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var allResults []Result
	seen := make(map[string]int)
	engineResults := make(map[string]int)

	for pr := range ch {
		if pr.err != nil {
			continue
		}
		normalized := normalizeProviderResults(pr.results, pr.name, query, opts.Count, pr.order)
		engineResults[pr.name] = len(normalized)
		for _, r := range normalized {
			key := canonicalResultKey(r.URL)
			if key == "" {
				continue
			}
			if idx, ok := seen[key]; ok {
				if r.Score > allResults[idx].Score {
					if allResults[idx].Description != "" && r.Description == "" {
						r.Description = allResults[idx].Description
					}
					if allResults[idx].Content != "" && r.Content == "" {
						r.Content = allResults[idx].Content
					}
					allResults[idx] = r
				}
				allResults[idx].Score += 1.25
				continue
			}
			seen[key] = len(allResults)
			allResults = append(allResults, r)
		}
	}

	if len(allResults) == 0 {
		return nil, fmt.Errorf("all search providers returned no results")
	}

	sort.Slice(allResults, func(i, j int) bool {
		if allResults[i].Score != allResults[j].Score {
			return allResults[i].Score > allResults[j].Score
		}
		return allResults[i].Rank < allResults[j].Rank
	})

	if opts.LiveCrawl == LiveCrawlPreferred && len(allResults) > 0 {
		fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		fetchPageContent(fetchCtx, allResults, e)
	}

	if opts.Type == SearchTypeDeep && len(allResults) > 0 {
		deepCtx, cancel := context.WithTimeout(ctx, time.Duration(len(providers))*10*time.Second)
		defer cancel()
		fetchPageContent(deepCtx, allResults, e)
	}

	if len(allResults) > opts.Count {
		allResults = allResults[:opts.Count]
	}

	for i := range allResults {
		allResults[i].Rank = i + 1
	}

	return allResults, nil
}

func fetchPageContent(ctx context.Context, results []Result, e *Engine) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for i := range results {
		if results[i].URL == "" {
			continue
		}
		wg.Add(1)
		go func(r *Result) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			page, err := e.FetchURL(ctx, r.URL, DefaultFetchOptions())
			if err != nil || page == nil {
				return
			}
			r.Content = page.Content
			if r.Description == "" {
				r.Description = truncate(page.Content, 300)
			}
		}(&results[i])
	}
	wg.Wait()
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func (e *Engine) deepFetch(ctx context.Context, results []Result, maxExtra int) []Result {
	if maxExtra <= 0 {
		return nil
	}

	type fetchJob struct {
		url   string
		title string
	}

	jobs := make([]fetchJob, 0, len(results))
	for _, r := range results {
		if r.URL != "" {
			jobs = append(jobs, fetchJob{url: r.URL, title: r.Title})
		}
	}

	if len(jobs) > maxExtra*2 {
		rand.Shuffle(len(jobs), func(i, j int) {
			jobs[i], jobs[j] = jobs[j], jobs[i]
		})
		jobs = jobs[:maxExtra*2]
	}

	var extra []Result
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for _, fj := range jobs {
		wg.Add(1)
		go func(f fetchJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			page, err := e.FetchURL(ctx, f.url, DefaultFetchOptions())
			if err != nil || page == nil {
				return
			}

			mu.Lock()
			extra = append(extra, Result{
				Title:       f.title,
				URL:         f.url,
				Description: page.Content,
				Content:     page.Content,
				Engine:      "deep_fetch",
			})
			mu.Unlock()
		}(fj)
	}
	wg.Wait()

	if len(extra) > maxExtra {
		extra = extra[:maxExtra]
	}
	return extra
}
