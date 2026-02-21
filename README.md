# NOCWALL-CE

NOCWALL-CE is the community/open foundation for the NOCWALL platform.

Target product direction:
- Users sign up at `nocwall.com`.
- Each user gets a provisioned browser-based NOC workspace.
- Telemetry is collected from:
  - Vendor/NMS APIs (UISP first, expanding to broad NMS API coverage)
  - Local network agents (Linux/SBC daemon)
- The wallboard UI focuses on dense, glanceable, per-device status cards with alert-first behavior.
- CE/PRO split direction: CE is intentionally minimal wallboard functionality; advanced capabilities are PRO.

This repository currently contains a local development stack and transitional code while we move from legacy UISP-NOC to NOCWALL.

## Docker Hub Listing (Copy/Paste)

Repository:
- `predheadtx/nocwall`

Short description:
- NOCWALL-CE is an open-core network wallboard for dense, glanceable device monitoring with multi-vendor NMS + agent telemetry ingestion.

Long description:
- NOCWALL-CE is the Community Edition of NOCWALL, built for NOC wall displays and fast network visibility. It includes a browser dashboard, per-account NMS source configuration, account-synced dashboard preferences, basic alerting (offline/ack/siren), and a Go API with telemetry ingest and inventory foundations (identity stitching and drift snapshots). UISP is the first connector and additional NMS APIs are planned.
- Current images are intended for development/testing while the hosted `nocwall.com` control-plane model is being completed.

## Current State Snapshot (2026-02-19)

- Account login/signup is functional for local CE testing.
- UISP sources are saved per account and load across browsers after login.
- Dashboard settings (density/metrics/AP siren mode) are account-synced.
- API inventory foundation is live:
  - canonical schema endpoints
  - identity stitching (multi-source merge by key hints)
  - source observation history
  - drift fingerprint snapshots
- Current connector status:
  - UISP connector implemented first
  - multi-vendor NMS adapter expansion in roadmap
- Product split status:
  - current branch is transitional and still contains features that will be gated/moved to PRO
  - upcoming releases prioritize reducing CE to minimal wallboard scope

## CE vs PRO Contract (Updated)

CE (free) target:
- Wall-mounted dashboard view
- Basic device list/cards
- Online/offline state
- Very limited card fields (name, role, site, last seen)
- Basic account/session for accessing the wallboard

PRO target:
- Advanced telemetry analytics and baselines
- Incident workflow, timeline, and team operations
- Correlation/dedup/suppression/escalation logic
- Topology intelligence, drift analysis, remediation tooling
- Integrations, exports, automation, governance, and enterprise controls

## What Works Today

- Legacy dashboard (PHP + JS) with:
  - account creation + login (demo/local JSON-backed user store)
  - per-account subscription state and license-aware feature gating (CE vs PRO)
  - live device cards
  - flashing offline behavior
  - siren audio alerts
  - ack / clear ack
  - outage simulation
  - station ping history modal
  - per-account UISP source management (`Account Settings`) with multiple UISP URLs and tokens
- New dashboard display controls:
  - persistent card density (`Normal`, `Compact`)
  - metric toggles (CPU, RAM, Temp, Latency, Uptime, Outage)
  - per-account preference sync (saved server-side, loaded across browsers after login)
  - per-card siren toggle (`Siren: On/Off`) on Gateways, APs, and Routers/Switches (persisted per account)
  - per-tab siren toggles for Gateways, APs, and Routers/Switches (persisted per account; default is Gateway on, AP/Router off)
  - drag-and-drop card ordering per tab (Gateways/APs/Routers-Switches), persisted per account
  - inventory/drift badges on cards (from API inventory overview)
  - per-card Inventory modal with identity details, interface summary, neighbor links, lifecycle score, and drift snapshots
  - CE strict-profile defaults (minimal card contract) with PRO-only feature gating
  - account subscription controls in `Account Settings`:
    - start monthly PRO subscription
    - cancel/resume subscription
    - mobile and local-agent entitlement indicators
    - agent bootstrap config preview for licensed users
- Go API preview with in-memory/file-backed store:
  - `GET /health`
  - `POST /auth/login`
  - `GET /mobile/config`
  - `GET /devices`
  - `GET /incidents`
  - `POST /incidents/:id/ack`
  - `GET /metrics/devices/:id`
  - `POST /push/register`
  - `GET /agents` (stub)
  - `POST /agents/register` (stub)
  - `POST /telemetry/ingest` (stub)
  - `POST /events/ingest` (stub)
  - `POST /sources/uisp/poll` (stub)
  - `GET /sources/uisp/status` (stub)
  - `GET /inventory/schema` (stub)
  - `GET /inventory/identities` (stub)
  - `GET /inventory/observations` (stub)
  - `GET /inventory/drift` (stub)
  - `GET /inventory/interfaces` (stub)
  - `GET /inventory/neighbors` (stub)
  - `GET /inventory/lifecycle` (stub)
  - `POST /inventory/identities/merge` (stub)
  - `GET /topology/nodes` (stub)
  - `GET /topology/edges` (stub)
  - `GET /topology/health` (stub)
  - `GET /topology/path` (stub)
  - identity stitching from telemetry fields (`mac`, `serial`, `hostname`, source/device hints)
  - drift fingerprint snapshots per identity
  - interface/neighbor ingestion mappers from telemetry payloads (`interfaces`, `neighbors`)
  - topology graph synthesis from identity + neighbor link inventory
  - topology tab renderer (SVG map) with resolved/unresolved/stale link coloring and path trace controls
- Docker Compose builds API and web locally from source by default (no private image dependency).
- Docker Compose wiring uses safe env placeholders (no hardcoded real keys).

## What Is Stubbed / Not Fully Implemented

- True hosted multi-tenant SaaS control plane.
- Production auth/session model (JWT/refresh/RBAC/SSO).
- Production persistence for API services (currently demo-level store).
- Production-grade payment processing lifecycle (Stripe webhooks, retries, dispute handling, dunning automation).
- Agent PKI/enrollment, secure long-lived channels, and fleet lifecycle management.
- Correlation/dedup/suppression/escalation and enterprise routing.
- Production mobile backend workflows and push delivery orchestration.

## What's Coming Next (Active Burndown Focus)

- Phase 0 CE minimalization:
  - enforce strict CE card contract (online/offline + limited fields)
  - feature-flag and gate advanced UI/API flows as PRO-only
  - add CI checks to prevent PRO leakage into CE
- R05: ingest mappers for interface stats and LLDP/CDP neighbor facts.
- R06: expanded inventory APIs (identity merge operations, interface stats, lifecycle scoring paths).
- R07: UI panels for merged identity, interface breakdown, and drift badges. (implemented as dashboard Inventory modal + badges)
- R08: merge-safety and migration rollback tests. (implemented as store tests, including migration replay/duplicate-id repair coverage)
- R09/R10: topology graph service + topology API endpoints (`/topology/nodes`, `/topology/edges`, `/topology/health`) now available as stubs.
- R11/R12: topology map tab + path trace endpoint (`/topology/path`) implemented as stubs.
- Add connector adapters beyond UISP (progressive rollout by NMS family).

## Quick Start (Local Dev)

1. Copy environment defaults:

```bash
cp .env.example .env
```

2. Start stack:

```bash
docker-compose up -d --build
```

SBC/edge bootstrap (recommended for Raspberry Pi and similar devices):

```bash
chmod +x ./NOCWALL.sh
./NOCWALL.sh /install
./NOCWALL.sh /deploy
```

Deploy CE + API + PRO extension profile:

```bash
./NOCWALL.sh /deploy --suite --pro
```

One-file build command (with action overloads):

```bash
./buildmultiarch.sh /install
./buildmultiarch.sh /preflight
./buildmultiarch.sh /build
./buildmultiarch.sh /update
./buildmultiarch.sh --u predheadtx -i nocwall:latest
```

Preflight runs:
- `scripts/ce-release-gate.sh`
- shell syntax checks for `buildmultiarch.sh` and `NOCWALL.sh`
- JS syntax check for `assets/app.js` (when `node` is installed)
- `docker compose config` validation
- `go test ./...` in `api` (when `go` is installed)

Defaults used by `buildmultiarch.sh`:
- Docker Hub user: `predheadtx`
- Web image repository: `nocwall` (published as `predheadtx/nocwall`)
- API image repository: `nocwall-api` (published as `predheadtx/nocwall-api`)
- Tag: `latest`
- Platforms: `linux/amd64,linux/arm64,linux/arm/v7`
- Extra publish tag: `git-<shortsha>` (enabled by default)

Custom example:

```bash
./buildmultiarch.sh --u predheadtx -i nocwall:latest
./buildmultiarch.sh /build --user predheadtx --web-repo nocwall --api-repo nocwall-api --tag latest
```

GitHub Actions release publishing:
- Workflow: `.github/workflows/release-images.yml`
- Publishes multi-arch images for both:
  - `predheadtx/nocwall`
  - `predheadtx/nocwall-api`
- Requires repository secrets:
  - `DOCKERHUB_USERNAME`
  - `DOCKERHUB_TOKEN`

3. Open the dashboard:
- `http://localhost`
- Create an account from the login screen (or sign in with bootstrap `admin` / `admin`).
- After login, open `Account Settings` from the header and add one or more UISP sources (`base URL + API token`) for that user.


Optional UISP source polling env vars:
- `UISP_URL` and `UISP_TOKEN` (optional server fallback only)
- `UISP_DEVICES_PATH` (default `/nms/api/v2.1/devices`)
- `UISP_POLL_INTERVAL_SEC` (0 disables background polling)
- `UISP_POLL_RETRIES` (default `1`)

Optional inventory bridge vars (web -> API):
- `NOCWALL_API_URL` (default `http://api:8080`)
- `API_TOKEN` (if API auth is enabled)

Feature profile vars:
- `NOCWALL_FEATURE_PROFILE` (`ce` default, `pro` enables advanced UI/actions)
- `NOCWALL_PRO_FEATURES` (optional explicit `true/false` override)
- `NOCWALL_STRICT_CE` (optional explicit `true/false` override)

Billing + licensing vars:
- `NOCWALL_BILLING_MODE` (`stripe`, `demo`, `off`)
- `NOCWALL_BILLING_SELF_ACTIVATE` (`true/false`, should be `false` for Stripe production)
- `NOCWALL_PRO_MONTHLY_USD` (display price for monthly PRO)
- `NOCWALL_STRIPE_SECRET_KEY` (required for Stripe mode)
- `NOCWALL_STRIPE_WEBHOOK_SECRET` (required for Stripe webhook signature validation)
- `NOCWALL_STRIPE_PRICE_ID` (required for Stripe mode)
- `NOCWALL_STRIPE_SUCCESS_URL` (optional, defaults to settings page callback)
- `NOCWALL_STRIPE_CANCEL_URL` (optional, defaults to settings page callback)
- `NOCWALL_STRIPE_PORTAL_RETURN_URL` (optional, defaults to settings page callback)

Stripe integration implementation:
- Uses official `stripe/stripe-php` (Composer dependency) for checkout, portal sessions, webhook signature verification, and subscription retrieval.

Stripe webhook endpoint:
- `POST /?webhook=stripe`
- Configure this endpoint in Stripe and subscribe to:
  - `checkout.session.completed`
  - `customer.subscription.created`
  - `customer.subscription.updated`
  - `customer.subscription.deleted`
  - `invoice.payment_failed`
  - `invoice.paid`

CE release gate:

```bash
./scripts/ce-release-gate.sh
```

Note:
- `UISP_*` variables are current connector-specific settings.
- As additional NMS connectors are added, each will get equivalent connector-scoped config.
4. Open:
- Dashboard: `http://localhost` (or your Caddy endpoint)
- API: `http://localhost:8080`

## Account + UISP Sources Flow (curl)

Register:

```bash
curl -i -c cookies.txt -b cookies.txt -X POST "http://localhost/?action=register" \
  -d "username=tester1234&password=Password123&password_confirm=Password123"
```

Login:

```bash
curl -i -c cookies.txt -b cookies.txt -X POST "http://localhost/?action=login" \
  -d "username=tester1234&password=Password123"
```

Add a UISP source to the account:

```bash
curl -c cookies.txt -b cookies.txt -X POST "http://localhost/?ajax=sources_save" \
  -d "name=MainUISP&url=https://isp.unmsapp.com&token=demo_token_1234567890&enabled=1"
```

List configured UISP sources:

```bash
curl -c cookies.txt -b cookies.txt "http://localhost/?ajax=sources_list"
```

Check subscription + license status:

```bash
curl -c cookies.txt -b cookies.txt "http://localhost/?ajax=billing_status"
```

Activate monthly PRO subscription (demo mode):

```bash
curl -c cookies.txt -b cookies.txt -X POST "http://localhost/?ajax=billing_subscribe"
```

Stripe CLI webhook test (local):

```bash
stripe listen --forward-to "http://localhost/?webhook=stripe"
```

## API Smoke Tests

Health:

```bash
curl http://localhost:8080/health
```

Create a down event (incident should appear):

```bash
curl -X POST http://localhost:8080/events/ingest \
  -H "Content-Type: application/json" \
  -d '{"type":"device_down","device_id":"demo-1","site":"lab","message":"demo down"}'
```

Create/refresh an agent (stub):

```bash
curl -X POST http://localhost:8080/agents/register \
  -H "Content-Type: application/json" \
  -d '{"id":"agent-lab-1","name":"Lab SBC","site_id":"lab","version":"0.1.0","capabilities":["discovery","snmp"]}'
```

Ingest telemetry (stub):

```bash
curl -X POST http://localhost:8080/telemetry/ingest \
  -H "Content-Type: application/json" \
  -d '{"source":"agent","agent_id":"agent-lab-1","event_type":"device_up","device_id":"sw-lab-1","device":"Switch Lab 1","site_id":"lab","online":true,"latency_ms":2.1}'
```

Topology API quick checks:

```bash
curl "http://localhost:8080/topology/nodes?limit=25"
curl "http://localhost:8080/topology/edges?limit=25"
curl "http://localhost:8080/topology/health"
curl "http://localhost:8080/topology/path?source_node_id=ident:IDENT_A&target_node_id=ident:IDENT_B"
```


Run manual UISP poll (demo mode if no valid UISP creds):

```bash
curl -X POST http://localhost:8080/sources/uisp/poll -H "Content-Type: application/json" -d '{"demo":true,"limit":50}'
```
Guardrail scan before publishing CE:

```bash
bash scripts/ce-release-gate.sh
```
List incidents:

```bash
curl http://localhost:8080/incidents
```

Inventory endpoints:

```bash
curl http://localhost:8080/inventory/schema
curl http://localhost:8080/inventory/identities
curl "http://localhost:8080/inventory/observations?limit=20"
curl "http://localhost:8080/inventory/drift?limit=20"
curl "http://localhost:8080/inventory/interfaces?limit=20"
curl "http://localhost:8080/inventory/neighbors?limit=20"
curl "http://localhost:8080/inventory/lifecycle?limit=20"
```

Manual identity merge (stub):

```bash
curl -X POST http://localhost:8080/inventory/identities/merge \
  -H "Content-Type: application/json" \
  -d '{"primary_id":"ident-primary","secondary_ids":["ident-secondary-1","ident-secondary-2"]}'
```

## Documentation

Documentation is consolidated around two primary files:
- `README.md` (product state, runbook, test commands)
- `BURNDOWN.md` (chronological multi-phase execution plan)

Legacy phase docs remain under `docs/` as historical planning references and should be treated as secondary.

## Security / Publishing Guardrails

- Do not commit real keys/tokens/customer details.
- Keep `.env` private; commit `.env.example` only.
- Treat advanced workflows (multi-tenant/RBAC/escalation/billing/mobile orchestration) as Pro/private until explicitly split.

## Repo Status Notes

- `assets/app.js` was restored from git history because it was missing in this checkout and is required by `index.php`.
- This repo is in active migration from UISP-NOC naming/architecture to NOCWALL-CE and hosted-first operation.





