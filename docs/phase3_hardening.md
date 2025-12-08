# Phase 3 Item 2 - Images, Healthchecks, Probes

## Images
- Build services as non-root; set USER in Dockerfiles; minimal base images (alpine/distroless where possible).
- Multi-stage builds; copy only runtime artifacts.

## Health/Readiness
- Liveness/readiness probes for api/poller/notifier (HTTP `/healthz`, `/ready`).
- Poller/notifier health includes last successful cycle time and last error message.
- Caddy health via admin or simple endpoint; db/redis monitored externally.

## Error/Diagnostics
- Health endpoints include request_id and last error summary (sanitized) for banner display.
- Probes configurable per environment; aggressive in staging to catch regressions.
