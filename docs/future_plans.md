# Future Plans

Phase-by-phase plan to revamp UISP NOC into a full-featured NOC monitoring application for browser and Android with parity across platforms.

## Phase 1: Discovery & Target Decisions
- [X] Inventory current codepaths (PHP UI, JS assets, Android WebView, Docker/Caddy/Gotify).
- [X] Confirm target users, roles, on-call schedule, notification channels, and SSO needs.
- [ ] Pick core stack: API framework, Postgres/Timescale, Redis/queue (NATS/Rabbit), front-end (React/Vue/Svelte), Android native (Kotlin).
- [ ] Define environments (dev/stage/prod) and hosting model.
- [ ] Decide identity (OIDC/SSO), RBAC roles, token format.
- [ ] Agree on alert policy defaults per role (gateway/AP/backbone): downtime, flap, latency, packet loss, utilization.
- [ ] Choose notification channels for v1 (Gotify, FCM push, email/SMS/webhook) and escalation rules.
- [ ] Approve dependency mapping approach (site -> gateway -> AP/switch), maintenance windows, and suppression rules.
- [ ] Lock migration strategy (parallel run vs cutover) and deprecation plan for PHP UI/WebView.

## Phase 2: Architecture & Data Modeling
- [ ] Draft entity model: devices, sites, roles, incidents, alerts, acknowledgements, maintenance windows, users, tokens, notification endpoints, metrics.
- [ ] Design API schema (REST/GraphQL) with versioning and auth flows.
- [ ] Define event model for poller -> queue -> API (normalized device state + deltas).
- [ ] Model alert engine inputs/outputs (rules, thresholds, suppression, escalations).
- [ ] Plan metrics storage (Postgres/Timescale schemas, retention policies) and backups.
- [ ] Design incident timeline/audit log schema.
- [ ] Specify WebSocket/SSE channels for live updates.
- [ ] Define mobile config endpoint (push token registration, config bootstrap).
- [ ] Document runbooks for device actions (simulate outage, ack, maintenance).
- [ ] Review data privacy, secrets handling, and PII boundaries.

## Phase 3: Infrastructure & Build System
- [ ] Create multi-service Docker/Compose/Helm layout: api, poller, notifier, web, caddy, gotify (optional), db, redis/queue.
- [ ] Add non-root images, healthchecks, readiness/liveness probes.
- [ ] Wire secrets via env/secret store; add config templates per environment.
- [ ] Add logging/metrics emitters (OpenTelemetry/Prometheus) and base dashboards.
- [ ] Set up migrations tooling and CI hooks.
- [ ] Add automated builds for amd64/arm64; cache dependencies.
- [ ] Provision staging infrastructure and seed test data.
- [ ] Document backup/restore for Postgres and artifacts.

## Phase 4: Backend API & Auth
- [ ] Scaffold API service with OIDC/session/token auth middleware and RBAC.
- [ ] Implement users/tokens/session management endpoints.
- [ ] Implement inventory endpoints (sites, devices, roles, tags).
- [ ] Implement incident/alert endpoints (list, create, acknowledge, clear, maintenance).
- [ ] Add history/metrics endpoints and websocket feeds for live state.
- [ ] Add mobile bootstrap endpoint (UISP URL/token, push registration).
- [ ] Add audit log endpoints and write-path hooks for acks/changes.
- [ ] Harden rate limits, CORS, CSRF, and input validation.
- [ ] Write API contract tests.

## Phase 5: Poller & Normalization
- [ ] Implement UISP poller service with rate limits, retries, and backoff.
- [ ] Normalize UISP device payloads into internal schema (role mapping, status).
- [ ] Publish state changes to queue; dedupe identical states.
- [ ] Persist periodic metrics into Postgres/Timescale.
- [ ] Add dependency awareness (site/gateway/AP links).
- [ ] Expose poller health/lag metrics.
- [ ] Add configurable polling cadence and per-role ping/latency collection.
- [ ] Test poller edge cases (timeouts, auth errors, partial data).

## Phase 6: Alert Engine & Policy
- [ ] Implement rule evaluation worker (downtime thresholds, flaps, sustained latency/jitter/packet loss, utilization, temperature).
- [ ] Add maintenance window and blackout handling.
- [ ] Add dependency-based suppression (e.g., mute APs when upstream gateway is down).
- [ ] Implement escalation schedules and repeat timers.
- [ ] Store incident lifecycle and SLA timers (MTTA/MTTR).
- [ ] Add manual incident creation and simulate outage hooks.
- [ ] Write unit/integration tests for the rule matrix.

## Phase 7: Notification Service
- [ ] Build notification router: fan-out to Gotify, FCM, email/SMS/webhook with templates.
- [ ] Add device/browser push token registration and expiry cleanup.
- [ ] Implement dedupe/throttle windows and per-channel priorities.
- [ ] Add actionable notifications (Ack/Clear/Runbook links).
- [ ] Deliverability metrics and dead-letter handling.
- [ ] End-to-end smoke tests for each channel.

## Phase 8: Web App (SPA)
- [ ] Scaffold SPA with routing, auth guard, and websocket client.
- [ ] Build wallboard/home with live counts, health, SLA timers, and incident queue.
- [ ] Device grids by role/site with filters, tags, search, and quick actions.
- [ ] Device detail: charts (latency/CPU/RAM/temp), history, incidents, runbook links.
- [ ] Incident views: list, filters, ack/clear, maintenance toggles, timeline.
- [ ] Settings: notification preferences, schedules, tokens, TLS info, test alert.
- [ ] Add outage simulation/maintenance UI, dependency graph, and map view.
- [ ] Responsive/mobile-friendly layouts; kiosk mode.
- [ ] Accessibility pass; loading/error states; offline cache for last-known state.
- [ ] Front-end tests (component and e2e) and performance budgets.

## Phase 9: Android App (Native)
- [ ] Scaffold Kotlin app with auth flow (OIDC/token) and secure storage.
- [ ] Implement API client plus websocket/SSE for live updates.
- [ ] Build wallboard and incident list; detail views with charts and actions.
- [ ] Add ack/maintenance/clear actions with optimistic UI.
- [ ] Integrate FCM push: token registration, background handling, actionable notifications.
- [ ] Implement background worker for periodic sync respecting Doze/battery optimizations.
- [ ] Add settings for sites, quiet hours, channel preferences, URLs.
- [ ] Kiosk/launcher mode option for NOC displays; offline cache for last state.
- [ ] Instrumentation/UI tests and WorkManager tests.
- [ ] Branding, icons, and store/MDM metadata.

## Phase 10: Ops, Observability, Hardening
- [ ] Add tracing/logging/metrics across services; dashboards for poller lag, queue depth, notification failures.
- [ ] Configure alerts on service health, SLA breaches, and push delivery errors.
- [ ] Security hardening: headers, TLS, secrets rotation, dependency scanning, SAST/DAST.
- [ ] Backups and retention policies for metrics/incidents; restore drills.
- [ ] CI/CD pipelines for API/web/poller/notifier/Android builds; signing and release tracks.
- [ ] Run load/soak tests for polling and notification fan-out; tune resources.

## Phase 11: Migration & Cutover
- [ ] Stand up new stack in staging; mirror polling/alerts alongside legacy PHP UI.
- [ ] Validate parity for device states, incidents, notifications; fix gaps.
- [ ] Pilot web SPA with limited users; pilot Android app with on-call techs.
- [ ] Train users/runbooks; gather feedback on thresholds and UI.
- [ ] Plan cutover window; freeze legacy changes; enable new stack in production.
- [ ] Monitor closely; document rollback plan.
- [ ] Deprecate PHP UI/WebView; archive legacy images; update docs.

## Phase 12: Post-Launch Iteration
- [ ] Triage production feedback; adjust UX and policies.
- [ ] Add additional integrations (ticketing, chatops).
- [ ] Expand device actions (reboot/power-cycle) with guardrails.
- [ ] Optimize costs and performance; refine retention and dashboards.
- [ ] Schedule recurring chaos drills and on-call handoff reviews.


