# Phase 1 Item 3 - Proposed Core Stack

Draft selections for core technologies. Adjust if constraints or preferences differ.

## Backend API
- Language/runtime: TypeScript on Node (fast prototyping, rich ecosystem) or Go (static, lean binaries). **Default**: TypeScript + NestJS/Express.
- Auth: OIDC/OAuth2 (JWT access + refresh), session cookies for browser, PAT/service tokens for automation.
- Validation: Zod/DTO validation at boundaries.
- Documentation: OpenAPI first; generated client for web/Android.

## Data Stores
- Primary DB: Postgres (optionally Timescale extension) for metrics/incidents/history.
- Cache/queue: Redis (for caching, locks, WebSocket/SSE fan-out, job queues).
- Message bus: Redis Streams or NATS for poller -> alert engine -> notifier events. **Default**: Redis Streams to reduce surface area.

## Poller & Workers
- Language: Go or Node (same as API) for UISP polling and alert evaluation workers. **Default**: TypeScript worker sharing types with API.

## Front-End (Web SPA)
- Framework: React + Vite (or Next.js if SSR needed). **Default**: React + Vite.
- State/data: React Query (SWR) + WS/SSE subscriptions.
- Charts/maps: Chart.js or Recharts; Leaflet/Mapbox for topology/map view.
- Auth: OIDC PKCE; service worker for push and offline cache.

## Android (Native)
- Language: Kotlin.
- Networking: Retrofit/OkHttp + Kotlinx Serialization; WebSockets/SSE for live updates.
- Push: FCM with actionable notifications.
- Storage: Room/Datastore for offline cache and preferences.

## Gateway/Proxy/TLS
- Caddy remains TLS terminator and reverse proxy in front of API + SPA + notifier endpoints; keeps optional Gotify host.

## CI/CD & Tooling
- CI: GitHub Actions (or GitLab CI) for lint/test/build docker images; Android build/signing pipeline.
- IaC/packaging: Docker/Compose for dev; Helm (or K8s manifests) for staging/prod.
- Observability: OpenTelemetry (traces/metrics/logs) exported to Prometheus/Grafana stack.

## Defaults Summary (if no overrides)
- API + workers: TypeScript (NestJS/Express) + Redis Streams + Postgres/Timescale.
- Web: React + Vite + React Query + WS/SSE.
- Android: Kotlin + Retrofit + FCM.
- Infra: Caddy + Docker/Compose for dev; Helm for prod; OpenTelemetry + Prometheus/Grafana.
