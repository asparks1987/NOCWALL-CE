# Phase 2 Item 7 - WebSocket/SSE Channels

## Channels
- `device_updates`: pushes device_state changes (status/latency/metrics summary).
- `incident_updates`: pushes incident create/update/resolve events.
- `alert_status`: pushes notification delivery status/errors for current user (privacy-scoped).
- `health`: poller lag, queue depth, notification failures for banners.

## Protocol
- Prefer WebSockets; SSE fallback if needed. Auth via access token; include request_id in server messages.

## Reliability
- Heartbeats/pings; reconnect with backoff.
- Resume support via last-seen cursor to avoid gaps.
- Backpressure handling; cap message sizes; compression where available.

## Error/Banner Expectations
- On disconnect/error, UI shows banner with reason and retry status; include last request_id and next retry time.
