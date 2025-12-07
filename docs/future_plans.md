# Future Plans

Phase-by-phase plan to revamp UISP NOC into a full-featured NOC monitoring application for browser and Android with parity across platforms.

## Phase 1: Discovery & Target Decisions
1. Inventory current codepaths (PHP UI, JS assets, Android WebView, Docker/Caddy/Gotify).
2. Confirm target users, roles, on-call schedule, notification channels, and SSO needs.
3. Pick core stack: API framework, Postgres/Timescale, Redis/queue (NATS/Rabbit), front-end (React/Vue/Svelte), Android native (Kotlin).
4. Define environments (dev/stage/prod) and hosting model.
5. Decide identity (OIDC/SSO), RBAC roles, token format.
6. Agree on alert policy defaults per role (gateway/AP/backbone): downtime, flap, latency, packet loss, utilization.
7. Choose notification channels for v1 (Gotify, FCM push, email/SMS/webhook) and escalation rules.
8. Approve dependency mapping approach (site -> gateway -> AP/switch), maintenance windows, and suppression rules.
9. Lock migration strategy (parallel run vs cutover) and deprecation plan for PHP UI/WebView.

## Phase 2: Architecture & Data Modeling
1. Draft entity model: devices, sites, roles, incidents, alerts, acknowledgements, maintenance windows, users, tokens, notification endpoints, metrics.
2. Design API schema (REST/GraphQL) with versioning and auth flows.
3. Define event model for poller -> queue -> API (normalized device state + deltas).
4. Model alert engine inputs/outputs (rules, thresholds, suppression, escalations).
5. Plan metrics storage (Postgres/Timescale schemas, retention policies) and backups.
6. Design incident timeline/audit log schema.
7. Specify WebSocket/SSE channels for live updates.
8. Define mobile config endpoint (push token registration, config bootstrap).
9. Document runbooks for device actions (simulate outage, ack, maintenance).
10. Review data privacy, secrets handling, and PII boundaries.

## Phase 3: Infrastructure & Build System
1. Create multi-service Docker/Compose/Helm layout: api, poller, notifier, web, caddy, gotify (optional), db, redis/queue.
2. Add non-root images, healthchecks, readiness/liveness probes.
3. Wire secrets via env/secret store; add config templates per environment.
4. Add logging/metrics emitters (OpenTelemetry/Prometheus) and base dashboards.
5. Set up migrations tooling and CI hooks.
6. Add automated builds for amd64/arm64; cache dependencies.
7. Provision staging infrastructure and seed test data.
8. Document backup/restore for Postgres and artifacts.

## Phase 4: Backend API & Auth
1. Scaffold API service with OIDC/session/token auth middleware and RBAC.
2. Implement users/tokens/session management endpoints.
3. Implement inventory endpoints (sites, devices, roles, tags).
4. Implement incident/alert endpoints (list, create, acknowledge, clear, maintenance).
5. Add history/metrics endpoints and websocket feeds for live state.
6. Add mobile bootstrap endpoint (UISP URL/token, push registration).
7. Add audit log endpoints and write-path hooks for acks/changes.
8. Harden rate limits, CORS, CSRF, and input validation.
9. Write API contract tests.

## Phase 5: Poller & Normalization
1. Implement UISP poller service with rate limits, retries, and backoff.
2. Normalize UISP device payloads into internal schema (role mapping, status).
3. Publish state changes to queue; dedupe identical states.
4. Persist periodic metrics into Postgres/Timescale.
5. Add dependency awareness (site/gateway/AP links).
6. Expose poller health/lag metrics.
7. Add configurable polling cadence and per-role ping/latency collection.
8. Test poller edge cases (timeouts, auth errors, partial data).

## Phase 6: Alert Engine & Policy
1. Implement rule evaluation worker (downtime thresholds, flaps, sustained latency/jitter/packet loss, utilization, temperature).
2. Add maintenance window and blackout handling.
3. Add dependency-based suppression (e.g., mute APs when upstream gateway is down).
4. Implement escalation schedules and repeat timers.
5. Store incident lifecycle and SLA timers (MTTA/MTTR).
6. Add manual incident creation and simulate outage hooks.
7. Write unit/integration tests for the rule matrix.

## Phase 7: Notification Service
1. Build notification router: fan-out to Gotify, FCM, email/SMS/webhook with templates.
2. Add device/browser push token registration and expiry cleanup.
3. Implement dedupe/throttle windows and per-channel priorities.
4. Add actionable notifications (Ack/Clear/Runbook links).
5. Deliverability metrics and dead-letter handling.
6. End-to-end smoke tests for each channel.

## Phase 8: Web App (SPA)
1. Scaffold SPA with routing, auth guard, and websocket client.
2. Build wallboard/home with live counts, health, SLA timers, and incident queue.
3. Device grids by role/site with filters, tags, search, and quick actions.
4. Device detail: charts (latency/CPU/RAM/temp), history, incidents, runbook links.
5. Incident views: list, filters, ack/clear, maintenance toggles, timeline.
6. Settings: notification preferences, schedules, tokens, TLS info, test alert.
7. Add outage simulation/maintenance UI, dependency graph, and map view.
8. Responsive/mobile-friendly layouts; kiosk mode.
9. Accessibility pass; loading/error states; offline cache for last-known state.
10. Front-end tests (component and e2e) and performance budgets.

## Phase 9: Android App (Native)
1. Scaffold Kotlin app with auth flow (OIDC/token) and secure storage.
2. Implement API client plus websocket/SSE for live updates.
3. Build wallboard and incident list; detail views with charts and actions.
4. Add ack/maintenance/clear actions with optimistic UI.
5. Integrate FCM push: token registration, background handling, actionable notifications.
6. Implement background worker for periodic sync respecting Doze/battery optimizations.
7. Add settings for sites, quiet hours, channel preferences, URLs.
8. Kiosk/launcher mode option for NOC displays; offline cache for last state.
9. Instrumentation/UI tests and WorkManager tests.
10. Branding, icons, and store/MDM metadata.

## Phase 10: Ops, Observability, Hardening
1. Add tracing/logging/metrics across services; dashboards for poller lag, queue depth, notification failures.
2. Configure alerts on service health, SLA breaches, and push delivery errors.
3. Security hardening: headers, TLS, secrets rotation, dependency scanning, SAST/DAST.
4. Backups and retention policies for metrics/incidents; restore drills.
5. CI/CD pipelines for API/web/poller/notifier/Android builds; signing and release tracks.
6. Run load/soak tests for polling and notification fan-out; tune resources.

## Phase 11: Migration & Cutover
1. Stand up new stack in staging; mirror polling/alerts alongside legacy PHP UI.
2. Validate parity for device states, incidents, notifications; fix gaps.
3. Pilot web SPA with limited users; pilot Android app with on-call techs.
4. Train users/runbooks; gather feedback on thresholds and UI.
5. Plan cutover window; freeze legacy changes; enable new stack in production.
6. Monitor closely; document rollback plan.
7. Deprecate PHP UI/WebView; archive legacy images; update docs.

## Phase 12: Post-Launch Iteration
1. Triage production feedback; adjust UX and policies.
2. Add additional integrations (ticketing, chatops).
3. Expand device actions (reboot/power-cycle) with guardrails.
4. Optimize costs and performance; refine retention and dashboards.
5. Schedule recurring chaos drills and on-call handoff reviews.


