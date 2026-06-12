package store

import (
	"sync"
	"time"

	"github.com/Abhinav7903/nebula/internal/collectors"
	"github.com/Abhinav7903/nebula/internal/detection"
)

type SearchStatus string

const (
	StatusPending SearchStatus = "pending"
	StatusRunning SearchStatus = "running"
	StatusDone    SearchStatus = "done"
	StatusFailed  SearchStatus = "failed"
	StatusCancelled SearchStatus = "cancelled"
)

type Stats struct {
	CollectorsRun    int   `json:"collectors_run"`
	CollectorsOK     int   `json:"collectors_ok"`
	CollectorsFailed int   `json:"collectors_failed"`
	ResultsFound     int   `json:"results_found"`
	DurationMs       int64 `json:"duration_ms"`
}

type Search struct {
	ID                  string          `json:"search_id"`
	Query               string          `json:"query"`
	QueryType           detection.QueryType `json:"query_type"`
	Status              SearchStatus    `json:"status"`
	StartedAt           time.Time       `json:"started_at"`
	FinishedAt          *time.Time      `json:"finished_at,omitempty"`
	Summary             string          `json:"summary"`
	Stats               Stats           `json:"stats"`
	Results             []collectors.Result `json:"results"`
	Collectors          []string        `json:"-"`
	CollectorsCompleted map[string]bool `json:"-"`
}

type MemoryStore struct {
	mu       sync.RWMutex
	searches map[string]*Search
	ttl      time.Duration
}

func NewMemoryStore(ttl time.Duration) *MemoryStore {
	s := &MemoryStore{
		searches: make(map[string]*Search),
		ttl:      ttl,
	}
	if ttl > 0 {
		go s.evictLoop()
	}
	return s
}

func (s *MemoryStore) Create(search *Search) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searches[search.ID] = search
}

func (s *MemoryStore) Get(id string) (*Search, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	search, ok := s.searches[id]
	if !ok {
		return nil, false
	}
	return search, true
}

func (s *MemoryStore) Update(id string, fn func(*Search)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if search, ok := s.searches[id]; ok {
		fn(search)
	}
}

func (s *MemoryStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.searches, id)
}

func (s *MemoryStore) List() []*Search {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Search, 0, len(s.searches))
	for _, search := range s.searches {
		out = append(out, search)
	}
	return out
}

func (s *MemoryStore) AddResult(searchID string, result collectors.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if search, ok := s.searches[searchID]; ok {
		search.Results = append(search.Results, result)
	}
}

func (s *MemoryStore) MarkCollectorDone(searchID, collector string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if search, ok := s.searches[searchID]; ok {
		if search.CollectorsCompleted == nil {
			search.CollectorsCompleted = make(map[string]bool)
		}
		search.CollectorsCompleted[collector] = true
	}
}

func (s *MemoryStore) evictLoop() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, search := range s.searches {
			if search.FinishedAt != nil && now.After(search.FinishedAt.Add(s.ttl)) {
				delete(s.searches, id)
			}
		}
		s.mu.Unlock()
	}
}
