package progress

import (
	"sync"
)

type Event struct {
	Event   string `json:"event"`
	Payload any    `json:"payload"`
}

type subscriber struct {
	ch     chan Event
	done   chan struct{}
	buffer int
}

type Hub struct {
	mu       sync.RWMutex
	subs     map[string][]*subscriber
	capacity int
}

func NewHub(capacity int) *Hub {
	return &Hub{
		subs:     make(map[string][]*subscriber),
		capacity: capacity,
	}
}

func (h *Hub) Subscribe(searchID string) (<-chan Event, chan struct{}, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub := &subscriber{
		ch:   make(chan Event, h.capacity),
		done: make(chan struct{}),
	}
	h.subs[searchID] = append(h.subs[searchID], sub)
	return sub.ch, sub.done, nil
}

func (h *Hub) Unsubscribe(searchID string, done chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	subs := h.subs[searchID]
	for i, s := range subs {
		if s.done == done {
			close(s.ch)
			h.subs[searchID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

func (h *Hub) Send(searchID string, evt Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, sub := range h.subs[searchID] {
		select {
		case sub.ch <- evt:
		default:
		}
	}
}

func (h *Hub) Close(searchID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, sub := range h.subs[searchID] {
		close(sub.ch)
	}
	delete(h.subs, searchID)
}
