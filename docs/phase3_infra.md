# Phase 3 Item 1 - Multi-Service Layout (Compose/Helm)

## Services
- `api`: REST/WS API (Node/Go).
- `poller`: UISP poller worker.
- `alert-engine`: rule evaluation worker.
- `notifier`: notification router (Gotify/FCM/email/SMS/webhook).
- `web`: SPA static hosting (served via Caddy or CDN).
- `caddy`: TLS terminator/reverse proxy.
- `gotify` (optional): embedded server for notifications.
- `db`: Postgres/Timescale.
- `redis`: cache/queue/streams.

## Networking
- Internal network for services; Caddy exposes 80/443.
- Health endpoints for each service; Caddy routes `/api`, `/ws`, `/` (SPA), `/gotify` (optional).

## Compose/Helm
- Compose for dev: single-network, mounted volumes for code and db data.
- Helm chart: deployments, services, ingress, secrets, configmaps; health probes; autoscaling config.

## Error/Diagnostics
- Health endpoints return verbose status (uptime, last error message, request_id).
- Compose overrides enable verbose logging; Helm values toggle debug/verbose banners.
