# Phase 5 Item 6 - Poller Health & Lag Metrics

## Health Endpoint
- `/health/poller`: returns status, last_success, last_error (message/code), lag seconds, queue backlog, request_id.
- Banners read this to show “Poller delayed” with reason.

## Metrics
- Prometheus metrics for poll_duration, poll_errors, lag, upstream_errors.
