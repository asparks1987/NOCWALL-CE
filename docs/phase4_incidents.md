# Phase 4 Item 4 - Incident/Alert Endpoints

## Endpoints
- `GET /incidents` (filters: status, severity, role, site, suppressed, acknowledged).
- `GET /incidents/{id}` with timeline.
- `POST /incidents/{id}/ack` (duration/reason), `POST /incidents/{id}/clear`.
- `POST /maintenance` (scope, window), `GET /maintenance`.
- `GET /alerts` and `GET /alerts/{id}` for delivery status/errors.

## Behavior
- Apply dependency and maintenance suppression; include suppression info in responses.
- Record acknowledgements with user/time; update incident timeline.

## Error/Banner Expectations
- Errors include request_id and suppression/validation details for banners.
