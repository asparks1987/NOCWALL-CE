# UISP NOC - Consolidated Plan and Status

This document consolidates current state, what was implemented, what remains, and platform/multi-arch targets.

## Current State (implemented)
- Legacy stack: PHP single-page app + Vanilla JS/Chart.js, SQLite, Gotify, Caddy TLS sidecar.
- Docker Compose stack with Caddy proxy. Health checks in Compose for app/API/Caddy.
- Android: Kotlin scaffold with global diagnostic banner (codes/details/request IDs), structured error events, FCM token capture, and API client wired to mobile-config/devices/incidents. Uses API data when available; WebView fallback remains.
- Go API preview (`/mobile/config`, `/devices`, `/incidents`, `/incidents/:id/ack`, `/metrics/devices/:id`, `/push/register`, `/health`) with in-memory store; placeholder auth token; Dockerfile and Compose wiring.
- SPA preview: React/Vite minimal page at `/spa/` that reads devices/incidents from the API.
- CI: GitHub Actions building Android (assembleDebug) and a Docker image. `build-multiarch.sh` builds/pushes both app and API images.

## Multi-Arch / Platform Targets
- Images must ship for amd64 + arm64 (Linux) and run on x86_64/ARM hosts; API image currently failing with exec format error because the pushed tag is not multi-arch.
- Client targets: browser (desktop/mobile), Android (phone/tablet), iPad (via browser/SPA). Native Android client must function on arm64/x86 emulation.
- Action: rebuild and push multi-arch app/API images (`./build-multiarch.sh --author <you> --tag <tag>`) before deployment (Portainer/stack).

## Remaining Work (high level)
- Replace in-memory API with persistent store (Postgres/Redis). Implement full auth/RBAC, inventory, incidents, metrics, notifications, audit logs per contracts.
- Implement poller workers + alert engine (downtime/flap/latency/loss/utilization/temp, suppression, maintenance, escalation, SLA).
- Notification service with multi-channel fan-out and actionable notifications; persist and use FCM tokens from Android.
- Replace SPA preview with full React/Vite app (wallboard, grids, incidents, settings, dependency graph/map, offline/error banners with request IDs).
- Finish migration path: parity validation against legacy UI, staged cutover, rollback.
- Add backend tests, load/soak tests, observability (OTel/Prometheus), security hardening.

## Phase Checklist (planning done; build in progress)
- **Discovery/Targets:** Complete.
- **Architecture/Data:** Complete (entity/API/event/rule/metrics/realtime/runbooks/privacy defined).
- **Infrastructure:** Baseline Compose/CI and multi-arch builder present; need real DB/cache and Helm/k8s packaging later.
- **Backend API/Auth:** Contracts defined; implementation pending beyond current preview.
- **Poller/Normalization:** Planned; not implemented.
- **Alert Engine:** Planned; not implemented.
- **Notifications:** Planned; minimal push register endpoint only.
- **Web SPA:** Preview only; full app pending.
- **Android Native:** Scaffold + diagnostics + API data + FCM token capture; awaiting live backend and full UI.
- **Ops/Hardening:** CI + health checks; need observability, security, backups, load tests.
- **Migration/Cutover:** Pending once new stack achieves parity.
- **Post-Launch:** Pending (integrations, actions, optimization, drills).

## Build & Run (current)
- **Stack:** `docker compose up -d` (requires images `predheadtx/uisp-noc:beta` and `predheadtx/uisp-api:beta` or your rebuilt multi-arch equivalents; Caddy file embedded via Compose config).
- **Android:** `cd android && ./gradlew assembleDebug` (JDK 11+, Android SDK). Diagnostic banner and API data enabled; FCM token captured and sent to `/push/register`.

## Known Issues from Logs
- `api` image exec format error on target host: rebuild/push multi-arch API image.
- Caddy warnings about HTTP/2/3 skipped: expected when serving HTTP only.

## Immediate Next Steps
- Rebuild/push multi-arch images for app/API and redeploy to clear exec format error.
- Implement persistent API (Postgres/Redis) with auth/RBAC, inventory/incidents/metrics/notifications.
- Expand SPA to real UI; connect Android to live backend; wire notification service using stored FCM tokens.
