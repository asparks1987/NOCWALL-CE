# NOCWALL-CE Burndown

Chronological execution checklist to move from current transitional state to fully functional, distributable, and sellable hosted NOCWALL.

Legend:
- `[ ]` pending
- `[x]` completed
- `[STUB]` implemented as placeholder, not production-complete

## Phase 1 - Product Baseline and Safe Core

### Epic 1 - Identity, Positioning, and CE Scope Freeze
- [x] Rename UI surface to NOCWALL-CE in active dashboard.
- [x] Align README to hosted-first product direction (`nocwall.com` provisioning model).
- [x] [STUB] Add local CE account creation + login flow (JSON-backed users store for demo use).
- [x] [STUB] Add account-scoped UISP token import path in dashboard header.
- [ ] Finalize CE vs Pro written contract for each module (API, UI, agent, mobile, notifications).
- [ ] Lock namespace/package naming (`nocwall_ce`, `nocwall_pro`, agent IDs, telemetry schema).

### Epic 2 - Secret Hygiene and Publish Safety
- [x] Remove hardcoded secrets from `docker-compose.yml`.
- [x] Add `.env.example` safe defaults.
- [x] Add automated secret scanning in CI (gitleaks action + CE guardrails workflow).
- [x] Add CE release gate script for blocked terms (`billing`, `license`, `rbac`, `sso`, etc.).

## Phase 2 - Telemetry Ingestion and Device Truth

### Epic 3 - Vendor API Ingestion (UISP-first)
- [x] [STUB] Build UISP connector worker with polling cadence and retry policy.
- [x] [STUB] Normalize vendor payloads into canonical `device/site/link/event` models (device/event subset in CE).
- [x] [STUB] Add backfill cursoring and idempotent ingestion (cursor + dedupe window).
- [x] [STUB] Multi-vendor connector abstraction scaffold.

### Epic 4 - Agent Framework and Secure Enrollment
- [x] [STUB] Add API stubs for `/agents/register` and `/telemetry/ingest`.
- [ ] Define signed enrollment flow (bootstrap token, rotation, revocation).
- [ ] Implement Linux/SBC agent daemon with discovery + heartbeat + telemetry push.
- [ ] Implement fleet health page (agent online/offline/version drift).

## Phase 3 - Alerting and Incident Operations

### Epic 5 - Core Alert Engine
- [x] [STUB] Wire `device_down` style ingest to incident creation in API preview.
- [ ] Add sustained conditions, flap detection, and latency/jitter thresholds.
- [ ] Add dedup keys and incident lifecycle state machine.
- [ ] Persist alert rules and per-site overrides.

### Epic 6 - Incident Workflow and Operator Controls
- [x] Existing dashboard ack/clear/simulate controls operational in legacy UI.
- [ ] Build canonical incident API with comments/timeline/audit.
- [ ] Add runbook links and action buttons per incident type.
- [ ] [STUB] Assignment/escalation placeholder endpoints (kept private/Pro).

## Phase 4 - Wallboard UX and Client Surfaces

### Epic 7 - Dense Wallboard UX Refinement
- [x] Add configurable display controls (density + metric visibility toggles).
- [ ] Add per-operator saved dashboard layouts/presets.
- [ ] Add advanced card packing (priority rows, collapsible secondary stats).
- [ ] Add wallboard kiosk reliability features (auto-reconnect, stale-data banners).

### Epic 8 - Mobile Companion and Push Pipeline
- [x] Android scaffold and push token registration path exists.
- [ ] Implement authenticated mobile sessions with account-scoped API access.
- [ ] Build incident feed + ack actions in native mobile UI.
- [ ] [STUB] APNS/FCM routing and notification preference center.

## Phase 5 - Hosted SaaS Readiness and Commercialization

### Epic 9 - Multi-Tenant Hosted Platform (Private/Pro Boundary)
- [ ] [STUB] Keep CE endpoints simple; move tenant/RBAC/billing logic to private repos.
- [ ] Implement org/user/workspace isolation in hosted control plane.
- [ ] Implement provisioning flow from `nocwall.com` signup to active tenant.
- [ ] Implement audit/compliance data retention policy and tooling.

### Epic 10 - Distribution, Operations, and Go-to-Market
- [ ] Publish versioned CE artifacts and reproducible builds.
- [ ] Finalize private Pro plugin/server repos and deployment automation.
- [ ] Execute staging soak tests, load tests, backup/restore drills.
- [ ] Launch readiness checklist: legal/billing/support/onboarding docs.

## Current Pass Notes
- [x] Added `/sources/uisp/poll` + `/sources/uisp/status` and optional background UISP polling cadence.
- [x] Local `docker-compose` now builds API and web images from source by default.
- [x] Dashboard display controls implemented with browser persistence.
- [x] Agent + telemetry ingest API stubs implemented.
- [x] Compose secrets replaced with placeholders.
- [x] Validate core run/build paths via Dockerized Go/PHP checks and API smoke tests.
- [x] Verified `docker-compose build api uisp-noc` and `docker-compose up` smoke path (health + event ingest + web 200).
- [x] Manual UISP poll smoke tested via Docker (/sources/uisp/poll, /sources/uisp/status).
- [x] Implemented `index.php` multi-user auth migration (`cache/users.json`) with legacy `cache/auth.json` fallback.
- [x] Added Create Account UI + login session user context.
- [x] Added per-account UISP token save/status endpoints and dashboard action (`UISP Token` button).

