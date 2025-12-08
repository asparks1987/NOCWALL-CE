# UISP NOC - Consolidated Plan and Status

This document consolidates the roadmap, current state, and future targets for the browser and Android revamp.

## Current State
- Legacy stack: PHP single-page app + Vanilla JS/Chart.js, SQLite, Gotify, Caddy TLS sidecar.
- Dockerized (Compose) with embedded Gotify; Caddy terminates TLS.
- Android: Kotlin app scaffold with a global diagnostic banner (codes/details/request IDs), structured error events, and a new API client hooked to mobile-config/push/devices/incidents endpoints (backend WIP). Legacy WebView remains as fallback.

## Future Architecture (browser + Android parity)
- Versioned API service with auth/RBAC, inventory/incidents/alerts/metrics, WebSocket/SSE, and audit logs.
- UISP poller workers publishing events to Redis Streams/NATS; alert engine with rules (downtime/flap/latency/loss/utilization/temperature), suppression, maintenance, escalations, and SLA tracking.
- Notification service with multi-channel fan-out (Gotify, FCM, email/SMS/webhook) plus actionable notifications and DLQ.
- Web SPA (React + Vite) with wallboard, device grids, incident console, dependency graph/map, settings, and offline/error banners with request IDs.
- Native Android app (Kotlin) consuming the same API: wallboard/incidents/actions, FCM actionable push, background sync, offline cache, kiosk mode.
- Ops: OpenTelemetry/Prometheus dashboards, CI/CD pipelines, backups/restore, load/soak tests, security hardening.
- Migration: parallel run with legacy UI, parity validation, staged cutover, rollback plan, legacy deprecation.

## Phase Checklist (completed planning; build in progress)
- **Discovery/Targets:** Users/roles, on-call schedules, notification channels, SSO, alert policies, dependency mapping, migration approach.
- **Architecture/Data:** Entity model, API schema, event model, alert engine inputs/outputs, metrics retention, incident/audit schema, realtime channels, mobile config endpoint, runbooks, privacy.
- **Infrastructure:** Multi-service layout (api/poller/notifier/web/caddy/gotify/db/redis), healthchecks, secrets/config templates, observability, CI/migrations, multi-arch builds, staging seeds, backups.
- **Backend API/Auth:** Endpoint contracts for auth, inventory, incidents/maintenance, metrics/history, mobile bootstrap, audit, security, contract tests.
- **Poller/Normalization:** UISP poller, normalization, event publishing/dedupe, metrics persistence, dependency awareness, health, cadence, tests.
- **Alert Engine:** Rule worker, maintenance/blackouts, dependency suppression, escalations, SLA tracking, manual incidents, tests.
- **Notifications:** Router, token registration/cleanup, dedupe/throttle, actionable notifications, deliverability/DLQ, smoke tests.
- **Web SPA:** Scaffold, wallboard, device grids/detail, incidents, settings, simulation/maintenance/graph/map, responsive/kiosk, accessibility/offline/error states, tests/perf.
- **Android Native:** Scaffold, API client + realtime, wallboard/incidents UI, ack/maintenance, FCM, WorkManager sync, settings, kiosk/offline, tests, branding.
- **Ops/Hardening:** Observability, ops alerts, security, retention/backups, CI/CD pipelines, load/soak tests.
- **Migration/Cutover:** Staging mirror, parity validation, pilots, training, cutover/rollback, legacy deprecation.
- **Post-Launch:** Feedback, integrations, expanded device actions, optimization, chaos drills/on-call reviews.

## Build & Run (current)
- **Browser/Backend (legacy):**
  - `docker compose up -d`
  - Environment vars: `UISP_URL`, `UISP_TOKEN`, `NOC_DOMAIN` (Caddy), optional `SHOW_TLS_UI=1`.
- **Android (native scaffold):**
  - `cd android && ./gradlew assembleDebug`
  - Requires JDK 11+ and Android SDK; defaults to existing UISP URL/token auth flow. Mobile config call is wired; backend endpoints still in progress.
  - Global error banner shows codes/details/request IDs; “Copy diagnostics” for support.

## Next Build Targets
- Wire the new API responses (devices/incidents) into the Android UI (replace legacy UISP endpoints as backend lands).
- Hook FCM registration to `registerPush`.
- Swap the WebView fallback for native flows once SPA/API are available.
- Implement SPA and backend services per contracts above; add CI and health checks.
