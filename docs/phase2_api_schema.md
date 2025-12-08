# Phase 2 Item 2 - API Schema (v1 Draft)

## Versioning
- `/api/v1` base; include `X-Request-ID` in requests/responses; errors structured as `{ code, message, request_id, details }`.

## Auth
- OIDC login endpoints; token refresh; PAT creation/management; session cookie for browser with CSRF token.

## Inventory
- `GET /sites`, `GET /sites/{id}`, `POST/PUT/PATCH /sites`.
- `GET /devices`, filters (site, role, status, tags); `GET /devices/{id}`; `PATCH /devices/{id}` for metadata/tags.
- `GET /topology` for dependency graph.

## Incidents & Alerts
- `GET /incidents` (filters: status, role, site, severity, suppressed); `GET /incidents/{id}`.
- `POST /incidents/{id}/ack` (duration, reason); `POST /incidents/{id}/clear`; `POST /maintenance`.
- `GET /alerts` (filter by incident/device/site); `GET /alerts/{id}` for delivery status and errors.

## Metrics/History
- `GET /metrics/devices/{id}` (time range, rollups); `GET /metrics/sites/{id}/aggregate`.
- `GET /health/poller` (lag, last fetch, error state) for banners.

## Notifications
- `POST /push/register` (device/browser token); `DELETE /push/register/{id}`.
- `POST /notifications/test` (per user or system) returns verbose status.

## Admin/Config
- `GET /config` (feature flags, banner messages, versions).
- `POST /users`, `GET /users`, `PATCH /users/{id}`; `GET /audit` for audit logs with request IDs.

## Error Handling
- Consistent error codes (e.g., `auth_invalid`, `not_found`, `validation_failed`, `rate_limited`, `backend_unavailable`).
- Response includes `request_id`, human-readable `message`, optional `detail`, and hint for operator banner.
