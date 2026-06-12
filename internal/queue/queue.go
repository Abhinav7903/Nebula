package queue

import (
	"container/heap"
	"sync"
	"time"

	"github.com/Abhinav7903/nebula/internal/metrics"
)

type PriorityQueue []*Job

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority < pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x any) {
	*pq = append(*pq, x.(*Job))
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[0 : n-1]
	return item
}

type Queue struct {
	mu        sync.RWMutex
	pq        PriorityQueue
	maxSize   int
	dlq       []*Job
	dlqMu     sync.Mutex
	cond      *sync.Cond
	startTime time.Time
}

func New(maxSize int) *Queue {
	q := &Queue{
		pq:      make(PriorityQueue, 0),
		maxSize: maxSize,
		startTime: time.Now(),
	}
	q.cond = sync.NewCond(&q.mu)
	heap.Init(&q.pq)
	return q
}

func (q *Queue) Push(job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.pq.Len() >= q.maxSize {
		return ErrQueueFull
	}
	heap.Push(&q.pq, job)
	metrics.QueueDepth.Set(float64(q.pq.Len()))
	q.cond.Signal()
	return nil
}

func (q *Queue) Pop() (*Job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for q.pq.Len() == 0 {
		q.cond.Wait()
	}
	job := heap.Pop(&q.pq).(*Job)
	metrics.QueueDepth.Set(float64(q.pq.Len()))
	return job, true
}

func (q *Queue) PopWithCtx(done <-chan struct{}) (*Job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for q.pq.Len() == 0 {
		select {
		case <-done:
			return nil, false
		default:
		}
		q.cond.Wait()
	}
	job := heap.Pop(&q.pq).(*Job)
	metrics.QueueDepth.Set(float64(q.pq.Len()))
	return job, true
}

func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.pq.Len()
}

func (q *Queue) SendToDLQ(job *Job) {
	q.dlqMu.Lock()
	defer q.dlqMu.Unlock()
	q.dlq = append(q.dlq, job)
}

func (q *Queue) DLQLen() int {
	q.dlqMu.Lock()
	defer q.dlqMu.Unlock()
	return len(q.dlq)
}
