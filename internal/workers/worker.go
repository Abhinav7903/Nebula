package workers

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/Abhinav7903/nebula/internal/collectors"
	"github.com/Abhinav7903/nebula/internal/metrics"
	"github.com/Abhinav7903/nebula/internal/progress"
	"github.com/Abhinav7903/nebula/internal/queue"
	"github.com/Abhinav7903/nebula/internal/store"
)

type Pool struct {
	queue      *queue.Queue
	registry   *collectors.Registry
	hub        *progress.Hub
	store      *store.MemoryStore
	logger     *slog.Logger
	numWorkers int
	done       chan struct{}
}

func NewPool(q *queue.Queue, reg *collectors.Registry, hub *progress.Hub, s *store.MemoryStore, logger *slog.Logger, numWorkers int) *Pool {
	return &Pool{
		queue:      q,
		registry:   reg,
		hub:        hub,
		store:      s,
		logger:     logger,
		numWorkers: numWorkers,
		done:       make(chan struct{}),
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.numWorkers; i++ {
		go p.runWorker(ctx, i)
	}
}

func (p *Pool) Stop() {
	close(p.done)
}

func (p *Pool) runWorker(ctx context.Context, id int) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("worker panic",
				"worker_id", id,
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		default:
		}

		job, ok := p.queue.PopWithCtx(p.done)
		if !ok {
			return
		}

		metrics.WorkersBusy.Inc()
		p.processJob(job)
		metrics.WorkersBusy.Dec()
	}
}

func (p *Pool) processJob(job *queue.Job) {
	log := p.logger.With("search_id", job.SearchID, "collector", job.Collector)
	log.Info("starting collector")

	p.hub.Send(job.SearchID, progress.Event{
		Event: "collector_started",
		Payload: map[string]any{
			"search_id": job.SearchID,
			"collector": job.Collector,
		},
	})

	collector := p.registry.Get(job.Collector)
	if collector == nil {
		log.Warn("collector not found")
		return
	}

	start := time.Now()
	results, err := collector.Execute(job.Ctx, job.Query, job.QueryType)
	duration := time.Since(start)

	metrics.CollectorDuration.WithLabelValues(job.Collector).Observe(duration.Seconds())

	if err != nil {
		metrics.ErrorsTotal.WithLabelValues(job.Collector).Inc()
		log.Error("collector failed", "error", err, "duration_ms", duration.Milliseconds())

		if job.Retries < job.MaxRetries {
			job.Retries++
			job.Ctx, job.Cancel = context.WithTimeout(context.Background(), job.Timeout)
			if err := p.queue.Push(job); err != nil {
				p.queue.SendToDLQ(job)
			}
			return
		}

		p.queue.SendToDLQ(job)
		p.hub.Send(job.SearchID, progress.Event{
			Event: "collector_done",
			Payload: map[string]any{
				"search_id":   job.SearchID,
				"collector":   job.Collector,
				"error":       err.Error(),
				"duration_ms": duration.Milliseconds(),
			},
		})
		p.store.MarkCollectorDone(job.SearchID, job.Collector)
		return
	}

	metrics.CollectorResults.WithLabelValues(job.Collector).Add(float64(len(results)))

	for _, r := range results {
		p.store.AddResult(job.SearchID, r)
		p.hub.Send(job.SearchID, progress.Event{
			Event: "collector_result",
			Payload: map[string]any{
				"search_id": job.SearchID,
				"collector": job.Collector,
				"result":    r,
			},
		})
	}

	log.Info("collector done", "results", len(results), "duration_ms", duration.Milliseconds())
	p.hub.Send(job.SearchID, progress.Event{
		Event: "collector_done",
		Payload: map[string]any{
			"search_id":    job.SearchID,
			"collector":    job.Collector,
			"results_count": len(results),
			"duration_ms":  duration.Milliseconds(),
		},
	})
	p.store.MarkCollectorDone(job.SearchID, job.Collector)
}

func (p *Pool) EnqueueCollectors(searchID, query, qtype string, collectors []string) error {
	for _, name := range collectors {
		job := queue.NewJob(searchID, name, query, qtype, 2, 3, 30*time.Second)
		if err := p.queue.Push(job); err != nil {
			return fmt.Errorf("enqueue %s: %w", name, err)
		}
	}
	return nil
}
