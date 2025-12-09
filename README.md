# UISP NOC

A self-hosted, zero-friction Network Operations Center for Ubiquiti UISP deployments. Legacy stack: PHP/JS UI, SQLite metrics, embedded Gotify, Caddy TLS sidecar, Docker Compose. Revamp in progress with a new API, SPA, and native Android client.

## Current State
- Legacy PHP/JS dashboard with siren, acks, outage simulation, TLS modal, Gotify alerts.
- Docker Compose with Caddy + embedded Gotify.
- Android: Kotlin scaffold with global diagnostic banner (codes/details/request IDs), new API client (mobile-config/devices/incidents), FCM push token capture, and legacy WebView fallback.
- New lightweight Go API skeleton (`api/`) exposing `/mobile/config`, `/devices`, `/incidents`, `/push/register`, `/health` with Dockerfile and Compose wiring. Minimal SPA preview at `web/index.html` proxied via Caddy at `/spa/`.
- CI: GitHub Actions builds Android (assembleDebug) and a Docker image. Healthchecks added for `uisp-noc`, `api`, and `caddy` in Compose.

## Roadmap
See `docs/PROJECT_PLAN.md` for the consolidated plan (API/poller/alerting/notifications/SPA/Android/migration). `docs/wiki.md` links remaining component docs.

## Run (legacy + api preview)
```bash
docker compose up -d
```
- Legacy UI: https://localhost (via Caddy)
- API preview: http://localhost:8080 (proxied via Caddy `/api`)
- Minimal SPA preview: http://localhost/spa/ (static demo hitting the API preview)

## Android Build
```bash
cd android
./gradlew assembleDebug
```
- Uses default debug keystore fallback if none provided.
- Diagnostic banner shows errors with request IDs; API data used when available.

## Multi-arch Image
```bash
./build-multiarch.sh --image youruser/uisp-noc:tag
```

## Documentation
- `docs/PROJECT_PLAN.md` — consolidated status/roadmap.
- `docs/wiki.md` — quick index; phase docs are pointers.
- `docs/CADDY.md`, `docs/DOCKERHUB.md`, `docs/GOTIFY.md`, `docs/inventory.md` — operations/components.

## Testing
- Android unit: `./gradlew test`
- Android instrumented: `./gradlew connectedAndroidTest` (device/emulator required)
- Legacy UI: manual verification.

## Next Steps
- Flesh out API endpoints per plan (auth, inventory, incidents, metrics, notifications) and persist data stores.
- Replace SPA preview with full React/Vite app consuming the new API.
- Wire FCM `registerPush` to live backend once available.
- Add backend tests and tighter CI checks.
