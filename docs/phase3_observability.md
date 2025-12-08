# Phase 3 Item 4 - Observability (OTel/Prometheus) & Dashboards

## Tracing
- OpenTelemetry SDK in api/poller/notifier; propagate request_id; sample in staging/prod.
- Caddy tracing optional; trace context passed to backend.

## Metrics
- Prometheus scrape targets for all services; key metrics: poller lag, queue depth, notification success/fail, API latency/error rates, WS disconnects, banner-visible errors.

## Dashboards
- Grafana dashboards for service health, poller performance, notification delivery, DB/Redis health.
- Admin banner pulls from health endpoints and key metrics (e.g., poller lag > threshold).

## Logging
- Structured JSON logs with level, request_id, correlation_id, user_id (if available); redaction for secrets.
- Central log aggregation; alerts on error spikes.

## Error/Diagnostics
- Health endpoints include last_error_message and time; banner uses this for verbose display.
