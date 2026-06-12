# Nebula — AGENTS.md

## Commands

```
make build          # go build -ldflags="-s -w" -o bin/nebula ./cmd/nebula
make run            # build + run
make test           # go test -v -race -count=1 ./...
make lint           # go vet ./...
make docker-build   # docker build -f Dockerfile .
```

Config path overridable via `NEBULA_CONFIG` env var.

## Entrypoint & Structure

- `cmd/nebula/main.go` — single binary, no subcommands
- `internal/` has 15 packages: api, collectors (22 collectors), config, deduplication, detection, logger, metrics, normalization, progress, queue, ranking, store, summary, tor, workers
- `configs/config.yaml` — YAML config with `${VAR}` env var substitution
- `internal/collectors/interface.go` — `Collector` interface (`Name`, `SupportedTypes`, `RequiresKey`, `Execute`)

## API Keys

Two sources, both feed into `os.Getenv`:
1. **`api.txt`** (legacy) — loaded first via `config.LoadEnvFrom()` with hardcoded key mapping (`apiKeyMapping` in `internal/config/config.go`). Path overridable via `NEBULA_API_TXT`.
2. **`.env` / environment variables** — standard env var names (see `.env.example`).

**Warning**: `api.txt` at repo root contains plaintext API keys. Do not commit it.

## Config & Key Resolution

1. `config.Load()` reads `api.txt` → `os.Setenv`
2. Unmarshal `config.yaml` (contains `${SHODAN_KEY}` etc.)
3. `ResolveEnv()` replaces `${VAR}` with `os.Getenv` values

## Collector Registration

Pattern in `main.go:registerCollectors()`:
- `always()` — registers unconditionally (DNS, whois, crtsh, etc.)
- `whenEnabled()` — skips if disabled in config or requires key but key is empty (Shodan, Censys, VirusTotal, etc.)

## HTTP API

- Go 1.22 ServeMux patterns: `POST /api/v1/search`, `GET /api/v1/search/{id}`, `GET /api/v1/search/{id}/stream` (SSE)
- Auth via `X-API-Key` header — disabled by default (`api_keys.require_key: false`)
- Rate limit: 10 req/min per IP, burst 3
- Responses use `application/problem+json` for errors

## Other Notes

- In-memory store with 5min TTL eviction — no database
- Worker pool: 500 workers, priority queue, max 10k queued, 30s job timeout, 3 retries, dead-letter queue
- SSE streaming via `internal/progress/hub.go`
- AI summary via Groq (LLaMA 3.3-70b) — skipped if `GROQ_API_KEY` not set (falls back to counting results)
- GeoIP expects `data/GeoLite2-City.mmdb` and `data/GeoLite2-ASN.mmdb` (not in repo)
- Tor support disabled by default, SOCKS5 on `127.0.0.1:9050`
- Prometheus metrics at `GET /metrics`
- Only existing test: `internal/detection/detector_test.go`
- No CI, no golangci-lint config, no pre-commit hooks
- Module path `github.com/yourusername/nebula` — may need updating
