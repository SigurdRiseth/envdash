# Air Quality & Environment Dashboard Service

REST web service for PROG2005 — Cloud Technologies. Aggregates live environmental data from five external APIs into configurable dashboards, with persistent state in Firebase Firestore and webhook-based notifications.

**Deployed at:** `http://10.212.171.150:8080` (NTNU network or VPN required)

---

## Table of Contents

- [Live Service — Quick Test](#live-service--quick-test)
- [Features](#features)
- [API Reference](#api-reference)
- [Authentication](#authentication)
- [Setup — Running Locally](#setup--running-locally)
- [Firebase Setup](#firebase-setup)
- [Deployment — OpenStack SkyHigh](#deployment--openstack-skyhigh)
- [Testing](#testing)
- [Caching Strategy](#caching-strategy)
- [AQI Level Classification](#aqi-level-classification)
- [Known Edge Cases & Limitations](#known-edge-cases--limitations)
- [External APIs](#external-apis)
- [Third-Party Libraries](#third-party-libraries)
- [AI Assistance](#ai-assistance)
- [Group Contributions](#group-contributions)

---

## Live Service — Quick Test

The service is deployed at `http://10.212.171.150:8080/envdash/v1` (reachable from the NTNU network or via VPN).

The workflow below demonstrates all core features in order. Replace `BASE` with the deployed URL or `http://localhost:8080/envdash/v1` for local testing.

```bash
BASE="http://10.212.171.150:8080/envdash/v1"

# 1. Check service health (no API key required)
curl "$BASE/status/"

# 2. Register an API key
KEY=$(curl -s -X POST "$BASE/auth/" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","email":"test@example.com"}' | grep -o '"key":"[^"]*"' | cut -d'"' -f4)
echo "API key: $KEY"

# 3. Register a dashboard configuration for Norway
ID=$(curl -s -X POST "$BASE/registrations/" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $KEY" \
  -d '{
    "country": "Norway",
    "isoCode": "NO",
    "features": {
      "temperature": true,
      "precipitation": true,
      "airQuality": true,
      "capital": true,
      "coordinates": true,
      "population": true,
      "area": true,
      "targetCurrencies": ["EUR", "USD", "SEK"]
    }
  }' | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Registration ID: $ID"

# 4. Retrieve the stored configuration
curl -H "X-API-Key: $KEY" "$BASE/registrations/$ID"

# 5. Retrieve a live populated dashboard (fetches all external APIs)
curl -H "X-API-Key: $KEY" "$BASE/dashboards/$ID"

# 6. Register a webhook to be notified on future dashboard retrievals
curl -X POST "$BASE/notifications/" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $KEY" \
  -d "{\"url\":\"https://webhook.site/YOUR-ID\",\"country\":\"NO\",\"event\":\"INVOKE\"}"

# 7. Trigger the webhook by retrieving the dashboard again
curl -H "X-API-Key: $KEY" "$BASE/dashboards/$ID"

# 8. Clean up
curl -X DELETE -H "X-API-Key: $KEY" "$BASE/registrations/$ID"
```

> Use [https://webhook.site](https://webhook.site) to get a free URL for inspecting webhook payloads.

---

## Features

- **Dashboard configurations** — store and manage which environmental metrics to display per country
- **Live data aggregation** — weather forecasts, air quality readings, country metadata, and exchange rates fetched concurrently from five external APIs
- **Webhook notifications** — lifecycle events (REGISTER, CHANGE, DELETE, INVOKE) and threshold alerts (THRESHOLD) delivered via HTTP POST
- **Persistent state** — all configurations and webhook registrations survive service restarts via Firebase Firestore
- **Response caching** — external API responses cached in Firestore with per-API TTLs to minimise outbound requests and respect rate limits
- **Auto cache purging** — expired cache entries evicted on startup and on a configurable schedule

### Advanced tasks implemented

| Task | Status |
|------|--------|
| `HEAD /registrations/` | Done |
| `PATCH /registrations/{id}` (partial update) | Done |
| `PATCH /notifications/{id}` (partial update) | Done |
| Threshold operators `>=` and `<=` | Done |
| Automatic cache purging | Done |
| API key authentication | Done |
| Compound thresholds (single webhook with high + low bound) | Done |

---

## API Reference

Base path: `/envdash/v1`

All endpoints except `/status/` and `/auth/` require an `X-API-Key` header with a valid key (see [Authentication](#authentication)).

### Registrations

| Method | Path | Status Codes | Description |
|--------|------|--------------|-------------|
| `POST` | `/registrations/` | 201, 400 | Register a new dashboard configuration |
| `GET` | `/registrations/` | 200 | List all configurations |
| `HEAD` | `/registrations/` | 200 | Headers only — no body |
| `GET` | `/registrations/{id}` | 200, 404 | Get a specific configuration |
| `PUT` | `/registrations/{id}` | 200, 400, 404 | Replace a configuration (empty body response) |
| `PATCH` | `/registrations/{id}` | 200, 400, 404 | Partially update a configuration |
| `DELETE` | `/registrations/{id}` | 204, 404 | Delete a configuration |

**POST /registrations/ — request body:**

```json
{
  "country": "Norway",
  "isoCode": "NO",
  "features": {
    "temperature": true,
    "precipitation": true,
    "airQuality": true,
    "capital": true,
    "coordinates": true,
    "population": true,
    "area": true,
    "targetCurrencies": ["EUR", "USD", "SEK"]
  }
}
```

Either `country` or `isoCode` (or both) must be provided.

**Response 201 Created:**

```json
{
  "id": "7f3a91bc04e2d158",
  "lastChange": "20250301 09:15"
}
```

**GET /registrations/{id} — response 200 OK:**

```json
{
  "id": "7f3a91bc04e2d158",
  "country": "Norway",
  "isoCode": "NO",
  "features": {
    "temperature": true,
    "precipitation": true,
    "airQuality": true,
    "capital": true,
    "coordinates": true,
    "population": true,
    "area": false,
    "targetCurrencies": ["EUR", "USD", "SEK"]
  },
  "lastChange": "20250301 09:15"
}
```

**PATCH /registrations/{id}** — send only the fields you want to change. Nested `features` fields are also patchable individually:

```json
{ "features": { "airQuality": false } }
```

---

### Dashboards

| Method | Path | Status Codes | Description |
|--------|------|--------------|-------------|
| `GET` | `/dashboards/{id}` | 200, 404 | Retrieve a live populated dashboard |

Fields whose feature flag is `false` are omitted from the response. Fetching a dashboard triggers INVOKE webhooks and evaluates THRESHOLD webhooks against the live values.

**GET /dashboards/{id} — response 200 OK:**

```json
{
  "country": "Norway",
  "isoCode": "NO",
  "features": {
    "temperature": -1.2,
    "precipitation": 0.80,
    "airQuality": { "pm25": 8.4, "pm10": 14.2, "level": "Good" },
    "capital": "Oslo",
    "coordinates": { "latitude": 62.0, "longitude": 10.0 },
    "population": 5379475,
    "area": 323802.0,
    "targetCurrencies": { "EUR": 0.087701, "USD": 0.095184, "SEK": 0.978272 }
  },
  "lastRetrieval": "20250301 18:15"
}
```

Notes:
- `temperature` and `precipitation` are means across all Open-Meteo forecast entries.
- `airQuality` aggregates PM readings across OpenAQ stations within 25 km of the country centroid.
- `lastRetrieval` is always the current server time — it is never cached.
- If one external API fails, that field is omitted while all others are still populated.

---

### Notifications (Webhooks)

| Method | Path | Status Codes | Description |
|--------|------|--------------|-------------|
| `POST` | `/notifications/` | 201, 400 | Register a webhook |
| `GET` | `/notifications/` | 200 | List all webhooks |
| `GET` | `/notifications/{id}` | 200, 404 | Get a specific webhook |
| `PATCH` | `/notifications/{id}` | 200, 400, 404 | Partially update a webhook |
| `DELETE` | `/notifications/{id}` | 204, 404 | Delete a webhook |

**POST /notifications/ — lifecycle event (REGISTER, CHANGE, DELETE, INVOKE):**

```json
{
  "url": "https://webhook.site/your-unique-id",
  "country": "NO",
  "event": "INVOKE"
}
```

**POST /notifications/ — threshold event:**

`threshold` is a list of conditions. All conditions must be satisfied simultaneously for the webhook to fire. Use multiple conditions on the same field to express a range.

```json
{
  "url": "https://webhook.site/your-unique-id",
  "country": "NO",
  "event": "THRESHOLD",
  "threshold": [
    { "field": "pm25", "operator": ">", "value": 35.0 }
  ]
}
```

**Compound threshold — high/low bound on the same field:**

```json
{
  "url": "https://webhook.site/your-unique-id",
  "country": "NO",
  "event": "THRESHOLD",
  "threshold": [
    { "field": "temperature", "operator": ">",  "value": 0 },
    { "field": "temperature", "operator": "<=", "value": 5 }
  ]
}
```

| Field | Values |
|-------|--------|
| `event` | `REGISTER`, `CHANGE`, `DELETE`, `INVOKE`, `THRESHOLD` |
| `threshold` | Array of one or more condition objects; all must be satisfied |
| `threshold[].field` | `pm25`, `pm10`, `temperature`, `precipitation` |
| `threshold[].operator` | `>`, `<`, `>=`, `<=` |
| `country` | ISO 3166-1 alpha-2. Leave empty to match all countries. |

**Response 201 Created:**

```json
{ "id": "kDwvOIcsueiwe" }
```

**Webhook payload — lifecycle event:**

```json
{
  "id": "kDwvOIcsueiwe",
  "country": "NO",
  "event": "INVOKE",
  "time": "20250301 14:22"
}
```

**Webhook payload — THRESHOLD event:**

```json
{
  "id": "mP9xTqwRklzc",
  "country": "NO",
  "event": "THRESHOLD",
  "time": "20250301 14:22",
  "details": {
    "conditions": [
      { "field": "temperature", "operator": ">",  "threshold": 0, "measuredValue": 2.17 },
      { "field": "temperature", "operator": "<=", "threshold": 5, "measuredValue": 2.17 }
    ]
  }
}
```

Webhooks are delivered asynchronously with one retry after 2 seconds on failure.

---

### Status

| Method | Path | Status Codes | Description |
|--------|------|--------------|-------------|
| `GET` | `/status/` | 200, 500 | Service health and upstream API status |

No API key required. Returns 500 if Firebase is unreachable.

**Response 200 OK:**

```json
{
  "countries_api":   200,
  "meteo_api":       200,
  "openaq_api":      200,
  "nominatim_api":   200,
  "currency_api":    200,
  "notification_db": 200,
  "webhooks":        4,
  "version":         "v1",
  "uptime":          3612
}
```

---

## Authentication

All endpoints except `GET /status/` and `POST /auth/` require the `X-API-Key` header.

```
X-API-Key: sk-envdash-a3f9c2b1e847d056...
```

| Condition | Response |
|-----------|----------|
| Header missing | `401 Unauthorized` |
| Key unknown or revoked | `403 Forbidden` |
| Key valid | Request proceeds normally |

**Register a key — POST /auth/:**

```json
// Request
{ "name": "my-client", "email": "user@example.com" }

// Response 201 Created
{ "key": "sk-envdash-a3f9c2b1e847d056", "createdAt": "20250301 09:15" }
```

**Revoke a key — DELETE /auth/{key}:** returns `204 No Content`.

Keys are stored in Firestore and persist across service restarts.

---

## Setup — Running Locally

**Prerequisites:** Go 1.21+, a Firebase project with Firestore enabled, an OpenAQ API key.

```bash
# 1. Clone the repository
git clone <repo-url>
cd envdash

# 2. Copy and fill in environment variables
cp .env.example .env
# Edit .env — at minimum set FIREBASE_PROJECT_ID, OPENAQ_API_KEY,
# and one of GOOGLE_APPLICATION_CREDENTIALS or FIREBASE_CREDENTIALS_JSON

# 3. Run
set -a && source .env && set +a && go run ./cmd/server

# Service is available at http://localhost:8080/envdash/v1
```

### With Docker

```bash
docker build -t envdash .
docker run -p 8080:8080 --env-file .env envdash
```

### With Docker Compose

```bash
docker compose up --build
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `FIREBASE_PROJECT_ID` | Yes | — | Firebase project ID |
| `OPENAQ_API_KEY` | Yes | — | OpenAQ API key (free at [explore.openaq.org](https://explore.openaq.org)) |
| `GOOGLE_APPLICATION_CREDENTIALS` | Yes* | — | Path to Firebase service account JSON file |
| `FIREBASE_CREDENTIALS_JSON` | Yes* | — | Inline Firebase credentials JSON (alternative to file path) |
| `SERVER_PORT` | No | `8080` | HTTP listen port |
| `COUNTRIES_API_URL` | No | `http://129.241.150.113:8080/v3.1` | REST Countries API base URL |
| `METEO_API_URL` | No | `https://api.open-meteo.com/v1` | Open-Meteo base URL |
| `OPENAQ_API_URL` | No | `https://api.openaq.org/v3` | OpenAQ base URL |
| `NOMINATIM_API_URL` | No | `https://nominatim.openstreetmap.org` | Nominatim base URL |
| `CURRENCY_API_URL` | No | `http://129.241.150.113:9090/currency` | Currency API base URL |
| `CACHE_PURGE_INTERVAL_HOURS` | No | `1` | How often to evict expired cache entries (hours) |
| `CACHE_TTL_HOURS` | No | — | Override all per-API cache TTLs with a single value. When set, replaces the individual defaults for all five APIs. Omit to use per-API defaults (see [Caching Strategy](#caching-strategy)). |

*Exactly one of `GOOGLE_APPLICATION_CREDENTIALS` or `FIREBASE_CREDENTIALS_JSON` is required.

---

## Firebase Setup

1. Go to [console.firebase.google.com](https://console.firebase.google.com) and create a project.
2. Enable **Firestore Database** — choose **Native mode** (not Datastore mode).
3. Go to **Project Settings → Service Accounts → Generate new private key**.
4. Download the JSON file and either:
   - Set `GOOGLE_APPLICATION_CREDENTIALS=/absolute/path/to/key.json`, or
   - Paste the JSON content into `FIREBASE_CREDENTIALS_JSON` (useful for Docker or CI without mounted volumes).

The service creates and manages four Firestore collections automatically:

| Collection | Purpose |
|-----------|---------|
| `registrations` | Dashboard configurations |
| `notifications` | Webhook registrations |
| `cache` | Cached API responses with TTL |
| `api_keys` | Registered API keys |

---

## Deployment — OpenStack SkyHigh

The service is deployed on an Ubuntu 22.04 VM at `10.212.171.150`, accessible from the NTNU network or VPN.

**Live base URL:** `http://10.212.171.150:8080/envdash/v1`

### SSH access

```bash
ssh -i ~/.ssh/envdash ubuntu@10.212.171.150
```

### Initial setup (once, on a fresh VM)

```bash
# On the VM — clone, configure, and start:
bash deployment/setup.sh
nano /home/ubuntu/envdash/.env          # fill in credentials
cd /home/ubuntu/envdash && docker compose up -d
```

### Deploy an update

```bash
# From the VM:
cd /home/ubuntu/envdash && ./deployment/deploy.sh

# Or from your local machine in one line:
ssh -i ~/.ssh/envdash ubuntu@10.212.171.150 'cd /home/ubuntu/envdash && ./deployment/deploy.sh'
```

### Check logs

```bash
docker compose logs -f
```

---

## Testing

### Unit tests (no external services required)

```bash
go test ./...

# With coverage
go test ./... -cover

# Verbose output
go test ./... -v
```

Handler tests use mock service implementations — no live APIs or Firebase needed.
Client tests use `httptest.NewServer` with stub JSON responses.
Service tests use in-memory stub repositories.
Webhook dispatcher tests use `httptest.NewServer` to capture payloads and verify retry behaviour.

### Current test coverage

| Package | Coverage |
|---------|----------|
| `internal/models` | ~96% |
| `internal/clients` | ~93% |
| `internal/handlers` | ~89% |
| `internal/services` | ~81% |
| `internal/webhook` | ~92% |

### Integration tests (real Firestore)

```bash
go test -tags=integration ./internal/firebase/...
```

Requires `FIREBASE_PROJECT_ID` and valid credentials. Tests use `test_`-prefixed collections and clean up after themselves.

---

## Caching Strategy

External API responses are cached in Firestore to minimise outbound requests and respect rate limits.

| API | Cache TTL | Rationale | Cache key |
|-----|-----------|-----------|-----------|
| REST Countries | 24 hours | Country metadata (population, area, capital) changes rarely | `countries:{ISO}` |
| Nominatim | 24 hours | Geocoding results are stable; also rate-limited to 1 req/sec | `nominatim:{ISO}` |
| Open-Meteo | 3 hours | Forecasts update a few times per day; short TTL keeps data reasonably fresh | `meteo:{lat},{lon}` |
| OpenAQ | 1 hour | Air quality readings change more frequently; 1 hour balances freshness vs. API load | `openaq:{lat},{lon}` |
| Currency | 1 hour | Exchange rates fluctuate during trading hours; 1 hour is a common convention | `currency:{base}` |

Expired entries are evicted on service startup and every `CACHE_PURGE_INTERVAL_HOURS` hours thereafter (default: 1 hour).

Use `CACHE_TTL_HOURS` to override all TTLs with a uniform value — useful for testing or temporarily reducing API load.

---

## AQI Level Classification

Air quality level is determined from PM2.5 concentration (µg/m³) using EPA AQI breakpoints:

| PM2.5 (µg/m³) | Level |
|---------------|-------|
| 0–12 | Good |
| 12.1–35.4 | Moderate |
| 35.5–55.4 | Unhealthy for Sensitive Groups |
| 55.5–150.4 | Unhealthy |
| 150.5–250.4 | Very Unhealthy |
| 250.5+ | Hazardous |
| No data (−1) | Unknown |

The classification is based on the PM2.5 mean across all OpenAQ monitoring stations within 25 km of the country centroid. PM10 is reported separately alongside the PM2.5-derived level.

---

## Known Edge Cases & Limitations

- **OpenAQ radius is 25 km, not 50 km:** The assignment specification states 50 km, but the OpenAQ v3 API enforces a hard limit of 25,000 m per request. If no monitoring stations are found within 25 km, air quality is reported as `{"pm25": -1, "pm10": -1, "level": "Unknown"}`.
- **Threshold registered for a disabled field:** A THRESHOLD webhook can be registered for any field (e.g. `pm25`) regardless of whether that field is enabled in the dashboard configuration (`airQuality: false`). The webhook is stored successfully but will never fire, because the field value is never populated during dashboard retrieval. This is intentional and matches the specification.
- **`lastRetrieval` is never cached:** The timestamp in a dashboard response always reflects the actual time of the request, not when the underlying data was last fetched from external APIs.
- **Multiple capital cities:** When a country has more than one listed capital (e.g. South Africa), the first value returned by the REST Countries API is used.
- **Partial upstream failure:** If one external API is unavailable during dashboard retrieval, that field is omitted from the response while all other fields are still populated. The service never fails completely due to a single upstream outage.
- **Wildcard webhooks:** A webhook registered with an empty `country` field matches lifecycle and threshold events for all countries.
- **Nominatim rate limit:** Nominatim enforces 1 request/second per client. The client implements a token-bucket rate limiter. The 24-hour cache means this limit is rarely approached in practice.
- **Threshold condition list — no enforced upper bound:** The `threshold` array accepts any number of conditions. All conditions must be satisfied simultaneously. There is no server-enforced maximum, though in practice one or two conditions cover all useful cases (single bound or high/low range).
- **Firebase required for all write operations:** The status endpoint remains available when Firebase is down, but all other endpoints (registrations, dashboards, notifications, auth) will return 500. The `/status/` response body will show `"notification_db": 503` in this case.

---

## External APIs

| Service | Default endpoint | Notes |
|---------|-----------------|-------|
| REST Countries | `http://129.241.150.113:8080/v3.1` | Course-hosted instance |
| Open-Meteo | `https://api.open-meteo.com/v1` | No key required |
| OpenAQ v3 | `https://api.openaq.org/v3` | Requires `X-API-Key` header |
| Nominatim (OSM) | `https://nominatim.openstreetmap.org` | Requires `User-Agent` header; 1 req/sec limit |
| Currency | `http://129.241.150.113:9090/currency` | Course-hosted instance |

All base URLs are configurable via environment variables.

---

## Third-Party Libraries

Only Go's standard library and the Firebase/Firestore Admin SDK are used:

| Library | Purpose |
|---------|---------|
| `firebase.google.com/go/v4` | Firebase Admin SDK |
| `cloud.google.com/go/firestore` | Firestore client |
| `google.golang.org/api` | Google API support (indirect, required by Firebase) |

No HTTP routers, ORMs, or web frameworks were added. The standard `net/http` package handles all routing and request handling.

---

## AI Assistance

AI-based tools (specifically Claude Code by Anthropic) were used as a supporting aid during the development of this project.

**Where and why:** AI assistance was most active during the initial architecture phase (e.g., structuring handler–service–repository layers and discussing interface boundaries) and during repetitive implementation work (test stubs, Firestore query patterns, handler boilerplate). It was also used when writing documentation throughout the code.

**When it was useful:** Boilerplate-heavy tasks benefited most — generating table-driven test skeletons, writing mock implementations of service interfaces, and structuring Firestore read/write patterns. It saved significant time on work that follows a predictable pattern once the design is settled.

**When it was not useful:** Anything involving the specific behaviour of external APIs required reading official documentation directly. The OpenAQ v3 radius hard limit (25,000 m), Nominatim's `User-Agent` requirement and rate limit, and the exact Firestore Admin SDK transaction API were all cases where AI-generated suggestions were either wrong or outdated and had to be corrected against the real documentation.

**How we handled it:** All AI-generated code was read, understood, tested, and reviewed before merging. Nothing was merged blind. The architecture decisions, data model, and API contract were our own – AI was used as a fast-typing assistant, not a decision-maker.

---

## Group Contributions

### Members

| Member | Primary areas |
|--------|--------------|
| Sigurd Riseth | Architecture and system design; all endpoint handlers and service logic; Firebase repository layer; webhook dispatcher; Docker and deployment setup; integration tests |
| Theodor Utvik | Code quality review and refactoring across all packages; unit test coverage; endpoint development and debugging; documentation; PR review and merging |

### How we organized

**Role distribution:** We split work around a driver/reviewer model. Sigurd took primary ownership of the architecture — designing the layered structure, implementing all endpoints end-to-end, and setting up Firebase and Docker. Theodor contributed actively to development: assisting with endpoint implementation and debugging, driving code quality through refactoring, writing and improving unit tests, and ensuring the code followed our agreed conventions. Every significant change was reviewed by both members before merging.

**Workflow:** All changes were developed on feature branches and merged via pull requests on GitLab. We required at least one review approval before merging to `main`. The CI pipeline (GitHub Actions, mirrored to GitLab) ran `go test ./...` on every push, so `main` was always green.

### Technical decisions

**Commit style:** We followed a `type: short description` convention (`feat`, `fix`, `test`, `docs`, `refactor`, `chore`) with atomic commits. This made the git history readable and made it easy to trace why each change was made.

**Architecture:** We chose a strict layered architecture (handlers → services → repositories/clients) with interface-based dependencies throughout. This kept handlers thin and made unit testing straightforward — each layer can be tested in isolation with stubs. We did not introduce any external HTTP routing libraries; Go's `net/http` ServeMux was sufficient.

**Caching in Firestore:** Rather than using an in-memory cache (which would not survive restarts), we stored cache entries as Firestore documents with an `expiresAt` timestamp. A background goroutine purges expired entries periodically. This keeps the cache consistent across deployments and requires no additional infrastructure.

**Concurrency:** Dashboard retrieval fans out up to four independent API calls (meteo, openaq, currency, and countries) concurrently using goroutines and `sync.WaitGroup`. A single countries API call is made first since its result (coordinates, base currency) is needed by the other three — this minimises total latency while avoiding redundant requests.

**Error handling:** All errors are returned as `{"error": "message"}` JSON with the appropriate HTTP status code. We chose never to return plain text error bodies. Internal errors are logged with enough context to diagnose failures without exposing implementation details to clients.

**No hardcoded configuration:** API keys, base URLs, Firebase credentials, and port numbers are read exclusively from environment variables via a typed `Config` struct. Defaults are provided where the specification defines them.
