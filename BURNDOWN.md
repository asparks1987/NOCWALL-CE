# ⚙️ Codex Execution Instructions

You are acting as an autonomous senior software engineer.

When instructed to “follow the directions at the top of this file”, you must follow the rules below EXACTLY.

---

## 0. Definition of Done (DoD)
A task is only “done” when:
- The implementation is complete AND
- Any required tests are added/updated AND
- The relevant docs are updated (if user-facing or developer-facing) AND
- The checklist item is marked [x] with brief notes AND
- No known build/test/runtime failures remain (or failures are explicitly documented as pre-existing)

If any DoD element cannot be met, do NOT mark the task done; document what’s missing and why.

---

## 1. Read & Plan
- Read this entire file before making changes.
- Identify all unchecked tasks.
- Group tasks by dependency and execution order.
- Determine which tasks can be completed immediately with available context.
- Produce a short plan (bullets) before editing code.

---

## 2. Execution Order
Work in this priority order unless explicitly overridden:

1. Fix broken builds, tests, or runtime errors.
2. Implement missing core functionality.
3. Improve correctness and data integrity.
4. Add tests for newly implemented features.
5. Improve documentation and developer experience.
6. Refactor only when it directly improves reliability or clarity.

Do NOT start optional or cosmetic tasks until functional tasks are complete.

---

## 3. Task Processing Rules
For each task, in top-to-bottom order unless dependencies require reordering:

- If it can be completed fully → implement it.
- If it can be partially completed → implement what is possible and document what remains.
- If it cannot be completed → explain precisely why and what is required.

### Stop Rule (No Guessing Past Blockers)
If you are blocked by missing requirements, unclear behavior, or missing credentials/access:
- Stop work on that task
- Document the blocker clearly
- Move to the next unblocked task (if any)

Do not invent APIs, endpoints, schema, or business rules.

---

## 4. Change Scope & Standards
All work must:
- Follow existing project conventions.
- Include type hints where applicable.
- Include docstrings/comments for non-obvious logic.
- Avoid breaking existing functionality.
- Prefer small, incremental changes.

### Refactor Policy
- No drive-by refactors.
- Only refactor code you touched, and only if it reduces bugs or clarifies behavior.
- If a refactor is non-trivial, split it into a separate checklist task.

### Secrets/Security
- Do not expose secrets or credentials.
- Do not log sensitive values.
- Do not weaken auth, CORS, CSRF, or encryption behavior.

---

## 5. Testing & Validation
After implementing tasks:
- Add or update tests where relevant.
- Run tests/build if possible; otherwise simulate by reasoning and note what would be run.
- Fix failures caused by new changes.
- Do not leave known failing tests.

### Evidence Required
For each task you touch, include:
- Files changed (paths)
- Commands run (or commands that SHOULD be run)
- Any relevant output/expected output

If testing is impossible, explain exactly why.

---

## 6. Documentation Updates
When implementing features:
- Update relevant docs.
- Add usage examples where helpful.
- Note operational impacts if any.

Documentation must reflect actual behavior.

---

## 7. Checklist Maintenance
After completing work:
- Mark completed items as done.
- Add brief implementation notes under each completed item.
- Add new tasks if gaps are discovered.
- Do not delete incomplete tasks.

Use this format:

- [x] Task description
  - Notes: what was implemented
  - Files: path1, path2
  - Tests: what ran / what to run

---

## 8. Progress Reporting
At the end of each execution session, provide:
- Summary of completed tasks
- Remaining high-priority items
- Blockers or risks
- Recommended next steps

Be concise and factual.

---

## 9. Autonomy Rules
- Do not ask questions unless blocked.
- Make reasonable technical decisions independently.
- Prefer shipping working solutions over perfect designs.
- Optimize for reliability and maintainability.

---

## 10. Safety & Scope
- Do not implement automated trading, financial advice logic, or unsafe operations unless explicitly authorized.
- Do not add tracking/telemetry unless explicitly requested.
- Respect security and compliance constraints.

---

Follow these rules strictly.

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
- [x] Per-account subscription licensing with CE/PRO feature entitlement checks
- [x] Stripe billing wired through official `stripe-php` SDK (checkout, portal, webhook verify)
- [x] Dockerized local stack
- [x] Beta auth hardening: CSRF protection, session fixation mitigation, secure session cookies, and login throttling  
  - Notes: Added CSRF token validation for login/register/logout and mutating AJAX routes, session cookie hardening (`HttpOnly` + `SameSite=Lax` + HTTPS-secure), session ID regeneration on login/register, and file-backed login lockout controls.
- [x] Stripe-managed trial + payment-link tier checkout flow  
  - Notes: Removed server-managed signup trial assignment; billing now supports Stripe payment-link tiers (env-configured) so trial windows and plan flows are owned by Stripe checkout/webhooks.
- [ ] Planned product split change: move non-minimal CE capabilities behind PRO gates.

## 25 New CE Adoption Features (Strictly Non-PRO)

These features are intentionally limited to wallboard usability, reliability, and onboarding.
They avoid PRO-only domains (team workflows, correlation, automation, enterprise controls, deep analytics).

1. `[x]` CEF01 `[CE]` First-run setup wizard (add source + test + start dashboard)
2. `[x]` CEF02 `[CE]` Demo data toggle from UI for instant wallboard preview  
   - Notes: Added per-account demo mode state, dashboard toggle button, settings toggle, and backend demo device feed wiring in `?ajax=devices`.
3. `[x]` CEF03 `[CE]` Source connectivity diagnostics panel (DNS/TLS/API reachability)  
   - Notes: Extended source probes with DNS/TLS/API checks, added `?ajax=sources_diagnostics`, and shipped a settings diagnostics panel with per-source status details.
4. `[x]` CEF04 `[CE]` Manual "Poll Now" button per source
5. `[x]` CEF05 `[CE]` Source status strip (last poll time, success/fail, latency)
6. `[x]` CEF06 `[CE]` Global search box for device name/MAC/hostname
7. `[x]` CEF07 `[CE]` Quick filters (all/offline/online by tab)
8. `[x]` CEF08 `[CE]` Sort controls (status, name, last-seen)
9. `[x]` CEF09 `[CE]` Group-by mode (role/site) for card layout
10. `[x]` CEF10 `[CE]` Saved default tab per account
11. `[x]` CEF11 `[CE]` Saved refresh interval preset per account
12. `[x]` CEF12 `[CE]` Kiosk mode hotkey + URL flag (hide controls/chrome)
13. `[x]` CEF13 `[CE]` Keyboard shortcuts overlay (`?` help modal)
14. `[x]` CEF14 `[CE]` Dashboard legend panel (status colors/icons meanings)
15. `[x]` CEF15 `[CE]` Card status-change highlight (recently changed online/offline)
16. `[x]` CEF16 `[CE]` "Last updated" stale-data warning banner
17. `[x]` CEF17 `[CE]` API degraded banner with retry backoff indicator
18. `[x]` CEF18 `[CE]` Read-only fallback rendering from last known cache snapshot
19. `[x]` CEF19 `[CE]` Local browser notification option for new offline events  
   - Notes: Added dashboard header toggle with account-synced preference, permission handling, and desktop notifications on live online->offline transitions.
20. `[x]` CEF20 `[CE]` Optional soft chime mode (lower-volume alert profile)  
   - Notes: Added account-synced alert sound profile toggle (`default`/`soft`) and applied lower playback volume to siren alert execution.
21. `[x]` CEF21 `[CE]` Theme presets (classic/high-contrast/light) with persistence  
   - Notes: Added account-synced theme preset setting with dashboard toggle and CSS theme classes for classic, high-contrast, and light modes.
22. `[x]` CEF22 `[CE]` Font scaling presets for distant wall displays  
   - Notes: Added account-synced font scaling preset (`normal`/`large`/`xlarge`) with dashboard toggle and applied scaling classes.
23. `[x]` CEF23 `[CE]` Card print/export to PNG snapshot for NOC reports  
   - Notes: Added dashboard header `Export PNG` action that captures current wallboard view and downloads a timestamped PNG report snapshot.
24. `[x]` CEF24 `[CE]` Import/export dashboard preferences JSON  
   - Notes: Added Account Settings export/import workflow for preferences JSON (dashboard settings, siren prefs, and card order) using existing prefs APIs.
25. `[x]` CEF25 `[CE]` In-app "What's new" changelog modal per release  
   - Notes: Added backend release-notes payload and dashboard modal with per-release seen tracking to surface “What’s New” updates in-app.

## Phase CE-A - CE Adoption and Wallboard Polish

### Epic CE-A1 - CE Feature Pack 1 (CEF01-CEF10)
- [x] RCE01 Implement first-run wizard with source save/test handoff.
- [x] RCE02 Add demo data UI toggle and API wiring.  
  - Notes: Implemented account-scoped demo mode APIs (`demo_mode_get/set`), wallboard demo data payload generation, and live UI toggles on dashboard/settings pages.
- [x] RCE03 Build source diagnostics panel and health probes.  
  - Notes: Upgraded `probe_uisp_source` for DNS/TLS/API diagnostics and added settings UI/actions to run and render diagnostics across active sources.
- [x] RCE04 Add per-source "Poll Now" actions and status strip.
- [x] RCE05 Add search + quick filters across tabs.
- [x] RCE06 Add sort modes and account-persisted defaults.
- [x] RCE07 Implement group-by layout mode (role/site).
- [x] RCE08 Add default-tab persistence.

### Epic CE-A2 - CE Feature Pack 2 (CEF11-CEF18)
- [x] RCE09 Add refresh interval presets and persistence.
- [x] RCE10 Add kiosk mode (hotkey + query flag).
- [x] RCE11 Add keyboard shortcuts help modal.
- [x] RCE12 Add dashboard legend panel.
- [x] RCE13 Add recent state-change highlights on cards.
- [x] RCE14 Add stale-data warning banner.
- [x] RCE15 Add API degraded/retry backoff banner.
- [x] RCE16 Add read-only cached snapshot fallback path.

### Epic CE-A3 - CE Feature Pack 3 (CEF19-CEF25)
- [x] RCE17 Add browser notifications option for offline events.  
  - Notes: Implemented `browser_notifications` preference in dashboard settings, UI toggle button, and transition-driven Notification API dispatch for new offline events.
- [x] RCE18 Add soft chime alert profile option.  
  - Notes: Implemented `alert_sound_profile` dashboard setting, dashboard header toggle, and soft-profile volume application during alert playback.
- [x] RCE19 Add theme presets and persistence.  
  - Notes: Implemented `theme_preset` dashboard setting, cycle toggle in header, and persistent class-based theme application.
- [x] RCE20 Add wall-distance font scaling presets.  
  - Notes: Implemented `font_scale_preset` dashboard setting, cycle toggle in header, and scaling via font-scale body classes.
- [x] RCE21 Add dashboard PNG snapshot export.  
  - Notes: Implemented client-side PNG export via `html2canvas` lazy-load, with active-tab filename tagging and in-app status notices.
- [x] RCE22 Add settings JSON import/export.  
  - Notes: Implemented JSON backup export and validated import handlers in Account Settings, including support for wrapped and raw preferences payloads.
- [x] RCE23 Add in-app release changelog modal.  
  - Notes: Implemented `?ajax=whats_new`, header button/modal rendering release notes, and local per-release seen persistence.
- [x] RCE24 Add CE-only tests and docs for all CE feature-pack flows.  
  - Notes: Added `scripts/ce-feature-pack-smoke.sh` for CE endpoint smoke coverage and `docs/ce_feature_pack_test_matrix.md` manual validation matrix across CEF01-CEF25.

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
- [x] [STUB] R00b Keep alert ack/siren-silence in CE; keep escalation/suppression/automation behind PRO gates.
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
- [x] R13 Add WAN SLA computation jobs (latency/loss/availability windows).
  - Notes: Added PRO `?ajax=wan_sla` endpoint that computes 1h/24h/7d SLA windows from poll metrics, including availability, loss approximation, and latency p95.
- [x] R14 Add HA pair watcher and failover-state eventing.
  - Notes: Added HA pair inference + state watcher in the API store, emitted failover/recovery transition events, exposed `/topology/ha/pairs` + `/topology/ha/events`, wired PRO `?ajax=topology_ha`, and rendered HA pair/event panels in the topology tab.
- [x] R15 Add topology QA fixtures and synthetic network datasets.
  - Notes: Added reusable topology fixture datasets under `api/testdata/topology` and automated fixture-driven graph/path tests in `api/store_test.go`.
- [x] R16 Add operational docs for graph rebuild, compaction, and troubleshooting.
  - Notes: Added `docs/topology_operations.md` runbook with rebuild flow, compaction guidance, validation checklist, and troubleshooting scenarios.

## Phase 2 - Telemetry Intelligence

### Epic 3 - Telemetry Reliability Layer (F11-F15)
- [x] R17 Implement retention policy engine with hot/warm/cold partitions.
  - Notes: Added tiered source-observation retention compaction (hot/warm/cold windows), persisted last-run retention summary in store state, and exposed a PRO telemetry retention API endpoint for diagnostics.
- [x] R18 Add per-device-class polling/sampling governor and queue priorities.
  - Notes: Added normalized per-device-class governor rules with queue priorities, sample-interval enforcement in telemetry ingest, source-event priority ordering, drop accounting, and telemetry governor diagnostics at `GET /telemetry/governor`.
- [x] R19 Add telemetry gap detector and missing-signal incident generation.
  - Notes: Added class-threshold-based telemetry gap detection with `telemetry_gap` incident create/resolve flow, poller gap-evaluation hooks, and regression tests for incident lifecycle behavior.
- [x] R20 Add clock skew checks at ingest and normalize timestamps with source confidence.
  - Notes: Added ingest timestamp normalization (`observed_at`/`observed_at_ms`) with skew checks, correction rules, confidence scoring, and persisted timestamp metadata on telemetry samples and source observations.
- [x] R21 Compute source quality scorecards (freshness, completeness, error rate).
  - Notes: Added per-source quality aggregation (freshness, completeness, poll error rate, skew metrics, warning signals) and generated source scorecards with overall health classification.
- [x] R22 Add API and UI views for data-quality and ingestion health.
  - Notes: Added `/telemetry/quality` and `/telemetry/ingestion/health` API views, app proxy `?ajax=telemetry_quality`, and topology-tab UI panels for source quality scorecards and ingestion health.
- [x] R23 Add load tests for retention compaction and sampling controls.
  - Notes: Added benchmark load tests for retention compaction and sampling-governed ingest in `api/store_benchmark_test.go`.
- [x] R24 Add runbooks for skew recovery and source degradation events.
  - Notes: Added telemetry skew/degradation operations runbook in `docs/telemetry_skew_recovery.md`.

### Epic 4 - Smart Alert Signal Processing (F16-F20)
- [x] R25 Build dynamic baseline model per metric/device role/site.
  - Notes: Added API baseline aggregation across telemetry tiers with per role+site metric baselines (`latency_ms`, `availability_pct`) including dynamic bounds (`mean ± sigma`) and summary endpoint `GET /telemetry/baselines`.
- [x] R26 Add anomaly windows for day-of-week/hour-of-day behavior.
  - Notes: Added per role+site day/hour anomaly windows in baseline reports and exposed those windows in the topology telemetry UI for operator visibility.
- [x] R27 Add alert confidence score pipeline and visible confidence badges.  
  - Notes: Added telemetry alert-intelligence scoring pipeline with per-incident confidence score/level/reasons and surfaced confidence badges in the topology UI.
- [x] R28 Add impact-radius estimator using topology dependencies.  
  - Notes: Added topology-graph-based impact radius estimation per active alert (managed reach, reach %, scope classification).
- [x] R29 Add storm shield summarizer for large simultaneous event bursts.  
  - Notes: Added burst grouping by alert type/source/site with thresholded storm summaries and summarized-count compaction output.
- [x] R30 Add per-feature toggle flags to keep CE and PRO boundaries explicit.  
  - Notes: Added explicit feature flags for alert confidence, impact radius, storm shield, and alert comparison in API/mobile flags and UI feature gating.
- [x] R31 Add comparison dashboard: raw alerts vs summarized alerts.  
  - Notes: Added raw-vs-summarized alert comparison metrics in the alert intelligence API and topology “Storm Shield Summary” UI section.
- [x] R32 Add unit/integration tests for false positive/negative behavior.  
  - Notes: Added store tests validating confidence scoring, impact estimation, storm summarization reduction, and low-confidence handling for conflicting signal cases.

## Phase 3 - Incident Operations and Wallboard UX

### Epic 5 - Incident Execution and Knowledge (F21-F25)
- [x] R33 Implement incident commander mode with ownership and command timeline.
  - Notes: Extended incidents with commander ownership fields and persistent command timeline entries; added timeline event emission for incident open/ack/resolve and commander handoff/note actions.
  - Files: `api/models.go`, `api/store.go`, `api/store_test.go`
  - Tests: `node --check assets/app.js`; `go test ./...` in `api/` (run in WSL/Linux shell with Go installed)
- [x] R34 Build shift handoff generator with unresolved incidents and key deltas.
  - Notes: Added shift handoff generation and history in API store with active/unassigned counts plus delta summaries (new incidents, resolved incidents, commander changes) relative to prior handoff; exposed API/PHP bridge actions and topology UI handoff generator/history panel.
  - Files: `api/models.go`, `api/store.go`, `api/main.go`, `app.php`, `assets/app.js`, `api/store_test.go`
  - Tests: `node --check assets/app.js`; `docker run --rm -v ${PWD}:/work -w /work/api golang:1.24 go test ./...`; `docker run --rm -v ${PWD}:/work -w /work php:8.2-cli php -l app.php`
- [ ] R35 Add export pipeline for incident timeline to Markdown and PDF.
- [ ] R36 Add root-cause hypothesis panel with confidence and evidence links.
- [ ] R37 Build playbook checklist runner with step state and completion tracking.
- [x] R38 Add audit events for checklist actions and commander handoffs.
  - Notes: Added persisted incident audit event stream with commander handoff/acknowledge/timeline-note events and checklist audit ingestion; exposed API/PHP bridge endpoints plus topology UI audit feed.
  - Files: `api/models.go`, `api/store.go`, `api/main.go`, `app.php`, `assets/app.js`, `api/store_test.go`, `docs/incident_commander_workspace.md`
  - Tests: `node --check assets/app.js`; `docker run --rm -v ${PWD}:/work -w /work/api golang:1.24 go test ./...`; `docker run --rm -v ${PWD}:/work -w /work php:8.2-cli php -l app.php`
- [x] R39 Add API endpoints and UI views for incident workspace mode.
  - Notes: Added incident workspace APIs (`/incidents/workspace`, `/incidents/:id/commander`, `/incidents/:id/timeline`), PHP AJAX bridge/proxy handlers, CSRF protection for commander/timeline mutations, and topology-tab incident commander/timeline UI controls.
  - Files: `api/main.go`, `app.php`, `assets/app.js`, `docs/incident_commander_workspace.md`
  - Tests: `node --check assets/app.js`; `php -l app.php` (run where PHP CLI is installed); `go test ./...` in `api/` (run in WSL/Linux shell with Go installed)
- [x] R40 Add docs/templates for handoff format and post-incident review.
  - Notes: Added operator-ready shift handoff and post-incident review templates, and updated incident commander workspace docs with handoff/audit endpoint usage.
  - Files: `docs/shift_handoff_template.md`, `docs/post_incident_review_template.md`, `docs/incident_commander_workspace.md`
  - Tests: Documentation update (no runtime test required)

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
