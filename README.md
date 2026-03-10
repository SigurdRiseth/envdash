# Air Quality & Environment Dashboard Service

REST web service for PROG2005 — Cloud Technologies. Aggregates live environmental data from multiple open APIs into configurable dashboards, with persistent state in Firebase Firestore and webhook-based notifications.

## Features

- **Dashboard configurations** — store and manage which environmental metrics to display per country
- **Live data aggregation** — weather forecasts, air quality readings, country metadata, and exchange rates fetched concurrently from five external APIs
- **Webhook notifications** — lifecycle events (REGISTER, CHANGE, DELETE, INVOKE) and threshold alerts (THRESHOLD) delivered via HTTP POST
- **Persistent state** — all configurations and webhook registrations survive service restarts via Firebase Firestore
- **Response caching** — external API responses are cached in Firestore with configurable TTLs, respecting rate limits and reducing outbound requests
- **Auto cache purging** — expired cache entries are evicted on startup and on a configurable interval

### Advanced tasks implemented

| Task | Status |
|------|--------|
| `HEAD /registrations/` | Done |
| `PATCH /registrations/{id}` (partial update) | Done |
| `PATCH /notifications/{id}` (partial update) | Done |
| Threshold operators `>=` and `<=` | Done |
| Automatic cache purging | Done |
| API key authentication | Done |

---

## API Endpoints

Base path: `/envdash/v1`

### Registrations

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/registrations/` | Register a new dashboard configuration |
| `GET` | `/registrations/` | List all configurations |
| `HEAD` | `/registrations/` | Headers only (no body) |
| `GET` | `/registrations/{id}` | Get a specific configuration |
| `PUT` | `/registrations/{id}` | Replace a configuration |
| `PATCH` | `/registrations/{id}` | Partially update a configuration |
| `DELETE` | `/registrations/{id}` | Delete a configuration |

### Dashboards

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/dashboards/{id}` | Retrieve a populated dashboard |

### Notifications (Webhooks)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/notifications/` | Register a webhook |
| `GET` | `/notifications/` | List all webhooks |
| `GET` | `/notifications/{id}` | Get a specific webhook |
| `PATCH` | `/notifications/{id}` | Partially update a webhook |
| `DELETE` | `/notifications/{id}` | Delete a webhook |

### Status

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/status/` | Service health and upstream status |

### Auth (API Keys)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/auth/` | Register a new API key — returns `{"apiKey": "sk-envdash-..."}` |
| `DELETE` | `/auth/{key}` | Revoke an API key |

All endpoints except `/status/` and `/auth/` require an `X-API-Key` header containing a valid key.
Requests without a key return `401 Unauthorized`; requests with an unknown key return `403 Forbidden`.



---

## Request / Response Examples

### POST /envdash/v1/registrations/

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

Response `201 Created`:

```json
{
  "id": "7f3a91bc04e2d158",
  "lastChange": "20250301 09:15"
}
```

### GET /envdash/v1/dashboards/{id}

Response `200 OK`:

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

### POST /envdash/v1/notifications/ — lifecycle event

```json
{
  "url": "https://webhook.site/your-unique-id",
  "country": "NO",
  "event": "INVOKE"
}
```

### POST /envdash/v1/notifications/ — threshold event

```json
{
  "url": "https://webhook.site/your-unique-id",
  "country": "NO",
  "event": "THRESHOLD",
  "threshold": {
    "field": "pm25",
    "operator": ">",
    "value": 35.0
  }
}
```

Supported operators: `>`, `<`, `>=`, `<=`
Supported fields: `pm25`, `pm10`, `temperature`, `precipitation`

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `FIREBASE_PROJECT_ID` | Yes | — | Firebase project ID |
| `OPENAQ_API_KEY` | Yes | — | OpenAQ API key (free at explore.openaq.org) |
| `GOOGLE_APPLICATION_CREDENTIALS` | Yes* | — | Path to Firebase service account JSON file |
| `FIREBASE_CREDENTIALS_JSON` | Yes* | — | Inline Firebase credentials JSON (alternative to above) |
| `SERVER_PORT` | No | `8080` | HTTP listen port |
| `COUNTRIES_API_URL` | No | `http://129.241.150.113:8080/v3.1` | REST Countries API base URL |
| `METEO_API_URL` | No | `https://api.open-meteo.com/v1` | Open-Meteo base URL |
| `OPENAQ_API_URL` | No | `https://api.openaq.org/v3` | OpenAQ base URL |
| `NOMINATIM_API_URL` | No | `https://nominatim.openstreetmap.org` | Nominatim base URL |
| `CURRENCY_API_URL` | No | `http://129.241.150.113:9090/currency` | Currency API base URL |
| `CACHE_PURGE_INTERVAL_HOURS` | No | `1` | Evict expired cache entries every N hours |
| `CACHE_TTL_HOURS` | No | — | Override all per-type cache TTLs (hours) |

*One of `GOOGLE_APPLICATION_CREDENTIALS` or `FIREBASE_CREDENTIALS_JSON` is required.

---

## Running Locally

```bash
# 1. Copy and fill in environment variables
cp .env.example .env
# Edit .env with your Firebase project ID, credentials, and OpenAQ key

# 2. Run the server
go run ./cmd/server

# Service is available at http://localhost:8080
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

### Integration tests (real Firestore)

```bash
go test -tags=integration ./internal/firebase/...
```

Requires `FIREBASE_PROJECT_ID` and valid credentials. Tests use `test_` prefixed collections and clean up after themselves.

---

## Firebase Setup

1. Go to console.firebase.google.com and create a project
2. Enable **Firestore Database** (Native mode, not Datastore mode)
3. Go to **Project Settings → Service Accounts → Generate new private key**
4. Download the JSON file and either:
   - Set `GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json`, or
   - Copy the JSON content into `FIREBASE_CREDENTIALS_JSON` (useful for Docker without mounted volumes)

---

## Testing

```bash
# Unit tests (no external services required)
go test ./...

# With coverage report
go test ./... -cover

# Verbose output
go test ./... -v
```

### Test coverage targets

| Package | Coverage |
|---------|----------|
| `internal/models` | ~100% |
| `internal/handlers` | ~88% |
| `internal/clients` | ~85% |
| `internal/services` | ~31% |

Handler tests use mock service implementations — no live APIs or Firebase required.
Client tests use `httptest.NewServer` to serve stub JSON responses.
Service tests use stub repository implementations.

---

## Caching Strategy

External API responses are cached in Firestore to minimise outbound requests and respect rate limits.

| API | Cache TTL | Cache key format |
|-----|-----------|-----------------|
| REST Countries | 24 hours | `countries:{ISO}` |
| Open-Meteo | 3 hours | `meteo:{lat},{lon}` |
| OpenAQ | 1 hour | `openaq:{lat},{lon}` |
| Nominatim | 24 hours | `nominatim:{ISO}` |
| Currency | 1 hour | `currency:{base}` |

Expired entries are deleted on service startup and every `CACHE_PURGE_INTERVAL_HOURS` hours thereafter.

---

## AQI Level Classification

Air quality level is classified based on PM2.5 concentration using EPA AQI breakpoints:

| PM2.5 (µg/m³) | Level |
|---------------|-------|
| 0–12 | Good |
| 12.1–35.4 | Moderate |
| 35.5–55.4 | Unhealthy for Sensitive Groups |
| 55.5–150.4 | Unhealthy |
| 150.5–250.4 | Very Unhealthy |
| 250.5+ | Hazardous |
| No data | Unknown |

---

## Known Edge Cases

- **Threshold on disabled field:** If a THRESHOLD webhook is registered for a field (e.g. `pm25`) that is not enabled in the dashboard configuration (`airQuality: false`), the webhook is stored successfully but will never fire. This matches the assignment specification.
- **No OpenAQ stations nearby:** If no monitoring stations are found within 50 km of the country centroid, air quality is reported as `{"pm25": -1, "pm10": -1, "level": "Unknown"}`.
- **`lastRetrieval` is never cached:** The timestamp in a dashboard response always reflects the actual request time, not when the underlying data was fetched from external APIs.
- **Multiple capital cities:** When a country has more than one capital (e.g. South Africa), the first value returned by the Countries API is used.
- **Partial upstream failure:** If one external API is unavailable during a dashboard retrieval, that field is omitted (`null`) from the response while other fields remain populated.
- **Webhook country is a wildcard:** A webhook with an empty `country` field matches events for all countries.
- **Nominatim rate limit:** Nominatim enforces 1 request/second. The client implements a token-bucket throttle. The 24-hour cache means this limit is rarely reached.

---

## External APIs

| Service | Endpoint |
|---------|----------|
| REST Countries | `http://129.241.150.113:8080/v3.1` |
| Open-Meteo | `https://api.open-meteo.com/v1` |
| OpenAQ v3 | `https://api.openaq.org/v3` |
| Nominatim (OSM) | `https://nominatim.openstreetmap.org` |
| Currency | `http://129.241.150.113:9090/currency` |

---

## Group Contributions

| Member | Primary responsibilities |
|--------|--------------------------|
| Sigurd Riseth | Architecture, all endpoints, tests, Firebase integration |
| *(TBD)* | *(to be filled in)* |

---

## Third-Party Libraries

Only Go's standard library and the Firebase/Firestore Admin SDK are used:

| Library | Purpose |
|---------|---------|
| `firebase.google.com/go/v4` | Firebase Admin SDK |
| `cloud.google.com/go/firestore` | Firestore client |
| `google.golang.org/api` | Google API support (indirect, required by Firebase) |

No additional HTTP routers, ORMs, or frameworks were added. The standard `net/http` package is sufficient for the scope of this assignment.
