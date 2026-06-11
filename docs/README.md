# Nebula

Nebula is a production-grade OSINT and intelligence collection platform built in Go.

[![Go Version](https://img.shields.io/badge/Go-1.22-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue)](docker/Dockerfile)

## Quick Start

```bash
# Clone and build
git clone <repo>
cd nebula
make build

# Configure
cp .env.example .env
# Edit .env with API keys

# Run
make run
```

## Configuration

See `configs/config.yaml` for full configuration. Key settings:

| Setting | Default | Description |
|---|---|---|
| `server.host` | 0.0.0.0 | Bind address |
| `server.port` | 8080 | HTTP port |
| `queue.workers` | 500 | Worker pool size |
| `rate_limit.requests_per_minute` | 10 | Per-IP rate limit |

API keys are configured via environment variables (see `.env.example`).

## API

```bash
# Create a search
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "example.com"}'

# Stream results via SSE
curl http://localhost:8080/api/v1/search/{id}/stream

# Get results
curl http://localhost:8080/api/v1/search/{id}

# List searches
curl http://localhost:8080/api/v1/searches

# Health check
curl http://localhost:8080/health
```

## Architecture

See [Architecture.md](Architecture.md) for a detailed breakdown.

## Adding a Collector

See [CollectorGuide.md](CollectorGuide.md) for a step-by-step walkthrough.

## Docker

```bash
make docker-build
docker compose -f docker/docker-compose.yml up
```
