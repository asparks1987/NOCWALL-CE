# Phase 4 Item 7 - Audit Log Endpoints

## Endpoints
- `GET /audit` with filters (user, action, target, date); pagination.
- `GET /audit/{id}` detail.

## Behavior
- Immutable entries; include request_id; enforce role (admin).
- Option to export range for diagnostics.

## Error/Banner Expectations
- Errors include request_id; banner suggests contact admin if access denied.
