# Phase 2 Item 1 - Entity Model (Draft)

## Core Entities
- **Site**: id, name, location, timezone, tags, parent (optional).
- **Device**: id, name, role, site_id, upstream_device_id (for dependency), ip/identifiers, status, metadata (cpu/ram/temp).
- **Incident**: id, device_id, site_id, type (offline/latency/flap/etc.), severity, started_at, resolved_at, acknowledged_until, suppression_reason, correlation_id.
- **Alert**: id, incident_id, channel, delivered_at, status, response (for actionable notifications), error (verbose), request_id.
- **Acknowledgement**: id, incident_id, user_id, duration, expires_at, reason.
- **MaintenanceWindow**: id, scope (site/device/role), start/end, reason.
- **User**: id, name, email, role, auth_provider, status.
- **Token**: id, user_id, type (access/refresh/pat/device), expires_at, scopes.
- **NotificationEndpoint**: id, user_id, type (gotify/fcm/email/sms/webhook), token/address, env, last_error.
- **MetricSample**: id, device_id, site_id, timestamp, latency, cpu, ram, temp, loss, jitter, throughput, online.
- **AuditLog**: id, user_id, action, target_type/id, payload, created_at, request_id.

## Error/Diagnostics Fields
- Correlation/request IDs on incidents/alerts/logs.
- Error detail on notification endpoints and alerts for banner/debug views.

## Relationships/Indexes (high level)
- Device -> Site (many-to-one), Device -> Device (upstream).
- Incident -> Device/Site; Alert -> Incident; Acknowledgement -> Incident/User.
- Metrics indexed by device_id, timestamp; incidents by site/role/status.

## Notes
- Support multi-tenant if needed later via tenant_id on all entities.
- Keep verbose error info for failed notifications and poller/parity mismatches for operator banners.
