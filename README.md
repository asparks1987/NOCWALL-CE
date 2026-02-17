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
  - live device cards
  - flashing offline behavior
  - siren audio alerts
  - ack / clear ack
  - outage simulation
  - station ping history modal
- New dashboard display controls:
  - persistent card density (`Normal`, `Compact`)
  - metric toggles (CPU, RAM, Temp, Latency, Uptime, Outage)
  - local storage persistence per browser
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
- Docker Compose wiring with safe env placeholders (no hardcoded real keys).

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
docker-compose up -d
```

3. Open:
- Dashboard: `http://localhost` (or your Caddy endpoint)
- API: `http://localhost:8080`

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
