# CLAUDE.md — Development Guidelines

This file provides context and conventions for Claude Code sessions working on this project.

## Project Summary

REST service in Go (Golang) for PROG2005 — Cloud Technologies. Aggregates live air quality and environmental data from five external APIs into configurable dashboards, with Firebase Firestore for persistence and webhook-based notifications.

**Module:** `envdash`
**Go version:** 1.26
**Port:** 8080 (default)

## Architecture

Layered, interface-driven architecture (handlers → services → repositories/clients):

```
cmd/server/main.go          entry point, dependency wiring
internal/config/            environment variable config
internal/models/            domain types
internal/handlers/          HTTP handlers (thin, delegate to services)
internal/services/          business logic
internal/clients/           external API clients (HTTPDoer interface)
internal/firebase/          Firestore repositories + cache
internal/webhook/           async webhook dispatcher
```

## Key Conventions

- **All errors** are returned as JSON: `{"error": "message"}` with appropriate HTTP status
- **Date format**: `time.Now().UTC().Format("20060102 15:04")` → `"20250301 09:15"`
- **IDs**: 16 hex chars generated with `crypto/rand`
- **External HTTP clients**: All implement `clients.HTTPDoer` interface for testability
- **Services and repositories**: Interface-based — handlers depend on interfaces, not concrete types
- **Concurrent API calls**: Use `sync.WaitGroup` + goroutines in dashboard service
- **Never hard-code** API keys, credentials, or base URLs — always use `Config`

## External APIs

| API | Base URL env var | Default |
|-----|-----------------|---------|
| REST Countries | `COUNTRIES_API_URL` | `http://129.241.150.113:8080/v3.1` |
| Open-Meteo | `METEO_API_URL` | `https://api.open-meteo.com/v1` |
| OpenAQ v3 | `OPENAQ_API_URL` | `https://api.openaq.org/v3` |
| Nominatim | `NOMINATIM_API_URL` | `https://nominatim.openstreetmap.org` |
| Currency | `CURRENCY_API_URL` | `http://129.241.150.113:9090/currency` |

Nominatim requires `User-Agent: envdash/1.0 (course project)` header and 1 req/sec rate limit.
OpenAQ requires `X-API-Key` header from `OPENAQ_API_KEY` env var.

## Firebase / Firestore Collections

| Collection | Purpose |
|-----------|---------|
| `registrations` | Dashboard configurations |
| `notifications` | Webhook registrations |
| `cache` | Cached API responses with TTL (`expiresAt` field) |
| `api_keys` | API keys (advanced auth task) |

Cache key format: `{type}:{identifier}` e.g. `countries:NO`, `meteo:62.0,10.0`

## Testing Rules (from assignment)

- Handler tests: `httptest` package, mock service interfaces, table-driven
- Client tests: `httptest.NewServer` with stub JSON responses
- Firebase/Firestore: **real Firestore instance** (NOT mocked) for integration tests
- **Never call live external APIs in tests** — always inject stub HTTP clients
- Integration tests use collections prefixed with `test_` and clean up after themselves
- Run with: `go test ./...` (unit) | `go test -tags=integration ./...` (integration)

## Advanced Tasks Status

- [x] HEAD /registrations/ — headers only
- [x] PATCH /registrations/{id} — partial update
- [x] Extra threshold operators >= and <=
- [x] PATCH /notifications/{id} — partial update
- [ ] Auto cache purging (background goroutine + `CACHE_PURGE_INTERVAL_HOURS`)
- [ ] API key auth (POST /auth/, X-API-Key middleware)

## AQI Levels (PM2.5, EPA breakpoints)

| PM2.5 µg/m³ | Level |
|------------|-------|
| 0–12 | Good |
| 12.1–35.4 | Moderate |
| 35.5–55.4 | Unhealthy for Sensitive Groups |
| 55.5–150.4 | Unhealthy |
| 150.5–250.4 | Very Unhealthy |
| 250.5+ | Hazardous |
| -1 (no data) | Unknown |

## Commit Style

Atomic commits. Format: `type: short description`
Types: `feat`, `fix`, `test`, `docs`, `refactor`, `chore`
Example: `feat: add POST /registrations/ handler with Firestore persistence`

## Running Locally

```bash
cp .env.example .env
# Fill in FIREBASE_PROJECT_ID, OPENAQ_API_KEY, GOOGLE_APPLICATION_CREDENTIALS
go run ./cmd/server
```
