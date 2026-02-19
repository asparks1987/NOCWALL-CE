# NOCWALL-CE

NOCWALL-CE is the community/open foundation for the NOCWALL platform.

Target product direction:
- Users sign up at `nocwall.com`.
- Each user gets a provisioned browser-based NOC workspace.
- Telemetry is collected from:
  - Vendor APIs (example: UISP API keys)
  - Local network agents (Linux/SBC daemon)
- The wallboard UI focuses on dense, glanceable, per-device status cards with alert-first behavior.

This repository currently contains a local development stack and transitional code while we move from legacy UISP-NOC to NOCWALL.

## What Works Today

- Legacy dashboard (PHP + JS) with:
  - account creation + login (demo/local JSON-backed user store)
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
  - AP card siren toggle (`Siren: On/Off`) persisted per account
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
- Docker Compose builds API and web locally from source by default (no private image dependency).
- Docker Compose wiring uses safe env placeholders (no hardcoded real keys).

## What Is Stubbed / Not Fully Implemented

- True hosted multi-tenant SaaS control plane.
- Production auth/session model (JWT/refresh/RBAC/SSO).
- Production persistence for API services (currently demo-level store).
- Agent PKI/enrollment, secure long-lived channels, and fleet lifecycle management.
- Correlation/dedup/suppression/escalation and enterprise routing.
- Production mobile backend workflows and push delivery orchestration.

## Quick Start (Local Dev)

1. Copy environment defaults:

```bash
cp .env.example .env
```

2. Start stack:

```bash
docker-compose up -d --build
```

One-file build command (with action overloads):

```bash
./buildmultiarch.sh /install
./buildmultiarch.sh /build
./buildmultiarch.sh /update
```

Defaults used by `buildmultiarch.sh /build`:
- Docker Hub user: `predheadtx`
- Image name: `NOCWALL` (published as repository `predheadtx/nocwall`)
- Tag: `latest`
- Platforms: `linux/amd64,linux/arm64,linux/arm/v7`

Custom example:

```bash
./buildmultiarch.sh /build --user predheadtx --name NOCWALL --tag latest
```

3. Open the dashboard:
- `http://localhost`
- Create an account from the login screen (or sign in with bootstrap `admin` / `admin`).
- After login, open `Account Settings` from the header and add one or more UISP sources (`base URL + API token`) for that user.


Optional UISP source polling env vars:
- `UISP_URL` and `UISP_TOKEN` (optional server fallback only)
- `UISP_DEVICES_PATH` (default `/nms/api/v2.1/devices`)
- `UISP_POLL_INTERVAL_SEC` (0 disables background polling)
- `UISP_POLL_RETRIES` (default `1`)
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





