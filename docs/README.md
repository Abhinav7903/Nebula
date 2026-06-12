# 🌌 Nebula

> **Production-grade OSINT & intelligence collection platform built in Go**

[![Go Version](https://img.shields.io/badge/Go-1.22-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue)](../docker/)
[![Collectors](https://img.shields.io/badge/collectors-22-purple)](../internal/collectors/)

Nebula is a self-hosted intelligence aggregation engine that fans out a single query — a domain, IP, email, crypto wallet, or hash — across 22+ OSINT data sources simultaneously. Results stream back in real time via Server-Sent Events and are optionally summarized by an AI model (Groq / LLaMA 3.3-70b).

---

## Table of Contents

- [Features](#features)
- [Repository Structure](#repository-structure)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Keys — Where to Get Them](#api-keys--where-to-get-them)
- [HTTP API Reference](#http-api-reference)
- [Collectors](#collectors)
- [Adding a New Collector](#adding-a-new-collector)
- [Docker](#docker)
- [GeoIP Setup](#geoip-setup)
- [Tor Support](#tor-support)
- [Metrics](#metrics)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **22 collectors** covering network, threat intelligence, email reputation, blockchain, and code sources
- **Real-time SSE streaming** — results appear as each collector finishes, no polling required
- **AI-powered summary** via Groq (LLaMA 3.3-70b); gracefully falls back to a count summary if no key is set
- **Priority job queue** — 500-worker goroutine pool, 10k job capacity, 30 s timeout, 3 retries, dead-letter queue
- **Automatic query-type detection** — domain, IPv4/IPv6, email, crypto wallet, file hash, URL, ASN, CVE
- **In-memory store** with TTL eviction (default 5 min) — no database required
- **Per-IP rate limiting** (10 req/min, burst 3) and optional API-key auth
- **Prometheus metrics** at `/metrics`
- **Tor support** (SOCKS5, disabled by default)
- **Docker-ready** with Docker Compose

---

## Repository Structure

```
nebula/
├── cmd/
│   └── nebula/
│       └── main.go              # Entrypoint — server init, collector registration
├── configs/
│   └── config.yaml              # Master configuration (supports ${VAR} substitution)
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── docs/
│   ├── README.md                # ← You are here
│   ├── Architecture.md          # Detailed system architecture
│   ├── CollectorGuide.md        # Step-by-step guide to adding collectors
│   └── LICENSE
├── internal/
│   ├── api/                     # HTTP handlers (search, stream, health)
│   ├── collectors/              # 22 collector implementations + interface
│   │   ├── interface.go         # Collector interface definition
│   │   ├── dns/
│   │   ├── shodan/
│   │   ├── censys/
│   │   ├── virustotal/
│   │   ├── greynoise/
│   │   ├── github/
│   │   ├── emailrep/
│   │   ├── urlscan/
│   │   ├── etherscan/
│   │   ├── tron/
│   │   └── ...                  # (remaining 12 collectors)
│   ├── config/                  # Config loading, env var resolution, api.txt mapping
│   ├── deduplication/           # Cross-collector result deduplication
│   ├── detection/               # Query-type auto-detection (regex + heuristics)
│   ├── logger/
│   ├── metrics/                 # Prometheus instrumentation
│   ├── normalization/           # Result normalisation across sources
│   ├── progress/                # SSE hub — per-search event fan-out
│   ├── queue/                   # Priority job queue
│   ├── ranking/                 # Result relevance scoring
│   ├── store/                   # In-memory search/result store with TTL
│   ├── summary/                 # Groq AI summary integration
│   ├── tor/                     # Optional Tor/SOCKS5 HTTP transport
│   └── workers/                 # Goroutine pool
├── web/                         # Frontend (TypeScript + CSS)
├── data/                        # GeoLite2 databases (not in repo — see below)
│   ├── GeoLite2-City.mmdb
│   └── GeoLite2-ASN.mmdb
├── .env.example                 # Environment variable template
├── .gitignore
├── AGENTS.md                    # AI-agent coding guide for this repo
├── Makefile                     # build / run / test / lint / docker targets
├── go.mod
└── go.sum
```

---

## Architecture

```
HTTP Client
    │
    ▼
┌──────────────────┐
│   API Layer      │  POST /api/v1/search
│   (net/http)     │  GET  /api/v1/search/{id}
└────────┬─────────┘  GET  /api/v1/search/{id}/stream (SSE)
         │
         ▼
┌──────────────────┐
│  Middleware       │  Auth · Rate-Limit · Recovery · Logging
└────────┬─────────┘
         │
         ▼
┌──────────────────┐     ┌──────────────────┐
│ Detection Engine │────▶│  Priority Queue  │
│ (query type)     │     │  (10k capacity)  │
└──────────────────┘     └────────┬─────────┘
                                  │
                          ┌───────┼──────────┐
                          ▼       ▼          ▼
                      ┌───────┐ ┌────────┐ ┌───────┐
                      │ DNS   │ │ Shodan │ │  VT   │  ... 22 collectors
                      └───┬───┘ └───┬────┘ └───┬───┘
                          └─────────┴───────────┘
                                    │
                                    ▼
                          ┌──────────────────┐
                          │   SSE Hub        │  streams results live
                          └────────┬─────────┘
                                   │
                                   ▼
                          ┌──────────────────┐
                          │  AI Summary      │  Groq / LLaMA 3.3-70b
                          └──────────────────┘
```

See [`Architecture.md`](Architecture.md) for the full breakdown.

---

## Quick Start

```bash
# 1. Clone
git clone https://github.com/Abhinav7903/Nebula.git
cd Nebula

# 2. Configure
cp .env.example .env
# Open .env and fill in the API keys you have (all are optional except GROQ for AI summary)

# 3. Build & run
make build
make run

# Server is now listening on http://localhost:8080
```

---

## Configuration

Primary config lives in [`../configs/config.yaml`](../configs/config.yaml). All `${VAR}` placeholders are substituted at startup from environment variables.

| Setting | Default | Description |
|---|---|---|
| `server.host` | `0.0.0.0` | Bind address |
| `server.port` | `8080` | HTTP port |
| `queue.workers` | `500` | Goroutine worker pool size |
| `queue.max_size` | `10000` | Max queued jobs |
| `queue.job_timeout` | `30s` | Per-collector timeout |
| `rate_limit.requests_per_minute` | `10` | Per-IP request limit |
| `rate_limit.burst` | `3` | Burst allowance |
| `store.ttl` | `5m` | In-memory result TTL |
| `api_keys.require_key` | `false` | Enforce `X-API-Key` header |
| `tor.enabled` | `false` | Route through Tor SOCKS5 |
| `tor.address` | `127.0.0.1:9050` | Tor proxy address |

API keys are loaded from `.env` (or real environment variables). The config path can be overridden with `NEBULA_CONFIG`.

---

## API Keys — Where to Get Them

Copy [`../.env.example`](../.env.example) to `.env` and fill in whichever keys you want to enable.

```bash
cp .env.example .env
```

| Variable | Service | Get your key at |
|---|---|---|
| `SHODAN_KEY` | Shodan — internet-wide port/banner scanner | https://account.shodan.io |
| `CENSYS_KEY` | Censys — certificate & host search | https://app.censys.io/account |
| `VT_KEY` | VirusTotal — malware & URL reputation | https://www.virustotal.com/gui/my-apikey |
| `GREYNOISE_KEY` | GreyNoise — benign/malicious IP classification | https://viz.greynoise.io/account/profile |
| `GITHUB_TOKEN` | GitHub — code & repo search | https://github.com/settings/tokens |
| `EMAILREP_KEY` | EmailRep — email risk scoring | https://emailrep.io/key |
| `URLSCAN_KEY` | URLScan.io — website scan & screenshot | https://urlscan.io/user/profile |
| `ETHERSCAN_KEY` | Etherscan — Ethereum on-chain data | https://etherscan.io/myapikey |
| `TRON_KEY` | TronScan — Tron blockchain data | https://tronscan.org/#/setting/api |
| `GROQ_API_KEY` | Groq — AI summary (LLaMA 3.3-70b) | https://console.groq.com/keys |

> **Note:** All keys are optional. Collectors with no key set are automatically skipped at startup. The platform works with zero keys (DNS, WHOIS, crt.sh, and other free collectors still run). The AI summary falls back to a simple result count if `GROQ_API_KEY` is absent.

> **Security:** Never commit `.env` or `api.txt` to version control. Both are in `.gitignore` by default.

---

## HTTP API Reference

All endpoints are under `http://localhost:8080`.

### Create a search

```bash
POST /api/v1/search
Content-Type: application/json

{"query": "example.com"}
```

Response:
```json
{"id": "abc123", "query": "example.com", "type": "domain", "status": "running"}
```

### Stream results (SSE)

```bash
GET /api/v1/search/{id}/stream
Accept: text/event-stream
```

Results arrive as SSE events as each collector finishes. Connect before or immediately after creating the search.

### Get all results

```bash
GET /api/v1/search/{id}
```

Returns the complete result set once the search is complete (or partially complete).

### List all searches

```bash
GET /api/v1/searches
```

### Health check

```bash
GET /health
```

### Prometheus metrics

```bash
GET /metrics
```

### Authentication (optional)

When `api_keys.require_key: true` is set in config, include the header:

```
X-API-Key: <your-configured-key>
```

---

## Collectors

Nebula ships with 22 collectors across multiple intelligence categories:

**Network & Infrastructure**
- `dns` — A, AAAA, MX, TXT, NS, CNAME resolution
- `whois` — domain/IP registration data
- `shodan` — port scans, banners, CVEs *(requires `SHODAN_KEY`)*
- `censys` — TLS certificates and host data *(requires `CENSYS_KEY`)*

**Threat Intelligence**
- `virustotal` — malware, URL, and file hash reputation *(requires `VT_KEY`)*
- `greynoise` — IP noise classification *(requires `GREYNOISE_KEY`)*

**Certificate Transparency**
- `crtsh` — certificate search via crt.sh (no key required)

**Email & Reputation**
- `emailrep` — email risk score and breach history *(requires `EMAILREP_KEY`)*

**Web & URL Analysis**
- `urlscan` — passive and active website scans *(requires `URLSCAN_KEY`)*

**Code & Leak Detection**
- `github` — code search across public repositories *(requires `GITHUB_TOKEN`)*

**Blockchain / Web3**
- `etherscan` — Ethereum address/transaction lookup *(requires `ETHERSCAN_KEY`)*
- `tron` — Tron address/transaction lookup *(requires `TRON_KEY`)*

> The remaining 10 collectors target additional data sources. See `internal/collectors/` for the full list.

---

## Adding a New Collector

See [`CollectorGuide.md`](CollectorGuide.md) for the complete walkthrough. The short version:

1. Create `internal/collectors/myservice/collector.go` implementing the `Collector` interface
2. Register it in `cmd/nebula/main.go` → `registerCollectors()`
3. Add config to `configs/config.yaml`
4. Add the config struct to `internal/config/config.go`
5. Add the env var to `.env.example`

```go
type Collector interface {
    Name() string
    SupportedTypes() []string   // e.g. []string{"domain", "ipv4"}
    RequiresKey() bool
    Execute(ctx context.Context, query, qtype string) ([]Result, error)
}
```

---

## Docker

```bash
# Build image
make docker-build

# Start with Docker Compose
docker compose -f ../docker/docker-compose.yml up

# With env vars
docker compose -f ../docker/docker-compose.yml --env-file ../.env up
```

---

## GeoIP Setup

Nebula supports GeoIP enrichment via MaxMind's free GeoLite2 databases. These are **not bundled** in the repository due to licensing.

1. Sign up free at https://www.maxmind.com/en/geolite2/signup
2. Download `GeoLite2-City.mmdb` and `GeoLite2-ASN.mmdb`
3. Place both files in `data/` at the repo root:

```
../data/
├── GeoLite2-City.mmdb
└── GeoLite2-ASN.mmdb
```

GeoIP lookup is skipped gracefully if the files are absent.

---

## Tor Support

To route all collector HTTP requests through Tor:

1. Install and start Tor (default SOCKS5 port `9050`)
2. In [`../configs/config.yaml`](../configs/config.yaml):

```yaml
tor:
  enabled: true
  address: "127.0.0.1:9050"
```

---

## Metrics

Prometheus metrics are exposed at `GET /metrics`. Scrape with:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: nebula
    static_configs:
      - targets: ['localhost:8080']
```

---

## Development

```bash
make build        # Compile binary → bin/nebula
make run          # Build + run
make test         # go test -v -race -count=1 ./...
make lint         # go vet ./...
make docker-build # Build Docker image
```

**Module path note:** `go.mod` currently uses the placeholder path `github.com/Abhinav7903/nebula`. Update it to your actual path if forking:

```bash
go mod edit -module github.com/Abhinav7903/Nebula
# Then update all internal imports accordingly
```

---

## Contributing

1. Fork the repo and create a feature branch
2. Follow the collector pattern in [`CollectorGuide.md`](CollectorGuide.md)
3. Run `make test` and `make lint` before opening a PR
4. Keep PRs focused — one collector or one feature per PR

---

## License

MIT — see [`LICENSE`](LICENSE).
