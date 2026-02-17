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
- [ ] Finalize CE vs Pro written contract for each module (API, UI, agent, mobile, notifications).
- [ ] Lock namespace/package naming (`nocwall_ce`, `nocwall_pro`, agent IDs, telemetry schema).

### Epic 2 - Secret Hygiene and Publish Safety
- [x] Remove hardcoded secrets from `docker-compose.yml`.
- [x] Add `.env.example` safe defaults.
- [ ] Add automated secret scanning in CI (trufflehog/gitleaks).
- [ ] Add CE release gate script for blocked terms (`billing`, `license`, `rbac`, `sso`, etc.).

## Phase 2 - Telemetry Ingestion and Device Truth

### Epic 3 - Vendor API Ingestion (UISP-first)
- [ ] Build UISP connector worker with polling cadence and retry policy.
- [ ] Normalize vendor payloads into canonical `device/site/link/event` models.
- [ ] Add backfill cursoring and idempotent ingestion.
- [ ] [STUB] Multi-vendor connector abstraction scaffold.

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
- [x] Dashboard display controls implemented with browser persistence.
- [x] Agent + telemetry ingest API stubs implemented.
- [x] Compose secrets replaced with placeholders.
- [ ] Validate full run in environment with Go + PHP toolchains installed.
