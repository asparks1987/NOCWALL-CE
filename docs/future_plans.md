# Future Plans

Phase-by-phase plan to revamp UISP NOC into a full-featured NOC monitoring application for browser and Android with parity across platforms.

## Phase 1: Discovery & Target Decisions
- [X] Inventory current codepaths (PHP UI, JS assets, Android WebView, Docker/Caddy/Gotify).
- [X] Confirm target users, roles, on-call schedule, notification channels, and SSO needs.
- [X] Pick core stack: API framework, Postgres/Timescale, Redis/queue (NATS/Rabbit), front-end (React/Vue/Svelte), Android native (Kotlin).
- [X] Define environments (dev/stage/prod) and hosting model.
- [X] Decide identity (OIDC/SSO), RBAC roles, token format.
- [X] Agree on alert policy defaults per role (gateway/AP/backbone): downtime, flap, latency, packet loss, utilization.
- [X] Choose notification channels for v1 (Gotify, FCM push, email/SMS/webhook) and escalation rules.
- [X] Approve dependency mapping approach (site -> gateway -> AP/switch), maintenance windows, and suppression rules.
- [X] Lock migration strategy (parallel run vs cutover) and deprecation plan for PHP UI/WebView.

## Phase 2: Architecture & Data Modeling
- [X] Draft entity model: devices, sites, roles, incidents, alerts, acknowledgements, maintenance windows, users, tokens, notification endpoints, metrics.
- [X] Design API schema (REST/GraphQL) with versioning and auth flows.
- [X] Define event model for poller -> queue -> API (normalized device state + deltas).
- [X] Model alert engine inputs/outputs (rules, thresholds, suppression, escalations).
- [X] Plan metrics storage (Postgres/Timescale schemas, retention policies) and backups.
- [X] Design incident timeline/audit log schema.
- [X] Specify WebSocket/SSE channels for live updates.
- [X] Define mobile config endpoint (push token registration, config bootstrap).
- [X] Document runbooks for device actions (simulate outage, ack, maintenance).
- [X] Review data privacy, secrets handling, and PII boundaries.

## Phase 3: Infrastructure & Build System
- [X] Create multi-service Docker/Compose/Helm layout: api, poller, notifier, web, caddy, gotify (optional), db, redis/queue.
- [X] Add non-root images, healthchecks, readiness/liveness probes.
- [X] Wire secrets via env/secret store; add config templates per environment.
- [X] Add logging/metrics emitters (OpenTelemetry/Prometheus) and base dashboards.
- [X] Set up migrations tooling and CI hooks.
- [X] Add automated builds for amd64/arm64; cache dependencies.
- [X] Provision staging infrastructure and seed test data.
- [X] Document backup/restore for Postgres and artifacts.

## Phase 4: Backend API & Auth
- [X] Scaffold API service with OIDC/session/token auth middleware and RBAC.
- [X] Implement users/tokens/session management endpoints.
- [X] Implement inventory endpoints (sites, devices, roles, tags).
- [X] Implement incident/alert endpoints (list, create, acknowledge, clear, maintenance).
- [X] Add history/metrics endpoints and websocket feeds for live state.
- [X] Add mobile bootstrap endpoint (UISP URL/token, push registration).
- [X] Add audit log endpoints and write-path hooks for acks/changes.
- [X] Harden rate limits, CORS, CSRF, and input validation.
- [X] Write API contract tests.

## Phase 5: Poller & Normalization
- [X] Implement UISP poller service with rate limits, retries, and backoff.
- [X] Normalize UISP device payloads into internal schema (role mapping, status).
- [X] Publish state changes to queue; dedupe identical states.
- [X] Persist periodic metrics into Postgres/Timescale.
- [X] Add dependency awareness (site/gateway/AP links).
- [X] Expose poller health/lag metrics.
- [X] Add configurable polling cadence and per-role ping/latency collection.
- [X] Test poller edge cases (timeouts, auth errors, partial data).

## Phase 6: Alert Engine & Policy
- [X] Implement rule evaluation worker (downtime thresholds, flaps, sustained latency/jitter/packet loss, utilization, temperature).
- [X] Add maintenance window and blackout handling.
- [X] Add dependency-based suppression (e.g., mute APs when upstream gateway is down).
- [X] Implement escalation schedules and repeat timers.
- [X] Store incident lifecycle and SLA timers (MTTA/MTTR).
- [X] Add manual incident creation and simulate outage hooks.
- [X] Write unit/integration tests for the rule matrix.

## Phase 7: Notification Service
- [X] Build notification router: fan-out to Gotify, FCM, email/SMS/webhook with templates.
- [X] Add device/browser push token registration and expiry cleanup.
- [X] Implement dedupe/throttle windows and per-channel priorities.
- [X] Add actionable notifications (Ack/Clear/Runbook links).
- [X] Deliverability metrics and dead-letter handling.
- [X] End-to-end smoke tests for each channel.

## Phase 8: Web App (SPA)
- [X] Scaffold SPA with routing, auth guard, and websocket client.
- [X] Build wallboard/home with live counts, health, SLA timers, and incident queue.
- [X] Device grids by role/site with filters, tags, search, and quick actions.
- [X] Device detail: charts (latency/CPU/RAM/temp), history, incidents, runbook links.
- [X] Incident views: list, filters, ack/clear, maintenance toggles, timeline.
- [X] Settings: notification preferences, schedules, tokens, TLS info, test alert.
- [X] Add outage simulation/maintenance UI, dependency graph, and map view.
- [X] Responsive/mobile-friendly layouts; kiosk mode.
- [X] Accessibility pass; loading/error states; offline cache for last-known state.
- [X] Front-end tests (component and e2e) and performance budgets.

## Phase 9: Android App (Native)
- [X] Scaffold Kotlin app with auth flow (OIDC/token) and secure storage.
- [X] Implement API client plus websocket/SSE for live updates.
- [X] Build wallboard and incident list; detail views with charts and actions.
- [X] Add ack/maintenance/clear actions with optimistic UI.
- [X] Integrate FCM push: token registration, background handling, actionable notifications.
- [X] Implement background worker for periodic sync respecting Doze/battery optimizations.
- [X] Add settings for sites, quiet hours, channel preferences, URLs.
- [X] Kiosk/launcher mode option for NOC displays; offline cache for last state.
- [X] Instrumentation/UI tests and WorkManager tests.
- [X] Branding, icons, and store/MDM metadata.

## Phase 10: Ops, Observability, Hardening
- [X] Add tracing/logging/metrics across services; dashboards for poller lag, queue depth, notification failures.
- [X] Configure alerts on service health, SLA breaches, and push delivery errors.
- [X] Security hardening: headers, TLS, secrets rotation, dependency scanning, SAST/DAST.
- [X] Backups and retention policies for metrics/incidents; restore drills.
- [X] CI/CD pipelines for API/web/poller/notifier/Android builds; signing and release tracks.
- [X] Run load/soak tests for polling and notification fan-out; tune resources.

## Phase 11: Migration & Cutover
- [X] Stand up new stack in staging; mirror polling/alerts alongside legacy PHP UI.
- [X] Validate parity for device states, incidents, notifications; fix gaps.
- [X] Pilot web SPA with limited users; pilot Android app with on-call techs.
- [X] Train users/runbooks; gather feedback on thresholds and UI.
- [X] Plan cutover window; freeze legacy changes; enable new stack in production.
- [X] Monitor closely; document rollback plan.
- [X] Deprecate PHP UI/WebView; archive legacy images; update docs.

## Phase 12: Post-Launch Iteration
- [X] Triage production feedback; adjust UX and policies.
- [X] Add additional integrations (ticketing, chatops).
- [X] Expand device actions (reboot/power-cycle) with guardrails.
- [X] Optimize costs and performance; refine retention and dashboards.
- [X] Schedule recurring chaos drills and on-call handoff reviews.































































































