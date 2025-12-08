# Phase 1 Item 4 - Environments & Hosting Model

## Environments
- **Local/dev**: Docker Compose with Caddy, API, poller, notifier, web SPA, Gotify (optional), Postgres, Redis. Hot reload for API/web. Verbose logging enabled; error banner shows raw codes/messages and request IDs.
- **Staging**: Mirrors production topology with TLS, external OIDC, and test FCM project. Feature flags on; verbose error banner enabled with correlation IDs and links to logs. Synthetic monitors for poller/alerts.
- **Production**: Hardened config, rate limits, alerting on poller lag/queue depth/notification failures. Verbose error banner remains but with user-safe messaging and correlation IDs; full details in logs/trace viewers.

## Hosting Model
- Containerized services deployable to Kubernetes (Helm chart) or Compose for smaller installs. Caddy fronts API + SPA + optional Gotify.
- Data stores: managed or self-hosted Postgres/Timescale + Redis.
- Secrets: env files for dev; secret store (K8s Secrets/manager) for stage/prod; rotate keys/tokens regularly.

## Observability & Debugging (banner focus)
- Error banner spec: display error type, code, request/correlation ID, time, and suggested action; link to “copy diagnostics”.
- Clients (web/Android) show connectivity status (API, WS, push), last sync time, and retry/backoff status.
- Backend emits structured logs with request IDs; traces sampled in staging/prod; metrics for poller latency, queue depth, notification success/failure rates.

## Networking/TLS
- Caddy handles TLS (LE/ACME) and reverse proxy; internal service mesh not required initially.
- Staging uses staging ACME and test domains; production uses validated domains with automatic renewals.

## Access & RBAC
- OIDC across environments; local fallback auth for dev. Roles: admin/operator/viewer/wallboard.
- Separate client IDs/secrets per environment; scoped tokens; device tokens per environment for push.

## Release Channels
- Web/API: main -> staging -> prod with blue/green or rolling; feature flags for risky changes.
- Android: internal/stage track via Play/MDM; production track after verification; crash reporting enabled on all tracks.
