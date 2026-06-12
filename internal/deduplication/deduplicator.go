package deduplication

import (
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/Abhinav7903/nebula/internal/collectors"
)

type Deduplicator struct {
	mu   sync.RWMutex
	seen map[string]struct{}
}

func New() *Deduplicator {
	return &Deduplicator{
		seen: make(map[string]struct{}),
	}
}

func (d *Deduplicator) IsDuplicate(result collectors.Result) bool {
	key := resultKey(result)
	d.mu.RLock()
	_, ok := d.seen[key]
	d.mu.RUnlock()
	return ok
}

func (d *Deduplicator) Mark(result collectors.Result) {
	key := resultKey(result)
	d.mu.Lock()
	d.seen[key] = struct{}{}
	d.mu.Unlock()
}

func (d *Deduplicator) Filter(rs []collectors.Result) []collectors.Result {
	out := make([]collectors.Result, 0, len(rs))
	for _, r := range rs {
		if d.IsDuplicate(r) {
			continue
		}
		d.Mark(r)
		out = append(out, r)
	}
	return out
}

func (d *Deduplicator) Reset() {
	d.mu.Lock()
	d.seen = make(map[string]struct{})
	d.mu.Unlock()
}

func resultKey(r collectors.Result) string {
	h := sha256.Sum256([]byte(r.Collector + "|" + r.Type + "|" + r.Title + "|" + r.Source))
	return fmt.Sprintf("%x", h[:8])
}
