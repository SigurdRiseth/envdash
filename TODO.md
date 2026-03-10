# TODO — Air Quality & Environment Dashboard Service

Checklist for tracking implementation progress across sessions.
Mark items `[x]` when complete. See `CLAUDE.md` for architecture details.

---

## Core Implementation — DONE

- [x] `go.mod` (Go 1.26, module `envdash`)
- [x] `.gitignore`
- [x] `CLAUDE.md`
- [x] `.env.example`
- [x] `internal/models/registration.go`
- [x] `internal/models/dashboard.go`
- [x] `internal/models/notification.go`
- [x] `internal/models/errors.go`
- [x] `internal/models/aqi.go` (AQILevel helper + tests)
- [x] `internal/config/config.go`
- [x] `internal/firebase/client.go`
- [x] `internal/firebase/registrations.go`
- [x] `internal/firebase/notifications.go`
- [x] `internal/firebase/cache.go`
- [x] `internal/clients/http.go`
- [x] `internal/clients/countries.go`
- [x] `internal/clients/meteo.go`
- [x] `internal/clients/openaq.go`
- [x] `internal/clients/nominatim.go`
- [x] `internal/clients/currency.go`
- [x] `internal/webhook/dispatcher.go`
- [x] `internal/services/registration.go`
- [x] `internal/services/dashboard.go`
- [x] `internal/services/notification.go`
- [x] `internal/services/status.go`
- [x] `internal/handlers/router.go`
- [x] `internal/handlers/registrations.go`
- [x] `internal/handlers/dashboards.go`
- [x] `internal/handlers/notifications.go`
- [x] `internal/handlers/status.go`
- [x] `cmd/server/main.go`
- [x] `Dockerfile` (multi-stage build)
- [x] `docker-compose.yml`
- [x] `README.md`

## Tests — DONE

- [x] `internal/models/aqi_test.go`
- [x] `internal/handlers/registrations_test.go`
- [x] `internal/handlers/dashboards_test.go`
- [x] `internal/handlers/notifications_test.go`
- [x] `internal/handlers/status_test.go`
- [x] `internal/clients/countries_test.go`
- [x] `internal/clients/meteo_test.go`
- [x] `internal/clients/openaq_test.go`
- [x] `internal/clients/currency_test.go`

## Tests — Remaining

- [ ] `internal/firebase/registrations_test.go` (integration, real Firestore, `//go:build integration`)
- [ ] `internal/firebase/notifications_test.go` (integration, real Firestore)
- [ ] `internal/firebase/cache_test.go` (integration, real Firestore)
- [ ] Improve client test coverage (nominatim, cache-hit paths) — currently 64%

## Advanced Tasks — DONE

- [x] HEAD /registrations/ — return headers only, no body
- [x] PATCH /registrations/{id} — partial update (only supplied fields)
- [x] PATCH /notifications/{id} — partial update
- [x] Extra threshold operators: `>=` and `<=`
- [x] Auto cache purging — background goroutine + startup purge
- [x] `CACHE_PURGE_INTERVAL_HOURS` env var (default: 1)

## Advanced Tasks — Remaining (Phase C)

- [ ] POST /auth/ — register client, receive API key (`sk-envdash-{hex}`)
- [ ] DELETE /auth/{key} — revoke API key
- [ ] Middleware: validate `X-API-Key` header on all routes except `/status/` and `/auth/`
- [ ] Return 401 for missing key, 403 for invalid/revoked key
- [ ] Store keys in `api_keys` Firestore collection
- [ ] Tests for auth middleware

## Deployment — Remaining

- [ ] Set up Firebase project (create + enable Firestore)
- [ ] Obtain OpenAQ API key
- [ ] Deploy to OpenStack SkyHigh instance
- [ ] Verify deployed service at public URL
- [ ] Update README with deployed URL

---

## Known Edge Cases (all documented in README)

- Threshold on disabled field → webhook accepted but never fires
- No OpenAQ stations within 50km → `{pm25: -1, pm10: -1, level: "Unknown"}`
- `lastRetrieval` is always real-time, never cached
- Webhook `country` wildcard: empty string matches all countries
- `country` and `isoCode` both optional — at least one required
- Partial upstream failure → affected fields null, others populated
