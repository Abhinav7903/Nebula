package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID         string
	SearchID   string
	Collector  string
	Query      string
	QueryType  string
	Priority   int
	Retries    int
	MaxRetries int
	CreatedAt  time.Time
	Timeout    time.Duration
	Ctx        context.Context
	Cancel     context.CancelFunc
}

func NewJob(searchID, collector, query, qtype string, priority, maxRetries int, timeout time.Duration) *Job {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &Job{
		ID:         uuid.NewString(),
		SearchID:   searchID,
		Collector:  collector,
		Query:      query,
		QueryType:  qtype,
		Priority:   priority,
		MaxRetries: maxRetries,
		CreatedAt:  time.Now(),
		Timeout:    timeout,
		Ctx:        ctx,
		Cancel:     cancel,
	}
}
