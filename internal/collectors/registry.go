package collectors

import "sync"

type Registry struct {
	mu     sync.RWMutex
	byName map[string]Collector
}

func NewRegistry() *Registry {
	return &Registry{
		byName: make(map[string]Collector),
	}
}

func (r *Registry) Register(c Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byName[c.Name()] = c
}

func (r *Registry) Get(name string) Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[name]
}

func (r *Registry) All() []Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Collector, 0, len(r.byName))
	for _, c := range r.byName {
		out = append(out, c)
	}
	return out
}

func (r *Registry) ByType(qtype string) []Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Collector
	for _, c := range r.byName {
		for _, t := range c.SupportedTypes() {
			if t == qtype {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

func (r *Registry) NamesByType(qtype string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for _, c := range r.byName {
		for _, t := range c.SupportedTypes() {
			if t == qtype {
				out = append(out, c.Name())
				break
			}
		}
	}
	return out
}
