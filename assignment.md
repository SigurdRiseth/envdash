# Group Assignment 2 — Air Quality & Environment Dashboard Service

**Course:** PROG2005 — Cloud Technologies  
**Group size:** 3 students (see [Group Size](#group-size--workload))  
**Deployment:** OpenStack (SkyHigh)  
**Language:** Go (Golang)  
<!--**Submission deadline:** See course wiki page-->

---

> **Note:** This assignment builds directly on Assignment 1. You will extend your REST service skills with persistent state management (Firebase), webhook-based notifications, and IaaS deployment using Docker on OpenStack.

---

## Table of Contents

[TOC]

---

## Overview

In this group assignment, you will develop a REST web application in Go that provides clients with configurable environment and air quality dashboards. Dashboards aggregate live data from multiple open APIs and are populated on demand. Configurations and webhook registrations are stored persistently (i.e., they survive service restarts) using Firebase Firestore. The service also includes a webhook-based notification system triggered by lifecycle events and data threshold crossings.

**The key new skills compared to Assignment 1 are:**

- Persistent state management using Firebase Firestore
- Webhook-based event notifications (lifecycle and threshold-driven)
- Dockerized deployment to an IaaS platform (OpenStack SkyHigh)
- Responsible and efficient use of third-party APIs (caching, minimal requests)

---

## External APIs

Your service will consume the following external APIs:

| Service | Endpoint / Documentation | Notes |
|---|---|---|
| REST Countries API | API: http://129.241.150.113:8080/v3.1<br>Docs: http://129.241.150.113:8080 | Course-hosted instance |
| Open-Meteo | Docs: https://open-meteo.com/en/features | Externally hosted — use responsibly |
| OpenAQ v3 | API: https://api.openaq.org/v3<br>Docs: https://docs.openaq.org/ | Free key via explore.openaq.org. Raw µg/m³ |
| Nominatim (OSM) | Docs: https://nominatim.org/release-docs/develop/api/Overview/ | No key required. Must set `User-Agent` header. 1 req/sec limit. |
| Currency API | API: http://129.241.150.113:9090/currency/<br>Docs: http://129.241.150.113:9090/ | Course-hosted instance |

> **Note:** When integrating external APIs, always find the most efficient endpoint. Minimise the total number of outbound requests — this is assessed. For testing, stub all external services (do **NOT** call live APIs in your tests).

---

## API Specification

Your service exposes four root endpoint paths (plus an optional fifth — see [Authentication](#endpoint-authentication-advanced-task)):

```
/envdash/v1/registrations/
/envdash/v1/dashboards/
/envdash/v1/notifications/
/envdash/v1/status/
```

**Convention:** `{value}` denotes a mandatory path parameter.

---

## Endpoint: Registrations

Manages the lifecycle of dashboard configurations. Each configuration specifies a country and which environmental features to display. Configurations are stored persistently in Firebase.

| Method | Path | Description | Status Codes |
|---|---|---|---|
| `POST` | `/registrations/` | Register a new dashboard configuration | 201 Created / 400 Bad Request |
| `GET` | `/registrations/{id}` | Retrieve a specific configuration by ID | 200 OK / 404 Not Found |
| `GET` | `/registrations/` | Retrieve all stored configurations | 200 OK |
| `PUT` | `/registrations/{id}` | Replace a specific configuration (updates timestamp) | 200 OK / 404 Not Found |
| `DELETE` | `/registrations/{id}` | Delete a specific configuration | 204 No Content / 404 Not Found |

---

### POST /registrations/ — Register new configuration

Store a new dashboard configuration. The server generates a unique ID and `lastChange` timestamp.

**Request body:**

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

| Field | Description |
|---|---|
| `country` | Country name. Can be omitted if `isoCode` is provided. |
| `isoCode` | ISO 3166-1 alpha-2 code. Can be omitted if `country` is provided. |
| `temperature` | Mean forecasted temperature (°C) |
| `precipitation` | Mean forecasted precipitation (mm) |
| `airQuality` | Mean PM2.5 reading across nearby monitoring stations (µg/m³) |
| `capital` | Capital city name |
| `coordinates` | Country centroid latitude/longitude |
| `population` | Country population |
| `area` | Land area (km²) |
| `targetCurrencies` | Exchange rates from the country's base currency to each listed currency |

**Response body (201 Created):**

```json
{
    "id": "7f3a91bc04e2d158",
    "lastChange": "20250301 09:15"
}
```

---

### GET /registrations/{id} — Retrieve specific configuration

**Response body (200 OK):**

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

---

### GET /registrations/ — Retrieve all configurations

Returns a JSON array of all stored configurations, each in the format shown above.

> ⭐ **Advanced Task:** Implement the `HEAD` method — return headers only, no body.

---

### PUT /registrations/{id} — Replace configuration

Replace the full configuration for the given ID. The server updates `lastChange`. The ID and timestamp must **not** be included in the request body.

> ⭐ **Advanced Task:** Implement `PATCH` — support partial updates where only supplied fields are changed.

**Response:** `200 OK` with empty body.

---

### DELETE /registrations/{id} — Delete configuration

Deletes the configuration. Returns `204 No Content` with an empty body on success.

---

## Endpoint: Dashboards

Retrieves a populated dashboard by fetching live data from all relevant external APIs based on the stored configuration. Only individual dashboard retrieval is supported — no bulk retrieval — to minimise API load.

| Method | Path | Description | Status Codes |
|---|---|---|---|
| `GET` | `/dashboards/{id}` | Retrieve a populated dashboard for the given configuration ID | 200 OK / 404 Not Found |

---

### GET /dashboards/{id} — Retrieve populated dashboard

**Response body (200 OK):**

```json
{
   "country": "Norway",
   "isoCode": "NO",
   "features": {
      "temperature": -1.2,
      "precipitation": 0.80,
      "airQuality": {
         "pm25": 8.4,
         "pm10": 14.2,
         "level": "Good"
      },
      "capital": "Oslo",
      "coordinates": {
         "latitude": 62.0,
         "longitude": 10.0
      },
      "population": 5379475,
      "area": 323802.0,
      "targetCurrencies": {
         "EUR": 0.087701,
         "USD": 0.095184,
         "SEK": 0.978272
      }
   },
   "lastRetrieval": "20250301 18:15"
}
```

> **Notes:**
> - Only populate fields whose feature flag is `true` in the configuration. Omit or set to `null` any field whose flag is `false`.
> - `temperature` and `precipitation` are mean values across all forecast entries returned by Open-Meteo.
> - Where multiple capital cities exist for a country, use the first value.
> - `coordinates` are the country centroid from the REST Countries API.
> - For `airQuality`, query OpenAQ v3 for monitoring stations within 50 km of the country coordinates and aggregate readings across all stations. If no stations are found within range, return -1 values and indicate the level as "unknown".
> - The `level` evaluation should be based on the [EPA AQI Breakpoints](https://aqs.epa.gov/aqsweb/documents/codetables/aqi_breakpoints.html). The calculation can be based on situational readings.
> - `lastRetrieval` is the server time at the moment of the request, not a cached value.

---

## Endpoint: Notifications (Webhooks)

Users can register webhooks triggered by two categories of event: **lifecycle events** (configuration changes and dashboard retrievals) and **threshold alerts** (live measured values crossing user-defined limits). All registrations are stored persistently in Firebase and survive service restarts.

| Method | Path | Description | Status Codes |
|---|---|---|---|
| `POST` | `/notifications/` | Register a new webhook (lifecycle or threshold) | 201 Created / 400 Bad Request |
| `GET` | `/notifications/{id}` | Retrieve a specific webhook registration | 200 OK / 404 Not Found |
| `GET` | `/notifications/` | List all registered webhooks | 200 OK |
| `DELETE` | `/notifications/{id}` | Delete a webhook registration | 204 No Content / 404 Not Found |

---

### Supported Event Types

| Event | Triggered when… |
|---|---|
| `REGISTER` | A new dashboard configuration is registered (`POST /registrations/`) |
| `CHANGE` | A configuration is updated (`PUT` or `PATCH /registrations/{id}`) |
| `DELETE` | A configuration is deleted (`DELETE /registrations/{id}`) |
| `INVOKE` | A populated dashboard is retrieved (`GET /dashboards/{id}`) |
| `THRESHOLD` | A live measured value crosses a user-defined threshold during dashboard retrieval |

> **Note on THRESHOLD:** This is a data-driven event. Unlike the lifecycle events above, it fires based on values returned during a `GET /dashboards/{id}` call — not on configuration changes. Multiple thresholds can be registered for the same country and field. Each crossing fires a separate webhook invocation.

---

### POST /notifications/ — Register webhook

**Request body — lifecycle events (REGISTER, CHANGE, DELETE, INVOKE):**

```json
{
   "url":     "https://webhook.site/your-unique-id",
   "country": "NO",
   "event":   "INVOKE"
}
```

**Request body — THRESHOLD event (extended fields required):**

```json
{
   "url":     "https://webhook.site/your-unique-id",
   "country": "NO",
   "event":   "THRESHOLD",
   "threshold": {
      "field":    "pm25",
      "operator": ">",
      "value":    35.0
   }
}
```

| Field | Description |
|---|---|
| `url` | URL to POST to when the event fires |
| `country` | ISO 3166-1 alpha-2 code filter. Omit or leave empty to match all countries. |
| `event` | One of: `REGISTER`, `CHANGE`, `DELETE`, `INVOKE`, `THRESHOLD` |
| `threshold.field` | Field to monitor: `pm25` \| `pm10` \| `temperature` \| `precipitation` |
| `threshold.operator` | Comparison operator: `>` or `<` |
| `threshold.value` | Numeric threshold value |

> **Note:** If the threshold `field` is not enabled in the corresponding dashboard configuration, the registration is accepted but will never fire. Document this edge case in your README.

**Response body (201 Created):**

```json
{ "id": "kDwvOIcsueiwe" }
```

---

### Webhook Invocation Payload

When any event fires, the service POSTs to the registered URL.

**Lifecycle events (REGISTER, CHANGE, DELETE, INVOKE):**

```json
{
   "id":      "kDwvOIcsueiwe",
   "country": "NO",
   "event":   "INVOKE",
   "time":    "20250301 14:22"
}
```

**THRESHOLD events** (includes the value that caused the trigger):

```json
{
   "id":      "mP9xTqwRklzc",
   "country": "NO",
   "event":   "THRESHOLD",
   "time":    "20250301 14:22",
   "details": {
      "field":         "pm25",
      "operator":      ">",
      "threshold":     35.0,
      "measuredValue": 47.3
   }
}
```

> **Note:** Where multiple webhooks match a single event, send one POST per matching webhook. For initial testing, use https://webhook.site to inspect invocations without needing to run a receiver service.

> ⭐ **Advanced Task:** Support additional operators beyond `>` and `<` (e.g., `>=`, `<=`). Support compound thresholds where both a high and low bound can be registered in a single webhook. Document any extensions clearly.

---

## Endpoint: Authentication (Advanced Task)

> ⭐ **Advanced Task:** Implement a simple API key system so that clients must register before using your service. This adds a realistic security layer and is excellent practice for understanding middleware patterns in Go.

| Method | Path | Description | Status Codes |
|---|---|---|---|
| `POST` | `/auth/` | Register a new client and receive an API key | 201 Created / 400 Bad Request |
| `DELETE` | `/auth/{key}` | Revoke an API key | 204 No Content / 404 Not Found |

### POST /auth/ — Register and obtain API key

**Request body:**

```json
{
   "name":  "my-client-app",
   "email": "user@example.com"
}
```

**Response body (201 Created):**

```json
{
   "key":       "sk-envdash-a3f9c2b1e847d056",
   "createdAt": "20250301 09:15"
}
```

Once issued, the client must include the key in the `X-API-Key` header on every subsequent request:

```
GET /envdash/v1/dashboards/7f3a91bc04e2d158 HTTP/1.1
X-API-Key: sk-envdash-a3f9c2b1e847d056
```

> **Implementation notes:**
> - Implement key validation as a middleware function wrapping all route handlers except `POST /auth/` itself.
> - Return `401 Unauthorized` for missing keys and `403 Forbidden` for invalid or revoked keys.
> - Store keys in Firebase so they persist across restarts.
> - `/envdash/v1/status/` should remain publicly accessible without a key.

---

## Endpoint: Status

Provides a health overview of all services this application depends on, along with operational metadata.

| Method | Path | Description | Status Codes |
|---|---|---|---|
| `GET` | `/status/` | Returns availability of upstream services and service metadata | 200 OK / 500 if critical dependency is down |

### GET /status/ — Service health

**Response body (200 OK):**

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

> **Note:** Return the actual HTTP status code obtained from a lightweight probe of each upstream service (e.g., a `HEAD` request or their documented health endpoint). If a service is unreachable, report its last known code or `503`.

---

## Additional Requirements

### Testing

- All endpoint handlers must be tested using Go's `httptest` package.
- Stub all upstream external services — **tests must not call live APIs**.
- Do **not** mock Firebase/Firestore — use a real Firestore instance in integration tests.
- Maximise test coverage as reported by `go test -cover`.
- Include at least one table-driven test for each endpoint.

### Caching

- Use Firebase to cache responses from upstream APIs to minimise repeated outbound calls.
- A reasonable cache strategy is required — document your TTL choices and rationale.

> ⭐ **Advanced Task:** Implement automatic cache purging: cached entries older than a configurable number of hours are evicted. Document the default TTL and how to configure it via environment variable.

### Error Handling

- Return meaningful JSON error bodies (not plain text) for all 4xx and 5xx responses.
- Handle partial upstream failures gracefully — if one API is unavailable, return what you can and indicate which fields could not be populated.
- Validate all request bodies and return `400` with a descriptive message for invalid input.

### API Efficiency

You will be assessed on how efficiently you query upstream services:

- Avoid querying the same upstream endpoint more than once per dashboard retrieval if data can be reused.
- Fetch only the fields you need — use API filtering parameters where available.
- Fan out concurrent requests (e.g., weather + air quality in parallel) where calls are independent.

---

## Deployment

The service must be containerised and deployed to the course OpenStack instance (SkyHigh) using Docker. Initial development should occur on your local machine.

**Requirements:**

- Provide a `Dockerfile` that builds and runs your service.
- Provide a `docker-compose.yml` if your setup requires multiple containers.
- The service must start correctly from a `docker run` command documented in your README.
- Configuration (API keys, Firebase credentials, ports) must be passed via environment variables — never hard-coded.

**Submission deliverables:**

- Link to your code repository (set to **internal** at submission time).
- URL to the deployed service running on SkyHigh.
- README documenting setup, environment variables, known issues, and group contributions.

---

## Group Size & Workload

This assignment is scoped for a group of **three students**. Your README must include a contributions section describing which areas each member was primarily responsible for (endpoints, testing, deployment, etc.).

### Groups of Four or More

If your group has more than three members, you are expected to implement additional features beyond the base requirements. Examples:

- Timezone-aware timestamps in all date/time fields.
- `PATCH` support on both `/registrations/` and `/notifications/` endpoints.
- An additional environmental data source (e.g., UV index, pollen count) integrated into the dashboard, using a documented open API.
- A simple HTML status page served at `/envdash/v1/status/ui` showing service health visually.

If in doubt about scope, discuss with course staff before committing.

---

## General Notes

### Professionalism

Apply professional standards throughout: clean commit history, meaningful commit messages, structured code, and thorough documentation. You may introduce additional endpoints or features as long as they do not violate the specification above.

### Third-Party Libraries

Be deliberate about external dependencies. The assignment is designed so that only Go's standard library and the Firebase/Firestore SDK are needed. If you choose to add other libraries, justify each choice in your README. Note that course staff may have limited ability to support highly specialised libraries.

### Rate Limits

Be considerate of rate limits on all external APIs, especially Open-Meteo and Nominatim. Given the number of student projects hitting these services simultaneously, unnecessary polling or bulk requests can cause problems for the whole cohort.

### Specification Ambiguities

Where the specification is silent on a detail, apply best practices and document your decision. If something is genuinely unclear, post an issue on the course repository as early as possible so clarifications can be shared with all groups.

---

## Peer Review

After the submission deadline, a separate peer review window opens. You will evaluate at least one other group's submission using a provided checklist. Each student must complete at least one review. The peer review deadline is on the course wiki.
