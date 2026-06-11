# Architecture

## Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  HTTP/S     │────▶│  API Layer   │────▶│  Detection  │
│  Client     │     │  (net/http)  │     │  Engine     │
└─────────────┘     └──────┬───────┘     └──────┬──────┘
                           │                     │
                           ▼                     ▼
                    ┌──────────────┐     ┌──────────────┐
                    │  Middleware   │     │  Job Queue   │
                    │  Auth/Rate/  │     │  (Priority)  │
                    │  Recovery    │     │              │
                    └──────────────┘     └──────┬───────┘
                                                │
                                                ▼
                                         ┌──────────────┐
                                         │  Worker Pool │
                                         │  (500 workers)│
                                         └──────┬───────┘
                                                │
                                    ┌───────────┼───────────┐
                                    ▼           ▼           ▼
                              ┌─────────┐ ┌─────────┐ ┌─────────┐
                              │ DNS     │ │ Shodan  │ │ Virus-  │
                              │ Col-    │ │ Col-    │ │ total   │
                              │ lector  │ │ lector  │ │ Col-    │
                              └─────────┘ └─────────┘ │ lector  │
                                                       └─────────┘
                                    │           │           │
                                    └───────────┼───────────┘
                                                ▼
                                         ┌──────────────┐
                                         │  SSE Hub     │
                                         │  (Progress)  │
                                         └──────┬───────┘
                                                │
                                    ┌───────────┘
                                    ▼
                              ┌──────────────┐
                              │  AI Summary  │
                              │  (Groq)      │
                              └──────────────┘
```

## Layers

### Handler Layer
HTTP handlers accept requests, delegate to the detection engine and job queue, and return responses. No business logic lives here.

### Detection Engine
Determines query type using regex and heuristics. See [Detector documentation](internal/detection/detector.go).

### Job Queue
In-memory priority queue with configurable capacity. Jobs are prioritized and dispatched to workers.

### Worker Pool
Configurable pool of goroutines that execute collectors concurrently. Each worker picks jobs from the queue, executes the associated collector, and streams results via SSE.

### Collector Layer
Each collector implements the `Collector` interface:

```go
type Collector interface {
    Name() string
    SupportedTypes() []string
    RequiresKey() bool
    Execute(ctx context.Context, query string, qtype string) ([]Result, error)
}
```

### SSE Hub
Manages per-search event channels. Collectors emit progress events that are fanned out to all SSE subscribers.

### AI Summary
After all collectors finish, results are sent to Groq (LLaMA 3.3) for an intelligence summary.

## In-Memory Store

Searches and results are stored in memory with TTL-based eviction (default 5 minutes). No database required.
