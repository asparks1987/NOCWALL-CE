# NOCWALL-CE Burndown (Recreated)

Chronological multi-phase plan to move from the current state to a fully functional, distributable, and sellable NOCWALL platform.

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

## 50 Net-New Features (Not Previously Planned in BURNDOWN)

### Inventory and Topology
1. `[ ]` F01 `[CE]` Multi-source device identity stitching (same device merged across UISP + agent feeds)
2. `[ ]` F02 `[CE]` Configuration drift fingerprints per device
3. `[ ]` F03 `[CE]` Interface-level utilization and error metrics
4. `[ ]` F04 `[CE]` LLDP/CDP neighbor discovery inventory
5. `[ ]` F05 `[PRO]` Hardware lifecycle risk score (EOS/EOL awareness)
6. `[ ]` F06 `[CE]` Auto-generated network topology map
7. `[ ]` F07 `[PRO]` Path trace view (gateway to target device)
8. `[ ]` F08 `[CE]` Link-health heatmap overlays
9. `[ ]` F09 `[PRO]` WAN circuit SLA tracker
10. `[ ]` F10 `[PRO]` Redundancy/HA state monitor

### Telemetry and Signal Quality
11. `[ ]` F11 `[CE]` Multi-tier retention (hot/warm/cold data windows)
12. `[ ]` F12 `[CE]` Sampling-rate governor by device class
13. `[ ]` F13 `[CE]` Telemetry gap detector with missing-data alerts
14. `[ ]` F14 `[CE]` Agent/server clock-skew detector
15. `[ ]` F15 `[CE]` Source data quality scorecard
16. `[ ]` F16 `[CE]` Dynamic baseline thresholds per metric
17. `[ ]` F17 `[PRO]` Seasonal anomaly detection (hour/day pattern aware)
18. `[ ]` F18 `[PRO]` Alert confidence scoring
19. `[ ]` F19 `[PRO]` Impact-radius estimator
20. `[ ]` F20 `[PRO]` Alert storm shield with event summarization

### Incident and Operator Workflow
21. `[ ]` F21 `[PRO]` Incident commander mode
22. `[ ]` F22 `[PRO]` Shift handoff auto-briefs
23. `[ ]` F23 `[CE]` Post-incident timeline export (Markdown/PDF)
24. `[ ]` F24 `[PRO]` Root-cause hypothesis assistant panel
25. `[ ]` F25 `[CE]` Playbook checklist runner on incident pages

### Dashboard and Wallboard UX
26. `[ ]` F26 `[CE]` Micro-sparklines inside device cards
27. `[ ]` F27 `[CE]` Per-card quick actions menu
28. `[ ]` F28 `[CE]` Pin/focus mode for critical devices
29. `[ ]` F29 `[CE]` Wall rotation scenes (auto-cycle filtered views)
30. `[ ]` F30 `[CE]` Accessibility profiles (colorblind/high-contrast/large text)

### Automation and Integrations
31. `[ ]` F31 `[PRO]` Safe remediation actions with approval queue
32. `[ ]` F32 `[PRO]` Drift auto-remediation suggestions
33. `[ ]` F33 `[CE]` Scheduled health audits
34. `[ ]` F34 `[PRO]` Change-window suggestions from traffic patterns
35. `[ ]` F35 `[CE]` Telemetry replay sandbox for testing rules
36. `[ ]` F36 `[CE]` Signed outbound webhook templates
37. `[ ]` F37 `[CE]` Inbound event adapter SDK
38. `[ ]` F38 `[PRO]` ChatOps command interface
39. `[ ]` F39 `[CE]` GraphQL read API for BI tools
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
50. `[ ]` F50 `[CE]` Public reliability scorecard API

## Phase 1 - Data Foundation and Topology

### Epic 1 - Inventory Graph and Device Identity (F01-F05)
- [ ] R01 Define canonical schema for `device_identity`, `interface`, `neighbor`, `hardware_profile`, `source_observation`.
- [ ] R02 Create DB migrations and backfill scripts from current device cache/API store.
- [ ] R03 Implement identity stitching engine (fingerprint by MAC/serial/hostname/site hints).
- [ ] R04 Add drift fingerprint generator and change-detection snapshots.
- [ ] R05 Build ingestion mappers for interface stats and LLDP/CDP neighbor facts.
- [ ] R06 Expose APIs for identity merges, drift history, interface stats, and lifecycle score.
- [ ] R07 Add UI panels for merged identity, interface breakdown, and drift badges.
- [ ] R08 Add tests: merge correctness, false-merge guardrails, migration rollback safety.

### Epic 2 - Topology and Path Intelligence (F06-F10)
- [ ] R09 Build topology graph service from stitched inventory and neighbor links.
- [ ] R10 Implement topology API (`/topology/nodes`, `/topology/edges`, `/topology/health`).
- [ ] R11 Add map renderer with link-health heatmap coloring and stale-link indicators.
- [ ] R12 Add path trace engine and endpoint for selected source/target devices.
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
- [ ] R78 Add CI release matrix for CE images, SBOM, signatures, and provenance.
- [ ] R79 Execute production-grade soak/load/failover drills and capture SLO metrics.
- [ ] R80 Finalize launch gate checklist: docs, support runbooks, legal, billing ops.

## Implementation Rules
- [ ] Keep CE feature work isolated from PRO code paths using explicit feature flags.
- [ ] Add migration scripts for every schema change with rollback instructions.
- [ ] Require test coverage and docs updates before marking any run complete.
- [ ] Treat unknown/private logic as PRO by default.
- [ ] Verify each phase with `build -> run -> smoke test` before starting next phase.
