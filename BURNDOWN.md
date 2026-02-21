# NOCWALL-CE Burndown (Recreated)

Chronological multi-phase plan to move from the current state to a fully functional, distributable, and sellable NOCWALL platform.
Strategic connector direction: UISP is first, with long-term expansion toward comprehensive multi-vendor NMS API coverage.
Open-core product rule: CE remains a minimal wall-mounted online/offline dashboard; advanced workflows, analytics, and operations are PRO.

Legend:
- `[ ]` pending
- `[x]` completed
- `[STUB]` placeholder only
- `[CE]` Community Edition target
- `[PRO]` Private/paid target

## Current Baseline (Already Working)
- [x] Local account signup/login
- [x] Per-account UISP source storage
- [x] Device cards, alert siren, ack/clear/simulate
- [x] Per-account dashboard preference sync
- [x] Dockerized local stack
- [ ] Planned product split change: move non-minimal CE capabilities behind PRO gates.

## 25 New CE Adoption Features (Strictly Non-PRO)

These features are intentionally limited to wallboard usability, reliability, and onboarding.
They avoid PRO-only domains (team workflows, correlation, automation, enterprise controls, deep analytics).

1. `[x]` CEF01 `[CE]` First-run setup wizard (add source + test + start dashboard)
2. `[ ]` CEF02 `[CE]` Demo data toggle from UI for instant wallboard preview
3. `[ ]` CEF03 `[CE]` Source connectivity diagnostics panel (DNS/TLS/API reachability)
4. `[x]` CEF04 `[CE]` Manual "Poll Now" button per source
5. `[x]` CEF05 `[CE]` Source status strip (last poll time, success/fail, latency)
6. `[x]` CEF06 `[CE]` Global search box for device name/MAC/hostname
7. `[x]` CEF07 `[CE]` Quick filters (all/offline/online by tab)
8. `[x]` CEF08 `[CE]` Sort controls (status, name, last-seen)
9. `[ ]` CEF09 `[CE]` Group-by mode (role/site) for card layout
10. `[x]` CEF10 `[CE]` Saved default tab per account
11. `[ ]` CEF11 `[CE]` Saved refresh interval preset per account
12. `[ ]` CEF12 `[CE]` Kiosk mode hotkey + URL flag (hide controls/chrome)
13. `[ ]` CEF13 `[CE]` Keyboard shortcuts overlay (`?` help modal)
14. `[ ]` CEF14 `[CE]` Dashboard legend panel (status colors/icons meanings)
15. `[ ]` CEF15 `[CE]` Card status-change highlight (recently changed online/offline)
16. `[ ]` CEF16 `[CE]` "Last updated" stale-data warning banner
17. `[ ]` CEF17 `[CE]` API degraded banner with retry backoff indicator
18. `[ ]` CEF18 `[CE]` Read-only fallback rendering from last known cache snapshot
19. `[ ]` CEF19 `[CE]` Local browser notification option for new offline events
20. `[ ]` CEF20 `[CE]` Optional soft chime mode (lower-volume alert profile)
21. `[ ]` CEF21 `[CE]` Theme presets (classic/high-contrast/light) with persistence
22. `[ ]` CEF22 `[CE]` Font scaling presets for distant wall displays
23. `[ ]` CEF23 `[CE]` Card print/export to PNG snapshot for NOC reports
24. `[ ]` CEF24 `[CE]` Import/export dashboard preferences JSON
25. `[ ]` CEF25 `[CE]` In-app "What's new" changelog modal per release

## Phase CE-A - CE Adoption and Wallboard Polish

### Epic CE-A1 - CE Feature Pack 1 (CEF01-CEF10)
- [x] RCE01 Implement first-run wizard with source save/test handoff.
- [ ] RCE02 Add demo data UI toggle and API wiring.
- [ ] RCE03 Build source diagnostics panel and health probes.
- [x] RCE04 Add per-source "Poll Now" actions and status strip.
- [x] RCE05 Add search + quick filters across tabs.
- [x] RCE06 Add sort modes and account-persisted defaults.
- [ ] RCE07 Implement group-by layout mode (role/site).
- [x] RCE08 Add default-tab persistence.

### Epic CE-A2 - CE Feature Pack 2 (CEF11-CEF18)
- [ ] RCE09 Add refresh interval presets and persistence.
- [ ] RCE10 Add kiosk mode (hotkey + query flag).
- [ ] RCE11 Add keyboard shortcuts help modal.
- [ ] RCE12 Add dashboard legend panel.
- [ ] RCE13 Add recent state-change highlights on cards.
- [ ] RCE14 Add stale-data warning banner.
- [ ] RCE15 Add API degraded/retry backoff banner.
- [ ] RCE16 Add read-only cached snapshot fallback path.

### Epic CE-A3 - CE Feature Pack 3 (CEF19-CEF25)
- [ ] RCE17 Add browser notifications option for offline events.
- [ ] RCE18 Add soft chime alert profile option.
- [ ] RCE19 Add theme presets and persistence.
- [ ] RCE20 Add wall-distance font scaling presets.
- [ ] RCE21 Add dashboard PNG snapshot export.
- [ ] RCE22 Add settings JSON import/export.
- [ ] RCE23 Add in-app release changelog modal.
- [ ] RCE24 Add CE-only tests and docs for all CE feature-pack flows.

## 50 Net-New Features (Not Previously Planned in BURNDOWN)

### Inventory and Topology
1. `[ ]` F01 `[PRO]` Multi-source device identity stitching (same device merged across UISP + agent feeds)
2. `[ ]` F02 `[PRO]` Configuration drift fingerprints per device
3. `[ ]` F03 `[PRO]` Interface-level utilization and error metrics
4. `[ ]` F04 `[PRO]` LLDP/CDP neighbor discovery inventory
5. `[ ]` F05 `[PRO]` Hardware lifecycle risk score (EOS/EOL awareness)
6. `[ ]` F06 `[PRO]` Auto-generated network topology map
7. `[ ]` F07 `[PRO]` Path trace view (gateway to target device)
8. `[ ]` F08 `[PRO]` Link-health heatmap overlays
9. `[ ]` F09 `[PRO]` WAN circuit SLA tracker
10. `[ ]` F10 `[PRO]` Redundancy/HA state monitor

### Telemetry and Signal Quality
11. `[ ]` F11 `[PRO]` Multi-tier retention (hot/warm/cold data windows)
12. `[ ]` F12 `[PRO]` Sampling-rate governor by device class
13. `[ ]` F13 `[PRO]` Telemetry gap detector with missing-data alerts
14. `[ ]` F14 `[PRO]` Agent/server clock-skew detector
15. `[ ]` F15 `[PRO]` Source data quality scorecard
16. `[ ]` F16 `[PRO]` Dynamic baseline thresholds per metric
17. `[ ]` F17 `[PRO]` Seasonal anomaly detection (hour/day pattern aware)
18. `[ ]` F18 `[PRO]` Alert confidence scoring
19. `[ ]` F19 `[PRO]` Impact-radius estimator
20. `[ ]` F20 `[PRO]` Alert storm shield with event summarization

### Incident and Operator Workflow
21. `[ ]` F21 `[PRO]` Incident commander mode
22. `[ ]` F22 `[PRO]` Shift handoff auto-briefs
23. `[ ]` F23 `[PRO]` Post-incident timeline export (Markdown/PDF)
24. `[ ]` F24 `[PRO]` Root-cause hypothesis assistant panel
25. `[ ]` F25 `[PRO]` Playbook checklist runner on incident pages

### Dashboard and Wallboard UX
26. `[ ]` F26 `[PRO]` Micro-sparklines inside device cards
27. `[ ]` F27 `[PRO]` Per-card quick actions menu
28. `[ ]` F28 `[PRO]` Pin/focus mode for critical devices
29. `[ ]` F29 `[PRO]` Wall rotation scenes (auto-cycle filtered views)
30. `[ ]` F30 `[PRO]` Accessibility profiles (colorblind/high-contrast/large text)

### Automation and Integrations
31. `[ ]` F31 `[PRO]` Safe remediation actions with approval queue
32. `[ ]` F32 `[PRO]` Drift auto-remediation suggestions
33. `[ ]` F33 `[PRO]` Scheduled health audits
34. `[ ]` F34 `[PRO]` Change-window suggestions from traffic patterns
35. `[ ]` F35 `[PRO]` Telemetry replay sandbox for testing rules
36. `[ ]` F36 `[PRO]` Signed outbound webhook templates
37. `[ ]` F37 `[PRO]` Inbound event adapter SDK
38. `[ ]` F38 `[PRO]` ChatOps command interface
39. `[ ]` F39 `[PRO]` GraphQL read API for BI tools
40. `[ ]` F40 `[PRO]` Data export pipelines to object storage

### Security, Reliability, and Commercialization
41. `[ ]` F41 `[PRO]` Per-account IP allowlists
42. `[ ]` F42 `[PRO]` Session anomaly detection
43. `[ ]` F43 `[PRO]` Vault-backed secret rotation jobs
44. `[ ]` F44 `[PRO]` Forensic audit bundle generator
45. `[ ]` F45 `[PRO]` Tamper-evident event log hashing
46. `[ ]` F46 `[PRO]` Usage metering by monitored device-hour
47. `[ ]` F47 `[PRO]` Self-serve plan limit banners/enforcement
48. `[ ]` F48 `[PRO]` Tenant cost/performance advisor
49. `[ ]` F49 `[PRO]` Zero-downtime hosted upgrade orchestration
50. `[ ]` F50 `[PRO]` Public reliability scorecard API

## Phase 0 - CE Minimalization and PRO Gating

### Epic 0 - Enforce Minimal CE Product Contract
- [x] [STUB] R00 Define strict CE card contract: online/offline, device name, role, site, last-seen only.
- [x] [STUB] R00a Add feature flags for PRO-only features with deny-by-default in CE builds.
- [x] [STUB] R00b Move ack/escalation/suppression/automation entry points behind PRO gates.
- [x] [STUB] R00c Remove non-essential CE dashboard controls and metrics from default CE UI.
- [x] [STUB] R00d Keep CE focused on wallboard display-only operations and basic health checks.
- [x] [STUB] R00e Add CI guardrails to fail CE builds when PRO-gated routes/components leak.

## Phase 1 - Data Foundation and Topology

### Epic 1 - Inventory Graph and Device Identity (F01-F05)
- [x] [STUB] R01 Define canonical schema for `device_identity`, `interface`, `neighbor`, `hardware_profile`, `source_observation`.
- [x] [STUB] R02 Create DB migrations and backfill scripts from current device cache/API store.
- [x] [STUB] R03 Implement identity stitching engine (fingerprint by MAC/serial/hostname/site hints).
- [x] [STUB] R04 Add drift fingerprint generator and change-detection snapshots.
- [x] [STUB] R05 Build ingestion mappers for interface stats and LLDP/CDP neighbor facts.
- [x] [STUB] R06 Expose APIs for identity merges, drift history, interface stats, and lifecycle score.
- [x] [STUB] R06a Expose read APIs for inventory schema, identities, observations, and drift snapshots.
- [x] [STUB] R07 Add UI panels for merged identity, interface breakdown, and drift badges.
- [x] [STUB] R08 Add tests: merge correctness, false-merge guardrails, migration rollback safety.
- [x] [STUB] R08a Add migration rollback safety coverage for schema upgrades and persistence replay.

### Epic 2 - Topology and Path Intelligence (F06-F10)
- [x] [STUB] R09 Build topology graph service from stitched inventory and neighbor links.
- [x] [STUB] R10 Implement topology API (`/topology/nodes`, `/topology/edges`, `/topology/health`).
- [x] [STUB] R11 Add map renderer with link-health heatmap coloring and stale-link indicators.
- [x] [STUB] R12 Add path trace engine and endpoint for selected source/target devices.
- [ ] R13 Add WAN SLA computation jobs (latency/loss/availability windows).
- [ ] R14 Add HA pair watcher and failover-state eventing.
- [ ] R15 Add topology QA fixtures and synthetic network datasets.
- [ ] R16 Add operational docs for graph rebuild, compaction, and troubleshooting.

## Phase 2 - Telemetry Intelligence

### Epic 3 - Telemetry Reliability Layer (F11-F15)
- [ ] R17 Implement retention policy engine with hot/warm/cold partitions.
- [ ] R18 Add per-device-class polling/sampling governor and queue priorities.
- [ ] R19 Add telemetry gap detector and missing-signal incident generation.
- [ ] R20 Add clock skew checks at ingest and normalize timestamps with source confidence.
- [ ] R21 Compute source quality scorecards (freshness, completeness, error rate).
- [ ] R22 Add API and UI views for data-quality and ingestion health.
- [ ] R23 Add load tests for retention compaction and sampling controls.
- [ ] R24 Add runbooks for skew recovery and source degradation events.

### Epic 4 - Smart Alert Signal Processing (F16-F20)
- [ ] R25 Build dynamic baseline model per metric/device role/site.
- [ ] R26 Add anomaly windows for day-of-week/hour-of-day behavior.
- [ ] R27 Add alert confidence score pipeline and visible confidence badges.
- [ ] R28 Add impact-radius estimator using topology dependencies.
- [ ] R29 Add storm shield summarizer for large simultaneous event bursts.
- [ ] R30 Add per-feature toggle flags to keep CE and PRO boundaries explicit.
- [ ] R31 Add comparison dashboard: raw alerts vs summarized alerts.
- [ ] R32 Add unit/integration tests for false positive/negative behavior.

## Phase 3 - Incident Operations and Wallboard UX

### Epic 5 - Incident Execution and Knowledge (F21-F25)
- [ ] R33 Implement incident commander mode with ownership and command timeline.
- [ ] R34 Build shift handoff generator with unresolved incidents and key deltas.
- [ ] R35 Add export pipeline for incident timeline to Markdown and PDF.
- [ ] R36 Add root-cause hypothesis panel with confidence and evidence links.
- [ ] R37 Build playbook checklist runner with step state and completion tracking.
- [ ] R38 Add audit events for checklist actions and commander handoffs.
- [ ] R39 Add API endpoints and UI views for incident workspace mode.
- [ ] R40 Add docs/templates for handoff format and post-incident review.

### Epic 6 - Dense Wallboard Upgrades (F26-F30)
- [ ] R41 Add micro-sparkline rendering pipeline for key card metrics.
- [ ] R42 Add quick-actions menu on cards (ack, silence, pin, diagnostics).
- [ ] R43 Add pin/focus board mode with keyboard shortcuts and persistence.
- [ ] R44 Add scene rotation scheduler for NOC wall cycling views.
- [ ] R45 Add accessibility profile system and WCAG-focused theme tests.
- [ ] R46 Add kiosk performance profiling to hold 60fps with large card counts.
- [ ] R47 Add visual regression suite for wallboard layouts.
- [ ] R48 Add operator docs for scene presets and accessibility tuning.

## Phase 4 - Automation and Ecosystem

### Epic 7 - Guided Remediation and Simulation (F31-F35)
- [ ] R49 Implement remediation action registry with dry-run support.
- [ ] R50 Build approval queue and dual-control workflow for risky actions.
- [ ] R51 Add drift-fix suggestion engine mapped to known config patterns.
- [ ] R52 Add scheduled health audit jobs and report artifacts.
- [ ] R53 Add change-window recommender using historical load curves.
- [ ] R54 Build telemetry replay sandbox (time-shifted event playback).
- [ ] R55 Add plugin hooks for remediation adapters and simulation injectors.
- [ ] R56 Add safety tests: rollback behavior, permission checks, replay isolation.

### Epic 8 - Integrations and Data Exchange (F36-F40)
- [ ] R57 Implement signed webhook templating and retry/dead-letter queue.
- [ ] R58 Publish inbound adapter SDK (schema contracts + validator CLI).
- [ ] R59 Build ChatOps command gateway with scoped command permissions.
- [ ] R60 Add GraphQL read layer for dashboard and analytics consumers.
- [ ] R61 Add object-storage export workers with partitioned datasets.
- [ ] R62 Add integration test harness (webhook, SDK adapter, ChatOps, GraphQL).
- [ ] R63 Add API versioning/deprecation policy docs.
- [ ] R64 Add sample integration packs for internal QA tenants.
- [ ] R64a Publish multi-vendor NMS connector contract (`discover`, `inventory`, `metrics`, `events`, `auth`).
- [ ] R64b Implement connector registry with per-vendor adapter loading and health state.
- [ ] R64c Ship first non-UISP NMS adapter and compatibility test matrix.
- [ ] R64d Define rolling goal and tracking for broad NMS API coverage.

## Phase 5 - Trust, Operability, and Commercial Readiness

### Epic 9 - Security and Trust Fabric (F41-F45)
- [ ] R65 Add per-account IP allowlist enforcement and bypass recovery flow.
- [ ] R66 Add session anomaly detector (geo/IP/device fingerprint changes).
- [ ] R67 Integrate secrets vault provider abstraction and rotation scheduler.
- [ ] R68 Build forensic bundle generator (logs, incident timeline, config snapshots).
- [ ] R69 Add tamper-evident hash chain for security-relevant events.
- [ ] R70 Add admin tooling for security policy simulation and dry-run rollout.
- [ ] R71 Add penetration test checklist and hardening verification pipeline.
- [ ] R72 Add compliance documentation and control mapping index.

### Epic 10 - Packaging, Pricing, and Reliability (F46-F50)
- [ ] R73 Implement usage metering pipeline (device-hours, ingestion volume, retention class).
- [ ] R74 Add plan-limit policy engine and in-app enforcement banners.
- [ ] R75 Build tenant sizing advisor from usage and performance telemetry.
- [ ] R76 Build orchestrated zero-downtime upgrade workflow for hosted clusters.
- [ ] R77 Publish reliability scorecard API and status computation service.
- [x] [STUB] R78 Add CI release matrix for CE images, SBOM, signatures, and provenance.
- [ ] R79 Execute production-grade soak/load/failover drills and capture SLO metrics.
- [ ] R80 Finalize launch gate checklist: docs, support runbooks, legal, billing ops.

## Implementation Rules
- [ ] Keep CE feature work isolated from PRO code paths using explicit feature flags.
- [ ] Add migration scripts for every schema change with rollback instructions.
- [ ] Require test coverage and docs updates before marking any run complete.
- [ ] Treat unknown/private logic as PRO by default.
- [ ] Verify each phase with `build -> run -> smoke test` before starting next phase.
