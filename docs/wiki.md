# UISP NOC Wiki (Index)

Central index for all documentation. Links are workspace-relative.

## Core References
- [README.md](../README.md) — project overview, revamp roadmap, current features.
- [docs/CADDY.md](CADDY.md) — TLS and reverse proxy notes.
- [docs/DOCKERHUB.md](DOCKERHUB.md) — container summary.
- [docs/GOTIFY.md](GOTIFY.md) — embedded Gotify details.
- [docs/future_plans.md](future_plans.md) — end-to-end phase plan.
- [docs/inventory.md](inventory.md) — codebase inventory (PHP/UI, JS, Android, Docker/Caddy, Gotify).

## Phase 1 — Discovery & Targets
- [docs/phase1_targets.md](phase1_targets.md) — users, roles, on-call, channels, SSO.
- [docs/phase1_stack.md](phase1_stack.md) — core stack choices + error banner/logging expectations.
- [docs/phase1_envs.md](phase1_envs.md) — environments/hosting, observability, banners.
- [docs/phase1_identity.md](phase1_identity.md) — identity, RBAC, token strategy.
- [docs/phase1_alert_policies.md](phase1_alert_policies.md) — default alert policies per role.
- [docs/phase1_notifications.md](phase1_notifications.md) — notification channels and escalation.
- [docs/phase1_dependency.md](phase1_dependency.md) — dependency mapping and suppression.
- [docs/phase1_migration.md](phase1_migration.md) — legacy -> new stack migration strategy.

## Phase 2 — Architecture & Data Modeling
- [docs/phase2_entities.md](phase2_entities.md) — entity model.
- [docs/phase2_api_schema.md](phase2_api_schema.md) — API schema draft.
- [docs/phase2_events.md](phase2_events.md) — event model (poller -> queue -> API).
- [docs/phase2_alert_engine.md](phase2_alert_engine.md) — alert engine inputs/outputs.
- [docs/phase2_metrics.md](phase2_metrics.md) — metrics storage/retention.
- [docs/phase2_incident_audit.md](phase2_incident_audit.md) — incident timeline + audit log.
- [docs/phase2_realtime.md](phase2_realtime.md) — WebSocket/SSE channels.
- [docs/phase2_mobile_bootstrap.md](phase2_mobile_bootstrap.md) — mobile config endpoint.
- [docs/phase2_runbooks.md](phase2_runbooks.md) — runbooks for device actions.
- [docs/phase2_privacy.md](phase2_privacy.md) — privacy, secrets, PII boundaries.

## Phase 3 — Infrastructure & Build
- [docs/phase3_infra.md](phase3_infra.md) — multi-service layout (Compose/Helm).
- [docs/phase3_hardening.md](phase3_hardening.md) — images, healthchecks, probes.
- [docs/phase3_secrets.md](phase3_secrets.md) — secrets/config templates.
- [docs/phase3_observability.md](phase3_observability.md) — OTel/Prometheus/dashboards.
- [docs/phase3_ci.md](phase3_ci.md) — migrations tooling & CI hooks.
- [docs/phase3_builds.md](phase3_builds.md) — multi-arch builds.
- [docs/phase3_staging.md](phase3_staging.md) — staging + test data.
- [docs/phase3_backup.md](phase3_backup.md) — backups/restore.

## Phase 4 — Backend API & Auth
- [docs/phase4_api_scaffold.md](phase4_api_scaffold.md) — auth/RBAC scaffold.
- [docs/phase4_auth_endpoints.md](phase4_auth_endpoints.md) — users/tokens/session endpoints.
- [docs/phase4_inventory.md](phase4_inventory.md) — inventory/topology endpoints.
- [docs/phase4_incidents.md](phase4_incidents.md) — incident/maintenance/alerts endpoints.
- [docs/phase4_metrics_endpoints.md](phase4_metrics_endpoints.md) — metrics/history endpoints.
- [docs/phase4_mobile_endpoint.md](phase4_mobile_endpoint.md) — mobile bootstrap endpoint.
- [docs/phase4_audit_endpoints.md](phase4_audit_endpoints.md) — audit endpoints.
- [docs/phase4_security.md](phase4_security.md) — security hardening.
- [docs/phase4_tests.md](phase4_tests.md) — API contract tests.

## Phase 5 — Poller & Normalization
- [docs/phase5_poller.md](phase5_poller.md) — UISP poller.
- [docs/phase5_normalization.md](phase5_normalization.md) — payload normalization.
- [docs/phase5_streams.md](phase5_streams.md) — event publishing/dedupe.
- [docs/phase5_persistence.md](phase5_persistence.md) — metrics persistence.
- [docs/phase5_dependency_runtime.md](phase5_dependency_runtime.md) — dependency awareness.
- [docs/phase5_health.md](phase5_health.md) — poller health/lag metrics.
- [docs/phase5_cadence.md](phase5_cadence.md) — polling cadence/pings.
- [docs/phase5_tests.md](phase5_tests.md) — poller edge-case tests.

## Phase 6 — Alert Engine & Policy
- [docs/phase6_rules.md](phase6_rules.md) — rule evaluation worker.
- [docs/phase6_maintenance.md](phase6_maintenance.md) — maintenance/blackout handling.
- [docs/phase6_suppression.md](phase6_suppression.md) — dependency-based suppression.
- [docs/phase6_escalation.md](phase6_escalation.md) — escalation/repeats.
- [docs/phase6_sla.md](phase6_sla.md) — SLA timers (MTTA/MTTR).
- [docs/phase6_manual.md](phase6_manual.md) — manual incidents/simulate outage.
- [docs/phase6_tests.md](phase6_tests.md) — rule matrix tests.

## Phase 7 — Notification Service
- [docs/phase7_router.md](phase7_router.md) — notification router.
- [docs/phase7_tokens.md](phase7_tokens.md) — push token registration/cleanup.
- [docs/phase7_throttle.md](phase7_throttle.md) — dedupe/throttle/priorities.
- [docs/phase7_actionable.md](phase7_actionable.md) — actionable notifications.
- [docs/phase7_metrics.md](phase7_metrics.md) — deliverability metrics/DLQ.
- [docs/phase7_tests.md](phase7_tests.md) — channel smoke tests.

## Phase 8 — Web App (SPA)
- [docs/phase8_spa_scaffold.md](phase8_spa_scaffold.md) — SPA scaffold/auth.
- [docs/phase8_wallboard.md](phase8_wallboard.md) — wallboard/home.
- [docs/phase8_device_grids.md](phase8_device_grids.md) — device grids.
- [docs/phase8_device_detail.md](phase8_device_detail.md) — device detail/charts.
- [docs/phase8_incidents.md](phase8_incidents.md) — incident views.
- [docs/phase8_settings.md](phase8_settings.md) — settings UI.
- [docs/phase8_tools.md](phase8_tools.md) — simulation/maintenance/graph/map.
- [docs/phase8_responsive.md](phase8_responsive.md) — responsive/kiosk.
- [docs/phase8_accessibility.md](phase8_accessibility.md) — accessibility/loading/offline.
- [docs/phase8_tests.md](phase8_tests.md) — front-end tests/perf.

## Phase 9 — Android (Native)
- [docs/phase9_scaffold.md](phase9_scaffold.md) — app scaffold.
- [docs/phase9_client.md](phase9_client.md) — API client + realtime.
- [docs/phase9_ui.md](phase9_ui.md) — wallboard/incidents UI.
- [docs/phase9_actions.md](phase9_actions.md) — ack/maintenance/clear actions.
- [docs/phase9_push.md](phase9_push.md) — FCM integration.
- [docs/phase9_sync.md](phase9_sync.md) — background sync.
- [docs/phase9_settings.md](phase9_settings.md) — Android settings.
- [docs/phase9_kiosk.md](phase9_kiosk.md) — kiosk/offline cache.
- [docs/phase9_tests_android.md](phase9_tests_android.md) — Android tests.
- [docs/phase9_branding.md](phase9_branding.md) — branding/store metadata.

## Phase 10 — Ops/Observability/Hardening
- [docs/phase10_ops.md](phase10_ops.md) — ops observability/hardening.
- [docs/phase10_alerts.md](phase10_alerts.md) — ops alerts.
- [docs/phase10_security.md](phase10_security.md) — security hardening.
- [docs/phase10_backups.md](phase10_backups.md) — backups/retention.
- [docs/phase10_cicd.md](phase10_cicd.md) — CI/CD pipelines.
- [docs/phase10_load.md](phase10_load.md) — load/soak tests.

## Phase 11 — Migration & Cutover
- [docs/phase11_staging.md](phase11_staging.md) — staging mirror.
- [docs/phase11_parity.md](phase11_parity.md) — parity validation.
- [docs/phase11_pilot.md](phase11_pilot.md) — pilots.
- [docs/phase11_training.md](phase11_training.md) — training/feedback.
- [docs/phase11_cutover.md](phase11_cutover.md) — cutover plan.
- [docs/phase11_monitor.md](phase11_monitor.md) — post-cutover monitoring/rollback.
- [docs/phase11_deprecate.md](phase11_deprecate.md) — deprecate legacy UI/WebView.

## Phase 12 — Post-Launch
- [docs/phase12_feedback.md](phase12_feedback.md) — post-launch feedback.
- [docs/phase12_integrations.md](phase12_integrations.md) — additional integrations.
- [docs/phase12_actions.md](phase12_actions.md) — expanded device actions.
- [docs/phase12_optimize.md](phase12_optimize.md) — optimization/retention.
- [docs/phase12_drills.md](phase12_drills.md) — chaos drills/on-call reviews.
