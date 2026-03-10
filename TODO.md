# TODO — Air Quality & Environment Dashboard Service

Checklist for tracking implementation progress across sessions.
Mark items `[x]` when complete. See `CLAUDE.md` for architecture details.

---

## Session 1 — Foundation

- [x] `go.mod`
- [x] `.gitignore`
- [x] `CLAUDE.md`
- [x] `TODO.md`
- [ ] `.env.example`
- [ ] `internal/models/registration.go`
- [ ] `internal/models/dashboard.go`
- [ ] `internal/models/notification.go`
- [ ] `internal/models/errors.go`
- [ ] `internal/config/config.go`

## Session 2 — Firebase + External Clients

- [ ] `internal/firebase/client.go`
- [ ] `internal/firebase/registrations.go`
- [ ] `internal/firebase/notifications.go`
- [ ] `internal/firebase/cache.go`
- [ ] `internal/clients/http.go`
- [ ] `internal/clients/countries.go`
- [ ] `internal/clients/meteo.go`
- [ ] `internal/clients/openaq.go`
- [ ] `internal/clients/nominatim.go`
- [ ] `internal/clients/currency.go`

## Session 3 — Services + Handlers + Main

- [ ] `internal/webhook/dispatcher.go`
- [ ] `internal/services/registration.go`
- [ ] `internal/services/dashboard.go`
- [ ] `internal/services/notification.go`
- [ ] `internal/services/status.go`
- [ ] `internal/handlers/router.go`
- [ ] `internal/handlers/registrations.go`
- [ ] `internal/handlers/dashboards.go`
- [ ] `internal/handlers/notifications.go`
- [ ] `internal/handlers/status.go`
- [ ] `cmd/server/main.go`

## Session 4 — Tests

- [ ] `internal/handlers/registrations_test.go`
- [ ] `internal/handlers/dashboards_test.go`
- [ ] `internal/handlers/notifications_test.go`
- [ ] `internal/handlers/status_test.go`
- [ ] `internal/clients/countries_test.go`
- [ ] `internal/clients/meteo_test.go`
- [ ] `internal/clients/openaq_test.go`
- [ ] `internal/clients/currency_test.go`
- [ ] `internal/firebase/registrations_test.go` (integration, real Firestore)
- [ ] `internal/firebase/notifications_test.go` (integration, real Firestore)
- [ ] `internal/firebase/cache_test.go` (integration, real Firestore)
- [ ] Run `go test ./... -cover` and verify ≥75% coverage

## Session 5 — Deployment + Docs

- [ ] `Dockerfile` (multi-stage build)
- [ ] `docker-compose.yml`
- [ ] `README.md` (setup, env vars, curl examples, edge cases, contributions)
- [ ] Deploy to OpenStack SkyHigh
- [ ] Verify deployed service responds correctly

---

## Advanced Tasks

### Phase A (Easy — prioritized)
- [ ] HEAD /registrations/ — return headers only, no body
- [ ] PATCH /registrations/{id} — partial update (only supplied fields)
- [ ] Extra threshold operators: `>=` and `<=` (in addition to `>` and `<`)
- [ ] PATCH /notifications/{id} — partial update

### Phase B (Medium)
- [ ] Auto cache purging — background goroutine deletes expired cache docs
- [ ] `CACHE_PURGE_INTERVAL_HOURS` env var (default: 1)
- [ ] Purge also runs on startup

### Phase C (Complex — last)
- [ ] POST /auth/ — register client, receive API key
- [ ] DELETE /auth/{key} — revoke API key
- [ ] Middleware: validate `X-API-Key` header on all routes except /status/ and /auth/
- [ ] 401 for missing key, 403 for invalid/revoked key
- [ ] Store keys in `api_keys` Firestore collection

---

## Known Edge Cases to Document

- If `threshold.field` is not enabled in the dashboard config, the webhook is accepted but never fires
- If OpenAQ finds no stations within 50km, return `{pm25: -1, pm10: -1, level: "Unknown"}`
- `lastRetrieval` in dashboard response is always the current server time — never cached
- Webhook `country` field is the ISO code, not the full country name
- Webhook matches when `notification.Country == ""` (wildcard) OR `notification.Country == registration.ISOCode`
- `country` and `isoCode` are both optional in POST /registrations/ but at least one must be present
