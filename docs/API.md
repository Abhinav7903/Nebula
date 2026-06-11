# API Reference

Base URL: `http://localhost:8080`

## Create Search

```http
POST /api/v1/search
Content-Type: application/json

{"query": "example.com"}
```

Response:

```json
{
  "search_id": "uuid",
  "query_type": "domain"
}
```

## Get Search Results

```http
GET /api/v1/search/{id}
```

Response: Full search response with results, stats, and summary.

## Stream SSE Events

```http
GET /api/v1/search/{id}/stream
```

Events (in order):

| Event | Payload |
|---|---|
| `search_started` | `{ search_id, query, query_type, collectors_planned }` |
| `collector_started` | `{ search_id, collector }` |
| `collector_result` | `{ search_id, collector, result }` |
| `collector_done` | `{ search_id, collector, results_count, duration_ms }` |
| `summary_started` | `{ search_id }` |
| `summary_done` | `{ search_id, summary }` |
| `search_done` | `{ search_id, stats }` |

## List Searches

```http
GET /api/v1/searches
```

## Delete Search

```http
DELETE /api/v1/search/{id}
```

## Health

```http
GET /health
→ {"status": "ok"}
```

## Readiness

```http
GET /ready
→ {"status": "ok"}
```

## Metrics

```http
GET /metrics
```

Prometheus metrics in text format.

## Error Responses

All errors return RFC 7807 problem JSON:

```json
{
  "type": "not_found",
  "title": "Not Found",
  "status": 404,
  "detail": "Search not found"
}
```
