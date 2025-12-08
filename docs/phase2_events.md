# Phase 2 Item 3 - Event Model (Poller -> Queue -> API)

## Event Types
- `device_state.changed`: device_id, site_id, role, status, latency, metrics snapshot, upstream_id, correlation_id.
- `metric.sampled`: device_id, metrics, timestamp, sample_id.
- `incident.triggered/resolved/updated`: incident_id, device_id, reason, suppression, correlation_id.
- `alert.delivery`: incident_id, channel, status, error (verbose), attempt, request_id.
- `topology.updated`: site_id, devices, links, source (UISP/manual), validation errors.

## Flow
- Poller fetches UISP, normalizes, emits `device_state.changed` + `metric.sampled` to Redis Streams.
- Alert engine consumes events, evaluates rules, emits `incident.*` and schedules notifications.
- Notifier consumes `incident.*` to send via channels, emitting `alert.delivery`.
- API exposes event history and current status; stores critical events in DB for auditing.

## Reliability & Debugging
- Each event carries `correlation_id` and `request_id` where applicable; retries with idempotency keys.
- Dead-letter streams for failed processing with verbose error payloads.
- Metrics on lag, processing time, and failure rates exported for banners/health endpoints.
