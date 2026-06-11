# Deployment

## Docker

### Build

```bash
make docker-build
```

### Run with Docker Compose

```bash
# Create .env file with API keys
cp .env.example .env
# Edit .env

# Start Nebula
docker compose -f docker/docker-compose.yml up -d

# Start with Tor support
docker compose -f docker/docker-compose.yml --profile tor up -d
```

### Configuration Mounting

The `docker-compose.yml` mounts `configs/config.yaml` into the container. Edit this file and restart to apply changes.

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `GROQ_API_KEY` | For AI summaries | Groq API key for LLaMA 3.3 |
| `SHODAN_KEY` | For Shodan | Shodan API key |
| `CENSYS_ID` | For Censys | Censys API ID |
| `CENSYS_SECRET` | For Censys | Censys API secret |
| `VT_KEY` | For VirusTotal | VirusTotal API key |
| `GITHUB_TOKEN` | For GitHub search | GitHub personal access token |

See `.env.example` for the full list.

## GeoIP MaxMind

For GeoIP lookups, download the free GeoLite2 databases:

```bash
mkdir -p data
wget https://git.io/GeoLite2-City.mmdb -O data/GeoLite2-City.mmdb
wget https://git.io/GeoLite2-ASN.mmdb -O data/GeoLite2-ASN.mmdb
```

Set paths in `configs/config.yaml` under `geoip`.

## Production Considerations

1. Set `api_keys.require_key: true` in production
2. Configure rate limiting per your needs
3. Use a reverse proxy (nginx/Caddy) for TLS termination
4. Monitor with Prometheus metrics at `/metrics`
5. Enable Tor for onion service lookups
6. Adjust worker count based on available CPU

## Health Checks

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Monitoring

Prometheus metrics available at `/metrics`:

- `nebula_searches_total`
- `nebula_searches_active`
- `nebula_queue_depth`
- `nebula_workers_busy`
- `nebula_collector_duration_seconds`
