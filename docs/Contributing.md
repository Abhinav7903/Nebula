# Contributing

## Getting Started

1. Fork the repository
2. Install Go 1.22+
3. Run `go mod tidy` to install dependencies
4. Run `make test` to verify everything works

## Code Style

- Standard library first
- No external HTTP frameworks
- No ORMs
- Dependency injection via constructors
- Clean architecture: handler → service → repository
- SOLID principles
- Follow existing patterns

## Adding a Collector

See [CollectorGuide.md](CollectorGuide.md).

## Testing

```bash
make test          # Run all tests
go test ./...      # Run all tests with verbose
go vet ./...       # Lint
```

## Pull Request Process

1. Create a feature branch
2. Write tests for new functionality
3. Ensure all tests pass
4. Run `go vet ./...`
5. Submit PR with description of changes

## Project Structure

```
cmd/nebula/          # Entrypoint
internal/
  api/               # HTTP handlers, middleware, SSE
  collectors/        # All collectors
  config/            # Configuration
  detection/         # Query type detection
  queue/             # Job queue
  workers/           # Worker pool
  progress/          # SSE event hub
  summary/           # AI summarizer
  store/             # In-memory store
  metrics/           # Prometheus metrics
  logger/            # Structured logging
  tor/               # SOCKS5 Tor client
configs/             # Configuration files
docker/              # Docker build files
docs/                # Documentation
```
