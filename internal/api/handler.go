package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/Abhinav7903/nebula/internal/collectors"
	"github.com/Abhinav7903/nebula/internal/deduplication"
	"github.com/Abhinav7903/nebula/internal/detection"
	"github.com/Abhinav7903/nebula/internal/metrics"
	"github.com/Abhinav7903/nebula/internal/progress"
	"github.com/Abhinav7903/nebula/internal/ranking"
	"github.com/Abhinav7903/nebula/internal/store"
	"github.com/Abhinav7903/nebula/internal/summary"
	"github.com/Abhinav7903/nebula/internal/websearch"
	"github.com/Abhinav7903/nebula/internal/workers"
)

type Handler struct {
	logger      *slog.Logger
	store       *store.MemoryStore
	registry    *collectors.Registry
	pool        *workers.Pool
	hub         *progress.Hub
	sse         *SSEWriter
	summarizer  summary.Summarizer
	websearch   *websearch.Engine
}

func NewHandler(logger *slog.Logger, s *store.MemoryStore, reg *collectors.Registry,
	pool *workers.Pool, hub *progress.Hub, summ summary.Summarizer, ws *websearch.Engine) *Handler {
	return &Handler{
		logger:     logger,
		store:      s,
		registry:   reg,
		pool:       pool,
		hub:        hub,
		sse:        NewSSEWriter(hub),
		summarizer: summ,
		websearch:  ws,
	}
}

type searchRequest struct {
	Query string `json:"query"`
}

func (h *Handler) CreateSearch(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		writeProblem(w, http.StatusBadRequest, "empty_query", "Query is required")
		return
	}
	if len(req.Query) > 512 {
		writeProblem(w, http.StatusBadRequest, "query_too_long", "Max query length is 512 characters")
		return
	}

	qtype := detection.Detect(req.Query)
	metrics.SearchesTotal.Inc()

	searchID := uuid.NewString()
	now := time.Now()

	collectorNames := h.registry.NamesByType(string(qtype))
	allNames := h.registry.NamesByType("all")
	seen := make(map[string]bool, len(collectorNames)+len(allNames))
	for _, n := range collectorNames {
		seen[n] = true
	}
	for _, n := range allNames {
		if !seen[n] {
			collectorNames = append(collectorNames, n)
			seen[n] = true
		}
	}

	search := &store.Search{
		ID:        searchID,
		Query:     req.Query,
		QueryType: qtype,
		Status:    store.StatusRunning,
		StartedAt: now,
		Collectors: collectorNames,
	}
	h.store.Create(search)
	metrics.SearchesActive.Inc()

	h.hub.Send(searchID, progress.Event{
		Event: "search_started",
		Payload: map[string]any{
			"search_id":         searchID,
			"query":             req.Query,
			"query_type":        qtype,
			"collectors_planned": len(collectorNames),
		},
	})

	go func() {
		h.pool.EnqueueCollectors(searchID, req.Query, string(qtype), collectorNames)
	}()

	go h.waitForCompletion(searchID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"search_id":  searchID,
		"query_type": qtype,
	})
}

func (h *Handler) waitForCompletion(searchID string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(120 * time.Second)

	for {
		select {
		case <-timeout:
			h.finalizeSearch(searchID)
			return
		case <-ticker.C:
			s, ok := h.store.Get(searchID)
			if !ok {
				return
			}
			done := len(s.CollectorsCompleted)
			if done >= len(s.Collectors) {
				h.finalizeSearch(searchID)
				return
			}
		}
	}
}

func (h *Handler) finalizeSearch(searchID string) {
	s, ok := h.store.Get(searchID)
	if !ok {
		return
	}

	s.Results = deduplication.New().Filter(s.Results)
	ranking.Rank(s.Results)

	h.hub.Send(searchID, progress.Event{
		Event: "summary_started",
		Payload: map[string]any{"search_id": searchID},
	})

	summaryText, _ := h.summarizer.Summarize(context.Background(), s.Query, string(s.QueryType), s.Results)

	finishedAt := time.Now()
	duration := finishedAt.Sub(s.StartedAt).Milliseconds()

	collectorCounts := make(map[string]int)
	for _, r := range s.Results {
		collectorCounts[r.Collector]++
	}

	h.store.Update(searchID, func(sr *store.Search) {
		sr.Status = store.StatusDone
		sr.FinishedAt = &finishedAt
		sr.Summary = summaryText
		sr.Stats = store.Stats{
			CollectorsRun:   len(s.Collectors),
			CollectorsOK:    len(collectorCounts),
			CollectorsFailed: len(s.Collectors) - len(collectorCounts),
			ResultsFound:    len(s.Results),
			DurationMs:      duration,
		}
	})

	metrics.SearchesActive.Dec()

	h.hub.Send(searchID, progress.Event{
		Event: "summary_done",
		Payload: map[string]any{
			"search_id": searchID,
			"summary":   summaryText,
		},
	})

	h.hub.Send(searchID, progress.Event{
		Event: "search_done",
		Payload: map[string]any{
			"search_id": searchID,
			"stats":     s.Stats,
		},
	})

	// Close hub after brief delay to let events flush
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.hub.Close(searchID)
	}()
}

func (h *Handler) GetSearch(w http.ResponseWriter, r *http.Request) {
	searchID := extractPathParam(r.URL.Path, "/api/v1/search/")
	if searchID == "" || strings.Contains(searchID, "/") {
		writeProblem(w, http.StatusBadRequest, "invalid_id", "Invalid search ID")
		return
	}

	s, ok := h.store.Get(searchID)
	if !ok {
		writeProblem(w, http.StatusNotFound, "not_found", "Search not found")
		return
	}

	respondJSON(w, http.StatusOK, s)
}

func (h *Handler) StreamSearch(w http.ResponseWriter, r *http.Request) {
	searchID := extractStreamID(r.URL.Path)
	if searchID == "" {
		writeProblem(w, http.StatusBadRequest, "invalid_id", "Invalid search ID")
		return
	}

	h.sse.Stream(w, r, searchID)
}

func (h *Handler) ListSearches(w http.ResponseWriter, r *http.Request) {
	searches := h.store.List()
	type summary struct {
		ID        string             `json:"search_id"`
		Query     string             `json:"query"`
		QueryType detection.QueryType `json:"query_type"`
		Status    store.SearchStatus `json:"status"`
		StartedAt time.Time          `json:"started_at"`
	}
	out := make([]summary, 0, len(searches))
	for _, s := range searches {
		out = append(out, summary{
			ID:        s.ID,
			Query:     s.Query,
			QueryType: s.QueryType,
			Status:    s.Status,
			StartedAt: s.StartedAt,
		})
	}
	respondJSON(w, http.StatusOK, out)
}

func (h *Handler) DeleteSearch(w http.ResponseWriter, r *http.Request) {
	searchID := extractPathParam(r.URL.Path, "/api/v1/search/")
	if searchID == "" || strings.Contains(searchID, "/") {
		writeProblem(w, http.StatusBadRequest, "invalid_id", "Invalid search ID")
		return
	}

	h.store.Update(searchID, func(s *store.Search) {
		s.Status = store.StatusCancelled
	})
	h.hub.Close(searchID)
	h.store.Delete(searchID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type webSearchRequest struct {
	Query    string   `json:"query"`
	Type     string   `json:"type"`
	Count    int      `json:"count"`
	LiveCrawl string  `json:"livecrawl"`
	URLs     []string `json:"urls,omitempty"`
	Format   string   `json:"format,omitempty"`
}

func (h *Handler) WebSearch(w http.ResponseWriter, r *http.Request) {
	var req webSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" && len(req.URLs) == 0 {
		writeProblem(w, http.StatusBadRequest, "empty_query", "Query or urls is required")
		return
	}
	if req.Count <= 0 || req.Count > 50 {
		req.Count = 10
	}

	opts := websearch.Options{
		Count: req.Count,
		Type:  websearch.SearchType(req.Type),
	}

	switch req.LiveCrawl {
	case "preferred":
		opts.LiveCrawl = websearch.LiveCrawlPreferred
	default:
		opts.LiveCrawl = websearch.LiveCrawlFallback
	}

	if opts.Type == "" {
		opts.Type = websearch.SearchTypeAuto
	}
	if opts.Type != websearch.SearchTypeAuto && opts.Type != websearch.SearchTypeFast && opts.Type != websearch.SearchTypeDeep {
		opts.Type = websearch.SearchTypeAuto
	}

	type response struct {
		Query   string              `json:"query,omitempty"`
		Results []websearch.Result  `json:"results,omitempty"`
		Pages   []*websearch.FetchedPage `json:"pages,omitempty"`
		Total   int                 `json:"total"`
		Sources []string            `json:"sources"`
	}

	resp := response{Query: req.Query}

	if len(req.URLs) > 0 {
		fetchOpts := websearch.DefaultFetchOptions()
		if opts.Type == websearch.SearchTypeDeep {
			fetchOpts.Timeout = 30 * time.Second
		}
		if req.Format != "" {
			fetchOpts.Format = req.Format
		}
		pages := h.websearch.FetchURLs(r.Context(), req.URLs, fetchOpts)
		resp.Pages = pages
		resp.Total = len(pages)
		for _, p := range pages {
			resp.Sources = append(resp.Sources, p.URL)
		}
	}

	if req.Query != "" {
		results, err := h.websearch.Search(r.Context(), req.Query, opts)
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "search_error", err.Error())
			return
		}
		resp.Results = results
		resp.Total += len(results)

		seen := make(map[string]bool)
		for _, r := range results {
			if !seen[r.Engine] {
				seen[r.Engine] = true
				resp.Sources = append(resp.Sources, r.Engine)
			}
		}
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func extractPathParam(path, prefix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimSuffix(trimmed, "/stream")
	trimmed = strings.TrimSuffix(trimmed, "/")
	return trimmed
}

func extractStreamID(path string) string {
	trimmed := strings.TrimPrefix(path, "/api/v1/search/")
	trimmed = strings.TrimSuffix(trimmed, "/stream")
	return trimmed
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
